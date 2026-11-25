package admin

import (
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const textLogMaxLines = 2000
const textLogMaxBytes = 2 << 20

var aiLineRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}) \[([^\]]+)\] provider=(\S+) model=(\S+) user=(\S+) scene=(\S+) task=(\S+)(?: configured=(\S+))?(?: (.*))?$`)
var aiLineShortRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}) \[([^\]]+)\] provider=(\S+) user=(\S+)(?: (.*))?$`)

func textLogBase(siteRoot, typ string) string {
	sub := ""
	switch typ {
	case "spider":
		sub = "log/spider"
	case "ai":
		sub = "log/ai"
	default:
		return ""
	}
	return filepath.Join(siteRoot, "data", filepath.FromSlash(sub))
}

func textLogYears(siteRoot, typ string) []string {
	base := textLogBase(siteRoot, typ)
	seen := map[int]bool{}
	var years []int
	if entries, err := os.ReadDir(base); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			y, err := strconv.Atoi(e.Name())
			if err != nil || y < 1970 || y > 2100 {
				continue
			}
			if !seen[y] {
				seen[y] = true
				years = append(years, y)
			}
		}
	}
	y := time.Now().Year()
	if !seen[y] {
		years = append(years, y)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))
	out := make([]string, len(years))
	for i, n := range years {
		out[i] = strconv.Itoa(n)
	}
	return out
}

func textLogDays(siteRoot, typ string, year, month int) []int {
	dir := filepath.Join(textLogBase(siteRoot, typ), strconv.Itoa(year), sprintfYM(year, month))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var days []int
	seen := map[int]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".log")
		if len(name) != 8 {
			continue
		}
		y, _ := strconv.Atoi(name[:4])
		m, _ := strconv.Atoi(name[4:6])
		d, _ := strconv.Atoi(name[6:8])
		if y != year || m != month || d < 1 || d > 31 {
			continue
		}
		if !seen[d] {
			seen[d] = true
			days = append(days, d)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(days)))
	return days
}

func textLogDayFile(siteRoot, typ string, year, month, day int) string {
	return filepath.Join(textLogBase(siteRoot, typ), strconv.Itoa(year), sprintfYM(year, month), sprintfYMD(year, month, day)+".log")
}

func sprintfYM(y, m int) string {
	return strconv.Itoa(y) + pad2(m)
}

func sprintfYMD(y, m, d int) string {
	return strconv.Itoa(y) + pad2(m) + pad2(d)
}

func pad2(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

func textLogReadTail(path string) (lines []string, truncated bool, exists bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, false, false
	}
	if len(b) == 0 {
		return nil, false, true
	}
	truncated = len(b) > textLogMaxBytes
	if truncated {
		b = b[len(b)-textLogMaxBytes:]
		if i := strings.IndexByte(string(b), '\n'); i >= 0 && i+1 < len(b) {
			b = b[i+1:]
		}
	}
	s := strings.ReplaceAll(string(b), "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	parts := strings.Split(s, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) > textLogMaxLines {
		truncated = true
		parts = parts[len(parts)-textLogMaxLines:]
	}
	return parts, truncated, true
}

func textLogParse(typ, line string) map[string]string {
	if typ == "ai" {
		return parseAiLogLine(line)
	}
	cols := strings.SplitN(line, "\t", 7)
	for len(cols) < 7 {
		cols = append(cols, "")
	}
	keys := []string{"time", "spider", "url", "ip", "os", "browser", "ua"}
	row := map[string]string{}
	for i, k := range keys {
		row[k] = html.EscapeString(cols[i])
	}
	return row
}

func parseAiLogLine(line string) map[string]string {
	row := map[string]string{
		"time": "", "action": "", "provider": "", "model": "", "user": "",
		"scene": "", "task": "", "configured": "", "detail": "",
		"model_show": "-", "scene_show": "-", "task_show": "-",
	}
	if m := aiLineRe.FindStringSubmatch(line); m != nil {
		row["time"], row["action"], row["provider"], row["model"] = m[1], m[2], m[3], m[4]
		row["user"], row["scene"], row["task"] = m[5], m[6], m[7]
		if len(m) > 8 {
			row["configured"] = m[8]
		}
		if len(m) > 9 {
			row["detail"] = m[9]
		}
	} else if m := aiLineShortRe.FindStringSubmatch(line); m != nil {
		row["time"], row["action"], row["provider"], row["user"] = m[1], m[2], m[3], m[4]
		if len(m) > 5 {
			row["detail"] = m[5]
		}
	} else {
		row["detail"] = line
	}
	row["model_show"] = aiFilled(row["model"])
	if row["configured"] != "" {
		row["model_show"] += "（配置 " + row["configured"] + "）"
	}
	row["scene_show"] = aiScene(row["scene"])
	row["task_show"] = aiTask(row["task"])
	for k, v := range row {
		row[k] = html.EscapeString(v)
	}
	return row
}

func aiFilled(v string) string {
	if v == "" || v == "-" {
		return "-"
	}
	return v
}

func aiScene(s string) string {
	switch s {
	case "generate":
		return "生成文章"
	case "rewrite":
		return "文本处理"
	case "tdk":
		return "TDK"
	case "alt":
		return "图片Alt"
	case "ping":
		return "测试连接"
	case "", "-":
		return "-"
	default:
		return s
	}
}

func aiTask(t string) string {
	switch t {
	case "continue":
		return "续写"
	case "rewrite":
		return "改写"
	case "shorten":
		return "缩写"
	case "polish":
		return "润色"
	case "text":
		return "文本"
	case "url":
		return "图片URL"
	case "base64":
		return "本地图"
	case "", "-":
		return "-"
	default:
		return t
	}
}

func flag01(on bool) string {
	if on {
		return "1"
	}
	return ""
}
