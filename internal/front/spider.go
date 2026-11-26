package front

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zn49/pbootcms-go/internal/app"
	"github.com/zn49/pbootcms-go/internal/domain"
	"github.com/zn49/pbootcms-go/internal/fsperm"
)

func noteSpider(a *app.App, r *http.Request) {
	if a.DB.Conf("spiderlog") == "0" {
		return
	}
	name := spiderName(r.UserAgent())
	if name == "" {
		return
	}
	url := r.URL.RequestURI()
	url = strings.NewReplacer("\x00", "", "\r", "", "\n", "", "\t", "").Replace(url)
	if len([]rune(url)) > 2000 {
		url = string([]rune(url)[:2000])
	}
	ua := r.UserAgent()
	now := time.Now()
	dir := filepath.Join(a.Cfg.SiteRoot, "data", "log", "spider", now.Format("2006"), now.Format("200601"))
	if err := fsperm.MkdirAll(dir, fsperm.DirPrivate); err != nil {
		return
	}
	line := domain.Now() + "\t" + name + "\t" + url + "\t" + ClientIP(r) + "\t" +
		cleanField(userOS(ua)) + "\t" + cleanField(userBS(ua)) + "\t" + cleanField(ua) + "\n"
	f, err := fsperm.OpenFile(filepath.Join(dir, now.Format("20060102")+".log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, fsperm.FileCache)
	if err != nil {
		return
	}
	_, _ = f.WriteString(line)
	_ = f.Close()
}

func cleanField(s string) string {
	return strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(strings.TrimSpace(s))
}

func spiderName(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "googlebot"):
		return "Google"
	case strings.Contains(ua, "baiduspider"):
		return "Baidu"
	case strings.Contains(ua, "webscan"):
		return "360WebScan"
	case strings.Contains(ua, "360spider"):
		return "360So"
	case strings.Contains(ua, "adsbot"):
		return "Adwords"
	case strings.Contains(ua, "bingbot"):
		return "Bing"
	case strings.Contains(ua, "slurp"):
		return "Yahoo"
	case strings.Contains(ua, "sosospider"):
		return "Soso"
	case strings.Contains(ua, "sogou"):
		return "Sogou"
	case strings.Contains(ua, "yodaobot"):
		return "Yodao"
	case strings.Contains(ua, "speedy"):
		return "Speedy"
	case strings.Contains(ua, "yandexbot"):
		return "Yandex"
	case strings.Contains(ua, "easouspider"):
		return "Easou"
	case strings.Contains(ua, "symantecspider"):
		return "Symantec"
	case strings.Contains(ua, "qiniu"):
		return "Qiniu"
	case strings.Contains(ua, "jiankongbao"):
		return "JianKongBao"
	case strings.Contains(ua, "dnspod"):
		return "DNSPod"
	case strings.Contains(ua, "linkpadbot"):
		return "Linkpad"
	case strings.Contains(ua, "mj12bot"):
		return "MJ12"
	case strings.Contains(ua, "dingtalkbot"):
		return "DingTalk"
	case strings.Contains(ua, "bytespider"):
		return "Byte"
	case strings.Contains(ua, "zoominfobot"):
		return "Zoominfo"
	case strings.Contains(ua, "yisouspider"):
		return "Yisou"
	case strings.Contains(ua, "spider"):
		return "other-spider"
	case strings.Contains(ua, "bot"):
		return "other-bot"
	default:
		return ""
	}
}

func userOS(ua string) string {
	u := strings.ToLower(ua)
	switch {
	case strings.Contains(u, "windows"):
		return "Windows"
	case strings.Contains(u, "android"):
		return "Android"
	case strings.Contains(u, "iphone"), strings.Contains(u, "ipad"):
		return "iOS"
	case strings.Contains(u, "mac os"):
		return "Mac"
	case strings.Contains(u, "linux"):
		return "Linux"
	default:
		return "Other"
	}
}

func userBS(ua string) string {
	u := strings.ToLower(ua)
	switch {
	case strings.Contains(u, "edg/"):
		return "Edge"
	case strings.Contains(u, "chrome"):
		return "Chrome"
	case strings.Contains(u, "firefox"):
		return "Firefox"
	case strings.Contains(u, "safari"):
		return "Safari"
	case strings.Contains(u, "msie"), strings.Contains(u, "trident"):
		return "IE"
	default:
		return "Other"
	}
}
