package front

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gbootcms/internal/app"
	"gbootcms/internal/auth"
	"gbootcms/internal/domain"
	entmember "gbootcms/internal/ent/member"
	"gbootcms/internal/ent/membercomment"
	"gbootcms/internal/fsperm"
	"gbootcms/internal/store"
	"gbootcms/internal/urlx"
)

func member(a *app.App, w http.ResponseWriter, r *http.Request, acode string, h hit) bool {
	if h.Kind == "comment" {
		switch h.Extra {
		case "del":
			delMyComment(a, w, r)
			return true
		case "my":
			if memberOf(a, r)["id"] == "" {
				http.Redirect(w, r, urlx.Home(Rules(a.DB), "member/login", 0), http.StatusFound)
				return true
			}
			return false
		default:
			if r.Method == http.MethodPost || h.Extra == "add" {
				postComment(a, w, r, acode)
				return true
			}
			http.Redirect(w, r, "/", http.StatusFound)
			return true
		}
	}
	action := strings.ToLower(h.Extra)
	switch action {
	case "logout":
		auth.SetCookie(w, r, &http.Cookie{Name: auth.FrontCookie(), Value: "", MaxAge: -1})
		http.Redirect(w, r, "/", http.StatusFound)
		return true
	case "sendemail":
		if r.Method == http.MethodPost {
			doSendEmail(a, w, r)
			return true
		}
		http.Redirect(w, r, "/", http.StatusFound)
		return true
	case "register":
		if r.Method == http.MethodPost {
			doRegister(a, w, r)
			return true
		}
		s := ensureFront(a, w, r)
		s.Values["sendemail"] = "1"
		_ = a.Front.Save(s)
		return false
	case "retrieve":
		if r.Method == http.MethodPost {
			doRetrieve(a, w, r)
			return true
		}
		s := ensureFront(a, w, r)
		s.Values["sendemail"] = "1"
		_ = a.Front.Save(s)
		return false
	case "ucenter":
		if memberOf(a, r)["id"] == "" {
			http.Redirect(w, r, urlx.Home(Rules(a.DB), "member/login", 0), http.StatusFound)
			return true
		}
		return false
	case "umodify":
		if memberOf(a, r)["id"] == "" {
			http.Redirect(w, r, urlx.Home(Rules(a.DB), "member/login", 0), http.StatusFound)
			return true
		}
		if r.Method == http.MethodPost {
			doUmodify(a, w, r)
			return true
		}
		return false
	case "isregister":
		if r.Method == http.MethodPost {
			doIsRegister(a, w, r)
			return true
		}
		http.Redirect(w, r, "/", http.StatusFound)
		return true
	case "upload":
		if r.Method == http.MethodPost {
			doMemberUpload(a, w, r)
			return true
		}
		http.Redirect(w, r, "/", http.StatusFound)
		return true
	default:
		if r.Method == http.MethodPost {
			doLogin(a, w, r)
			return true
		}
		return false
	}
}

func delMyComment(a *app.App, w http.ResponseWriter, r *http.Request) {
	m := memberOf(a, r)
	if m["id"] == "" {
		JSON(w, 0, "请先登录")
		return
	}
	id := domain.Atoi(QueryVal(r, "id"), 0)
	if id == 0 {
		id = domain.Atoi(r.FormValue("id"), 0)
	}
	if id == 0 {
		JSON(w, 0, "传递参数有误")
		return
	}
	n, err := a.DB.Ent.MemberComment.Delete().Where(
		membercomment.IDEQ(id), membercomment.UIDEQ(domain.Atoi(m["id"], 0)),
	).Exec(bg)
	if err != nil || n == 0 {
		JSON(w, 0, "删除失败")
		return
	}
	JSON(w, 1, "删除成功")
}

func doLogin(a *app.App, w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	if a.DB.Conf("login_status") != "1" {
		JSON(w, 0, "系统已经关闭登录功能！")
		return
	}
	ip := remoteIP(r)
	if !auth.LoginAllowed("m:"+ip, 8, 900) {
		JSON(w, 0, "尝试过多，请15分钟后再试")
		return
	}
	if a.DB.Conf("login_check_code") != "0" && !checkCode(a, w, r, true) {
		auth.LoginFail("m:" + ip)
		return
	}
	user := strings.TrimSpace(r.FormValue("username"))
	one, err := a.DB.Ent.Member.Query().Where(entmember.UsernameEQ(user), entmember.StatusEQ("1")).First(bg)
	if err != nil {
		one, err = a.DB.Ent.Member.Query().Where(entmember.UseremailEQ(user), entmember.StatusEQ("1")).First(bg)
	}
	if err != nil {
		one, err = a.DB.Ent.Member.Query().Where(entmember.UsermobileEQ(user), entmember.StatusEQ("1")).First(bg)
	}
	if err != nil || one.Password != auth.Encrypt(r.FormValue("password")) {
		if s := loadFront(a, r); s != nil && s.Values["code"] != "" {
			delete(s.Values, "code")
			_ = a.Front.Save(s)
		}
		auth.LoginFail("m:" + ip)
		JSON(w, 0, "账号或密码错误")
		return
	}
	auth.LoginOK("m:" + ip)
	// rotate session id after auth (anti fixation)
	s, _ := a.Front.New()
	s.Values["uid"] = strconv.Itoa(one.ID)
	_ = a.Front.Save(s)
	auth.SessionCookie(w, r, auth.FrontCookie(), s.ID)
	back := safeMemberBack(r.FormValue("backurl"), QueryVal(r, "backurl"))
	if back == "" {
		back = urlx.Home(Rules(a.DB), "member/ucenter", 0)
	}
	if wantsJSON(r) {
		JSONTourl(w, 1, "登录成功", back)
		return
	}
	http.Redirect(w, r, back, http.StatusFound)
}

func safeMemberBack(vals ...string) string {
	for _, u := range vals {
		u = strings.TrimSpace(u)
		if u == "" || strings.ContainsAny(u, "\r\n\\") {
			continue
		}
		// relative path only (blocks //evil.com and scheme-relative)
		if strings.HasPrefix(u, "/") && !strings.HasPrefix(u, "//") {
			return u
		}
	}
	return ""
}

func doRegister(a *app.App, w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	if a.DB.Conf("register_status") != "1" {
		JSON(w, 0, "注册已关闭")
		return
	}
	if a.DB.Conf("register_check_code") != "0" && !checkCode(a, w, r, true) {
		return
	}
	user := strings.TrimSpace(r.FormValue("username"))
	pass := r.FormValue("password")
	if user == "" || pass == "" {
		JSON(w, 0, "请填写账号和密码")
		return
	}
	if _, hasR := r.Form["rpassword"]; hasR && pass != r.FormValue("rpassword") {
		JSON(w, 0, "确认密码不正确！")
		return
	}
	if a.DB.Conf("password_strong") != "0" && !auth.Strong(pass) {
		JSON(w, 0, auth.StrongMsg())
		return
	}
	useremail := strings.TrimSpace(r.FormValue("useremail"))
	usermobile := strings.TrimSpace(r.FormValue("usermobile"))
	switch a.DB.Conf("register_type") {
	case "2":
		useremail = user
		if !emailRe.MatchString(useremail) {
			JSON(w, 0, "账号格式不正确，请输入正确的邮箱账号！")
			return
		}
		if n, _ := a.DB.Ent.Member.Query().Where(entmember.UseremailEQ(useremail)).Count(bg); n > 0 {
			JSON(w, 0, "您输入的邮箱已被注册！")
			return
		}
	case "3":
		usermobile = user
		if !mobileRe.MatchString(usermobile) {
			JSON(w, 0, "账号格式不正确，请输入正确的手机号码！")
			return
		}
		if n, _ := a.DB.Ent.Member.Query().Where(entmember.UsermobileEQ(usermobile)).Count(bg); n > 0 {
			JSON(w, 0, "您输入的手机号码已被注册！")
			return
		}
	default:
		if !accountRe.MatchString(user) {
			JSON(w, 0, "用户账号含有不允许的特殊字符！")
			return
		}
		if n, _ := a.DB.Ent.Member.Query().Where(entmember.UsernameEQ(user)).Count(bg); n > 0 {
			JSON(w, 0, "您输入的账号已被注册！")
			return
		}
	}
	now := domain.Now()
	nick := orForm(r, "nickname", user)
	created, err := a.DB.Ent.Member.Create().
		SetUcode("U" + now).SetUsername(user).SetUseremail(useremail).SetUsermobile(usermobile).
		SetNickname(nick).SetPassword(auth.Encrypt(pass)).SetHeadpic("").SetStatus("1").SetActivation("1").SetGid("1").
		SetWxid("").SetQqid("").SetWbid("").SetScore(0).SetRegisterTime(now).SetLoginCount(0).
		SetLastLoginIP("").SetLastLoginTime("").SetSex(r.FormValue("sex")).SetBirthday("").SetQq("").
		Save(bg)
	if err != nil {
		JSON(w, 0, "注册失败")
		return
	}
	diy, err := collectMemberDIY(a, r)
	if err != nil {
		JSON(w, 0, err.Error())
		return
	}
	if err := applyMemberDIY(a, created.ID, diy); err != nil {
		JSON(w, 0, "注册失败")
		return
	}
	s := ensureFront(a, w, r)
	s.Values["uid"] = strconv.Itoa(created.ID)
	_ = a.Front.Save(s)
	JSONTourl(w, 1, "注册成功", urlx.Home(Rules(a.DB), "member/ucenter", 0))
}

func doRetrieve(a *app.App, w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	s := ensureFront(a, w, r)
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	code := strings.ToLower(strings.TrimSpace(r.FormValue("checkcode")))
	user := strings.TrimSpace(r.FormValue("username"))
	pass := r.FormValue("password")
	if code == "" || user == "" || pass == "" || email == "" {
		JSON(w, 0, "请完整填写")
		return
	}
	if s.Values["retrieve_checkcode"] == "" || !strings.EqualFold(code, s.Values["retrieve_checkcode"]) {
		JSON(w, 0, "验证码错误！")
		return
	}
	if s.Values["retrieve_email"] != "" && s.Values["retrieve_email"] != email {
		JSON(w, 0, "与发送邮箱不一致！")
		return
	}
	one, err := a.DB.Ent.Member.Query().Where(entmember.UsernameEQ(user)).First(bg)
	if err != nil {
		JSON(w, 0, "该用户不存在！")
		return
	}
	if one.Useremail == "" || !strings.EqualFold(one.Useremail, email) {
		JSON(w, 0, "与注册邮箱不匹配，请联系管理员！")
		return
	}
	if a.DB.Conf("password_strong") != "0" && !auth.Strong(pass) {
		JSON(w, 0, auth.StrongMsg())
		return
	}
	_, err = a.DB.Ent.Member.UpdateOneID(one.ID).SetPassword(auth.Encrypt(pass)).Save(bg)
	if err != nil {
		JSON(w, 0, "修改失败！")
		return
	}
	delete(s.Values, "retrieve_checkcode")
	delete(s.Values, "retrieve_email")
	_ = a.Front.Save(s)
	JSONTourl(w, 1, "修改成功！", urlx.Home(Rules(a.DB), "member/login", 0))
}

func doUmodify(a *app.App, w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	m := memberOf(a, r)
	if m["id"] == "" {
		http.Redirect(w, r, urlx.Home(Rules(a.DB), "member/login", 0), http.StatusFound)
		return
	}
	id := domain.Atoi(m["id"], 0)
	one, err := a.DB.Ent.Member.Get(bg, id)
	if err != nil {
		alertFront(w, "用户不存在！")
		return
	}
	if r.FormValue("opassword") == "" {
		alertFront(w, "请输入当前密码！")
		return
	}
	if one.Password != auth.Encrypt(r.FormValue("opassword")) {
		alertFront(w, "您输入的当前密码不正确！")
		return
	}
	email := strings.TrimSpace(r.FormValue("useremail"))
	mobile := strings.TrimSpace(r.FormValue("usermobile"))
	if email != "" {
		if !emailRe.MatchString(email) {
			alertFront(w, "邮箱格式不正确，请输入正确的邮箱账号！")
			return
		}
		if n, _ := a.DB.Ent.Member.Query().Where(entmember.UseremailEQ(email), entmember.IDNEQ(id)).Count(bg); n > 0 {
			alertFront(w, "您输入的邮箱已被注册！")
			return
		}
	}
	if mobile != "" {
		if !mobileRe.MatchString(mobile) {
			alertFront(w, "手机格式不正确，请输入正确的手机号码！")
			return
		}
		if n, _ := a.DB.Ent.Member.Query().Where(entmember.UsermobileEQ(mobile), entmember.IDNEQ(id)).Count(bg); n > 0 {
			alertFront(w, "您输入的手机号码已被注册！")
			return
		}
	}
	pass := r.FormValue("password")
	if pass != "" && pass != r.FormValue("rpassword") {
		alertFront(w, "确认密码不正确！")
		return
	}
	diy, err := collectMemberDIY(a, r)
	if err != nil {
		alertFront(w, err.Error())
		return
	}
	up := a.DB.Ent.Member.UpdateOneID(id).
		SetNickname(r.FormValue("nickname")).SetUseremail(email).SetUsermobile(mobile).
		SetHeadpic(r.FormValue("headpic")).SetSex(r.FormValue("sex")).
		SetBirthday(r.FormValue("birthday")).SetQq(r.FormValue("qq"))
	if pass != "" {
		if a.DB.Conf("password_strong") != "0" && !auth.Strong(pass) {
			JSON(w, 0, auth.StrongMsg())
			return
		}
		up = up.SetPassword(auth.Encrypt(pass))
	}
	if _, err := up.Save(bg); err != nil {
		alertFront(w, "资料修改失败！")
		return
	}
	if err := applyMemberDIY(a, id, diy); err != nil {
		alertFront(w, "资料修改失败！")
		return
	}
	http.Redirect(w, r, urlx.Home(Rules(a.DB), "member/umodify", 0), http.StatusFound)
}

var memberDIYBlocked = map[string]bool{
	"id": true, "ucode": true, "username": true, "password": true, "status": true, "gid": true,
	"wxid": true, "qqid": true, "wbid": true, "score": true, "register_time": true,
	"login_count": true, "last_login_ip": true, "last_login_time": true, "activation": true,
	"useremail": true, "usermobile": true, "nickname": true, "headpic": true,
}

func collectMemberDIY(a *app.App, r *http.Request) (map[string]any, error) {
	fields, _ := a.DB.Maps("SELECT name,required,description FROM ay_member_field WHERE status='1' ORDER BY sorting,id")
	out := map[string]any{}
	for _, f := range fields {
		name := f["name"]
		if !store.Ident(name) || memberDIYBlocked[name] {
			continue
		}
		v := strings.TrimSpace(r.FormValue(name))
		if vs := r.Form[name]; len(vs) > 1 {
			v = strings.Join(vs, ",")
		}
		if f["required"] == "1" && v == "" {
			desc := f["description"]
			if desc == "" {
				desc = name
			}
			return nil, fmt.Errorf("%s不能为空！", desc)
		}
		out[name] = v
	}
	return out, nil
}

func applyMemberDIY(a *app.App, id int, cols map[string]any) error {
	if len(cols) == 0 {
		return nil
	}
	var sets []string
	args := make([]any, 0, len(cols)+1)
	for k, v := range cols {
		if !store.Ident(k) || memberDIYBlocked[k] {
			continue
		}
		sets = append(sets, k+"=?")
		args = append(args, v)
	}
	if len(sets) == 0 {
		return nil
	}
	args = append(args, id)
	return a.DB.Exec("UPDATE ay_member SET "+strings.Join(sets, ",")+" WHERE id=?", args...)
}

func doMemberUpload(a *app.App, w http.ResponseWriter, r *http.Request) {
	if memberOf(a, r)["id"] == "" {
		JSON(w, 0, "请先登录！")
		return
	}
	_ = r.ParseMultipartForm(20 << 20)
	f, hdr, err := r.FormFile("upload")
	if err != nil {
		f, hdr, err = r.FormFile("file")
	}
	if err != nil {
		JSON(w, 0, "没有文件")
		return
	}
	defer f.Close()
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(hdr.Filename), "."))
	allow := a.DB.Conf("home_upload_ext")
	if allow == "" {
		allow = "jpg,jpeg,png,gif,webp"
	}
	ok := false
	for _, e := range strings.Split(allow, ",") {
		if strings.EqualFold(strings.TrimSpace(e), ext) {
			ok = true
			break
		}
	}
	if !ok {
		JSON(w, 0, "不允许的扩展名")
		return
	}
	day := time.Now().Format("20060102")
	dir := filepath.Join(a.Cfg.SiteRoot, "static", "upload", "image", day)
	if err := fsperm.MkdirAll(dir, fsperm.DirUpload); err != nil {
		JSON(w, 0, "保存失败")
		return
	}
	name := strconv.FormatInt(time.Now().UnixNano(), 10) + "." + ext
	dst := filepath.Join(dir, name)
	out, err := fsperm.Create(dst, fsperm.FileUpload)
	if err != nil {
		JSON(w, 0, "保存失败")
		return
	}
	if _, err := io.Copy(out, f); err != nil {
		_ = out.Close()
		JSON(w, 0, "保存失败")
		return
	}
	_ = out.Close()
	if a.DB.Conf("watermark_open") == "1" && (ext == "jpg" || ext == "jpeg" || ext == "png") {
		Watermark(dst, a.DB.Conf("watermark_text"))
	}
	JSON(w, 1, "/static/upload/image/"+day+"/"+name)
}

func doIsRegister(a *app.App, w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	user := strings.TrimSpace(r.FormValue("username"))
	if user == "" {
		JSON(w, 1, "账号不能为空！")
		return
	}
	typ := a.DB.Conf("register_type")
	switch typ {
	case "2":
		if !emailRe.MatchString(user) {
			JSON(w, 1, "账号格式不正确，请输入正确的邮箱账号！")
			return
		}
		if n, _ := a.DB.Ent.Member.Query().Where(entmember.UseremailEQ(user)).Count(bg); n > 0 {
			JSON(w, 1, "您输入的邮箱已被注册！")
			return
		}
		JSON(w, 0, "您输入的邮箱可以使用！")
	case "3":
		if !mobileRe.MatchString(user) {
			JSON(w, 1, "账号格式不正确，请输入正确的手机号码！")
			return
		}
		if n, _ := a.DB.Ent.Member.Query().Where(entmember.UsermobileEQ(user)).Count(bg); n > 0 {
			JSON(w, 1, "您输入的手机号码已被注册！")
			return
		}
		JSON(w, 0, "您输入的手机号码可以使用！")
	default:
		if !accountRe.MatchString(user) {
			JSON(w, 1, "用户账号含有不允许的特殊字符！")
			return
		}
		if n, _ := a.DB.Ent.Member.Query().Where(entmember.UsernameEQ(user)).Count(bg); n > 0 {
			JSON(w, 1, "您输入的账号已被注册！")
			return
		}
		JSON(w, 0, "您输入的账号可以使用！")
	}
}

func doSendEmail(a *app.App, w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	s := ensureFront(a, w, r)
	if s.Values["sendemail"] != "1" {
		JSON(w, 0, "非法提交发送邮件！")
		return
	}
	if last := domain.Atoi(s.Values["lastsend"], 0); last > 0 && int(time.Now().Unix())-last < 10 {
		JSON(w, 0, "您提交太频繁了，请稍后再试！")
		return
	}
	to := strings.TrimSpace(r.FormValue("to"))
	if to == "" || !emailRe.MatchString(to) {
		JSON(w, 0, "邮箱格式不正确，请输入正确的邮箱账号！")
		return
	}
	retrieve := r.FormValue("retrieve") == "1"
	if !retrieve && a.DB.Conf("register_check_code") != "2" {
		JSON(w, 0, "发送失败，后台配置非邮箱验证码模式！")
		return
	}
	code := fmt.Sprintf("%04d", 1000+int(time.Now().UnixNano()%9000))
	s.Values["lastsend"] = strconv.FormatInt(time.Now().Unix(), 10)
	if retrieve {
		s.Values["retrieve_checkcode"] = strings.ToLower(code)
		s.Values["retrieve_email"] = strings.ToLower(to)
	} else {
		s.Values["code"] = strings.ToLower(code)
	}
	_ = a.Front.Save(s)
	subj := "【GbootCMS】您有新的验证码信息，请注意查收！"
	body := "您的验证码为：" + code
	if err := sendMailTo(a, to, subj, body); err != nil {
		JSON(w, 0, "发送失败，"+err.Error())
		return
	}
	JSON(w, 1, "发送成功！")
}

var (
	emailRe   = regexp.MustCompile(`^[\w]+@[\w.]+\.[a-zA-Z]+$`)
	mobileRe  = regexp.MustCompile(`^1[0-9]{10}$`)
	accountRe = regexp.MustCompile(`^[\w@.]+$`)
)

func alertFront(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte("<script>alert(" + strconv.Quote(msg) + ");history.back()</script>"))
}

func orForm(r *http.Request, name, def string) string {
	if v := strings.TrimSpace(r.FormValue(name)); v != "" {
		return v
	}
	return def
}

func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "json") || r.FormValue("ajax") == "1"
}

func JSONTourl(w http.ResponseWriter, code int, data any, tourl string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "data": data, "tourl": tourl})
}

func sendMailTo(a *app.App, to, subject, body string) error {
	host := a.DB.Conf("smtp_server")
	user := a.DB.Conf("smtp_username")
	pass := a.DB.Conf("smtp_password")
	if host == "" || user == "" {
		return fmt.Errorf("未配置SMTP")
	}
	port := a.DB.Conf("smtp_port")
	if port == "" {
		port = "25"
	}
	to = mailHeader(to)
	subject = mailHeader(subject)
	addr := host + ":" + port
	msg := "To: " + to + "\r\nSubject: " + subject + "\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n" + body
	var sa smtp.Auth
	if pass != "" {
		sa = smtp.PlainAuth("", user, pass, host)
	}
	return smtp.SendMail(addr, sa, user, []string{to}, []byte(msg))
}

// enrichMember adds template aliases used by {user:*} tags.
func enrichMember(a *app.App, m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	if m["id"] == "" {
		return m
	}
	m["registertime"] = m["register_time"]
	m["logincount"] = m["login_count"]
	m["lastlogintime"] = m["last_login_time"]
	m["lastloginip"] = long2ipMember(m["last_login_ip"])
	if m["headpic"] == "" {
		m["headpic"] = "/apps/admin/view/default/images/logo.png"
	}
	if m["gname"] == "" && m["gid"] != "" {
		rows, _ := a.DB.Maps("SELECT gname FROM ay_member_group WHERE id=?", m["gid"])
		if len(rows) > 0 {
			m["gname"] = rows[0]["gname"]
		}
	}
	return m
}

func long2ipMember(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || strings.Contains(v, ".") {
		return v
	}
	n, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		return v
	}
	return fmt.Sprintf("%d.%d.%d.%d", byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}
