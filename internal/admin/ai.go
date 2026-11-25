package admin

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"gbootcms/internal/app"
	"gbootcms/internal/domain"
	"gbootcms/internal/front"
)

var aiCallLog sync.Map // username -> []int64 unix

func aiCall(a *app.App, w http.ResponseWriter, r *http.Request, action string) {
	user := "uid:0"
	if sess := adminSess(a, r); sess != nil {
		user = or(sess.Values["username"], "uid:"+sess.Values["id"])
	}
	scene, task := action, "-"
	fail := func(msg string) {
		aiWriteLog(a, user, "error", scene, task, msg)
		front.JSON(w, 0, msg)
	}
	if r.Method != http.MethodPost {
		fail("请使用POST")
		return
	}
	if action != "ping" && a.DB.Conf("ai_enabled") != "1" {
		fail("AI 功能未启用")
		return
	}
	if !aiRateOK(a, user) {
		fail("调用过于频繁，请稍后重试")
		return
	}
	var messages []map[string]string
	switch action {
	case "ping":
		messages = []map[string]string{{"role": "user", "content": "你好，请回复连接成功。"}}
	case "generate":
		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			fail("标题不能为空")
			return
		}
		messages = []map[string]string{
			{"role": "system", "content": "你是中文写作助手。用 <p> 输出正文，不要 Markdown。"},
			{"role": "user", "content": "请围绕《" + title + "》写一篇短文，分 3 段，每段用 <p> 包裹。"},
		}
	case "rewrite":
		text := strings.TrimSpace(r.FormValue("text"))
		mode := r.FormValue("mode")
		task = or(mode, "-")
		prompt := map[string]string{
			"continue": "续写下面的文本，只返回新增部分：",
			"rewrite":  "改写下面的文本，只返回全文：",
			"shorten":  "压缩下面的文本，只返回全文：",
			"polish":   "润色下面的文本，只返回全文：",
		}[mode]
		if text == "" || prompt == "" {
			fail("请提供文本和处理方式")
			return
		}
		messages = []map[string]string{
			{"role": "system", "content": "你是中文编辑。用 <p> 输出，不要 Markdown。"},
			{"role": "user", "content": prompt + "\n" + text},
		}
	case "tdk":
		title := strings.TrimSpace(r.FormValue("title"))
		body := strings.TrimSpace(r.FormValue("content"))
		if title == "" && body == "" {
			fail("请提供文章标题或正文")
			return
		}
		messages = []map[string]string{
			{"role": "system", "content": "只输出 JSON，字段为 title、description、keywords。"},
			{"role": "user", "content": "标题：" + title + "\n正文：" + body},
		}
	case "alt":
		task = "text"
		if strings.TrimSpace(r.FormValue("image_src")) == "" || (strings.TrimSpace(r.FormValue("title")) == "" && strings.TrimSpace(r.FormValue("content")) == "") {
			fail("请提供图片，以及标题或正文")
			return
		}
		messages = []map[string]string{
			{"role": "system", "content": "只输出 10 到 20 个汉字的图片说明。"},
			{"role": "user", "content": "标题：" + r.FormValue("title") + "\n正文：" + r.FormValue("content")},
		}
	default:
		fail("不支持的操作")
		return
	}
	text, err := chat(a, messages)
	if err != nil {
		fail(err.Error())
		return
	}
	if action == "tdk" {
		var tdk map[string]string
		raw := strings.TrimSpace(text)
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(raw, "```")
		if json.Unmarshal([]byte(strings.TrimSpace(raw)), &tdk) != nil {
			fail("AI 返回内容解析失败")
			return
		}
		aiWriteLog(a, user, "ok", scene, task, "tdk")
		front.JSON(w, 1, tdk)
		return
	}
	key := "content"
	if action == "ping" {
		key = "reply"
	}
	if action == "alt" {
		key = "alt"
	}
	aiWriteLog(a, user, "ok", scene, task, key)
	front.JSON(w, 1, map[string]string{key: text})
}

func aiRateOK(a *app.App, user string) bool {
	limit := domain.Atoi(a.DB.Conf("ai_rate_limit"), 20)
	if limit <= 0 {
		limit = 20
	}
	now := time.Now().Unix()
	var log []int64
	if v, ok := aiCallLog.Load(user); ok {
		for _, t := range v.([]int64) {
			if now-t < 60 {
				log = append(log, t)
			}
		}
	}
	if len(log) >= limit {
		aiCallLog.Store(user, log)
		return false
	}
	log = append(log, now)
	aiCallLog.Store(user, log)
	return true
}

func chat(a *app.App, messages []map[string]string) (string, error) {
	key := aiKeyPlain(a)
	if key == "" {
		if a.DB.Conf("ai_api_key") != "" {
			return "", errString("API Key 解密失败，请重新保存 API Key")
		}
		return "", errString("未配置 API Key")
	}
	base := strings.TrimRight(strings.TrimSpace(a.DB.Conf("ai_api_base")), "/")
	if base == "" {
		base = "https://api.deepseek.com"
	}
	url := base + "/v1/chat/completions"
	if strings.HasSuffix(base, "/chat/completions") {
		url = base
	} else if strings.HasSuffix(base, "/v1") {
		url = base + "/chat/completions"
	}
	model := a.DB.Conf("ai_model")
	if model == "" {
		model = "deepseek-chat"
	}
	body, _ := json.Marshal(map[string]any{
		"model":      model,
		"messages":   messages,
		"max_tokens": domain.Atoi(a.DB.Conf("ai_max_tokens"), 1024),
	})
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &parsed) != nil {
		return "", errString("AI 返回无法解析")
	}
	if parsed.Error.Message != "" {
		return "", errString(parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return "", errString("AI 返回内容为空")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

type errString string

func (e errString) Error() string { return string(e) }
