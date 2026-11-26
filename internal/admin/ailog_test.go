package admin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zn49/pbootcms-go/internal/app"
	"github.com/zn49/pbootcms-go/internal/config"
	"github.com/zn49/pbootcms-go/internal/store"
)

func TestAiWriteLogRoundtrip(t *testing.T) {
	root := t.TempDir()
	a := &app.App{
		Cfg: config.File{SiteRoot: root},
		DB:  &store.DB{},
	}
	aiWriteLog(a, "admin", "ok", "generate", "-", "content")
	now := time.Now()
	p := filepath.Join(root, "data", "log", "ai", now.Format("2006"), now.Format("200601"), now.Format("20060102")+".log")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	line := strings.TrimSpace(string(b))
	row := textLogParse("ai", line)
	if row["action"] != "ok" || row["scene"] != "generate" || row["user"] != "admin" {
		t.Fatalf("%v from %s", row, line)
	}
}
