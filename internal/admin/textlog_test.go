package admin

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTextLogSpiderRoundtrip(t *testing.T) {
	root := t.TempDir()
	now := time.Now()
	y, m, d := now.Year(), int(now.Month()), now.Day()
	dir := filepath.Join(textLogBase(root, "spider"), strconvYear(y), sprintfYM(y, m))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	line := now.Format("2006-01-02 15:04:05") + "\tbaidu\thttp://x/\t1.1.1.1\tLinux\tChrome\tMozilla\n"
	if err := os.WriteFile(textLogDayFile(root, "spider", y, m, d), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	days := textLogDays(root, "spider", y, m)
	if len(days) == 0 || days[0] != d {
		t.Fatalf("days %v", days)
	}
	lines, _, ok := textLogReadTail(textLogDayFile(root, "spider", y, m, d))
	if !ok || len(lines) != 1 {
		t.Fatalf("read %v %v", ok, lines)
	}
	row := textLogParse("spider", lines[0])
	if row["spider"] != "baidu" || row["url"] != "http://x/" {
		t.Fatalf("%v", row)
	}
}

func strconvYear(y int) string {
	return sprintfYM(y, 1)[:4]
}
