package admin

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/zn49/pbootcms-go/internal/app"
	"github.com/zn49/pbootcms-go/internal/domain"
	"github.com/zn49/pbootcms-go/internal/fsperm"
)

var spaceRe = regexp.MustCompile(`\s+`)

func aiWriteLog(a *app.App, user, action, scene, task, detail string) {
	now := time.Now()
	dir := filepath.Join(a.Cfg.SiteRoot, "data", "log", "ai", now.Format("2006"), now.Format("200601"))
	if err := fsperm.MkdirAll(dir, fsperm.DirPrivate); err != nil {
		return
	}
	provider := or(a.DB.Conf("ai_provider"), "deepseek")
	model := or(a.DB.Conf("ai_model"), "deepseek-chat")
	line := domain.Now() + " [" + action + "]" +
		" provider=" + aiLogToken(provider) +
		" model=" + aiLogToken(model) +
		" user=" + aiLogToken(user) +
		" scene=" + aiLogToken(scene) +
		" task=" + aiLogToken(task)
	if detail != "" {
		detail = spaceRe.ReplaceAllString(stripTags(detail), " ")
		detail = strings.TrimSpace(detail)
		if utf8.RuneCountInString(detail) > 200 {
			detail = string([]rune(detail)[:200])
		}
		line += " " + detail
	}
	line += "\n"
	f, err := fsperm.OpenFile(filepath.Join(dir, now.Format("20060102")+".log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, fsperm.FileCache)
	if err != nil {
		return
	}
	_, _ = f.WriteString(line)
	_ = f.Close()
}

func aiLogToken(v string) string {
	v = strings.TrimSpace(stripTags(v))
	v = spaceRe.ReplaceAllString(v, "_")
	if v == "" || strings.Contains(v, "=") {
		return "-"
	}
	return v
}

func stripTags(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == '<':
			in = true
		case r == '>':
			in = false
		case !in:
			b.WriteRune(r)
		}
	}
	return b.String()
}
