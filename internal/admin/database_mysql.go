package admin

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gbootcms/internal/app"
	"gbootcms/internal/fsperm"
)

var tableNameRe = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

func safeTableList(in []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" || !tableNameRe.MatchString(t) || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func quoteIdent(tables []string) []string {
	out := make([]string, len(tables))
	for i, t := range tables {
		out[i] = "`" + t + "`"
	}
	return out
}

func mysqlTableNames(a *app.App) ([]string, error) {
	rows, err := a.DB.Maps("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, row := range rows {
		for _, v := range row {
			if tableNameRe.MatchString(v) {
				out = append(out, v)
			}
			break
		}
	}
	return out, nil
}

func mysqlTableStatus(a *app.App) ([]map[string]string, float64, float64) {
	rows, err := a.DB.Maps("SHOW TABLE STATUS")
	if err != nil {
		return nil, 0, 0
	}
	var sumSize, sumFree float64
	for _, row := range rows {
		dl := parseFloat(row["Data_length"])
		il := parseFloat(row["Index_length"])
		df := parseFloat(row["Data_free"])
		size := (dl + il) / 1024
		free := df / 1024
		sumSize += size
		sumFree += free
		row["Size"] = trimFloat(size)
		if df > 0 {
			row["Free"] = trimFloat(free) + " KB"
		} else {
			row["Free"] = "-"
		}
	}
	return rows, sumSize, sumFree
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func trimFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}

// shortcut: dumps INSERT rows without full PHP CREATE options parity; upgrade when restore tooling needs exact mysqldump
func backupMySQLTables(a *app.App, tables []string) error {
	dir := filepath.Join(a.Cfg.SiteRoot, "data", "backup", time.Now().Format("20060102150405"))
	if err := fsperm.MkdirAll(dir, fsperm.DirPrivate); err != nil {
		return err
	}
	var ok bool
	for _, table := range tables {
		if !tableNameRe.MatchString(table) {
			continue
		}
		var b strings.Builder
		b.WriteString("-- gbootcms dump\nSET NAMES utf8mb4;\n\n")
		create, err := a.DB.Maps("SHOW CREATE TABLE `" + table + "`")
		if err != nil || len(create) == 0 {
			return fmt.Errorf("表结构 %s: %v", table, err)
		}
		ct := create[0]["Create Table"]
		if ct == "" {
			for _, v := range create[0] {
				if strings.Contains(v, "CREATE TABLE") {
					ct = v
					break
				}
			}
		}
		b.WriteString("DROP TABLE IF EXISTS `" + table + "`;\n")
		b.WriteString(ct)
		b.WriteString(";\n\n")
		rows, err := a.DB.Maps("SELECT * FROM `" + table + "`")
		if err != nil {
			return err
		}
		for _, row := range rows {
			cols := make([]string, 0, len(row))
			vals := make([]string, 0, len(row))
			for k, v := range row {
				cols = append(cols, "`"+k+"`")
				vals = append(vals, mysqlQuote(v))
			}
			if len(cols) == 0 {
				continue
			}
			b.WriteString("INSERT INTO `" + table + "` (" + strings.Join(cols, ",") + ") VALUES (" + strings.Join(vals, ",") + ");\n")
		}
		name := table + ".sql"
		if err := fsperm.WriteFile(filepath.Join(dir, name), []byte(b.String()), fsperm.FilePrivate); err != nil {
			return err
		}
		ok = true
	}
	if !ok {
		return fmt.Errorf("没有可备份的表")
	}
	return nil
}

func mysqlQuote(v string) string {
	if v == "" {
		return "''"
	}
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `'`, `\'`)
	v = strings.ReplaceAll(v, "\n", `\n`)
	v = strings.ReplaceAll(v, "\r", `\r`)
	v = strings.ReplaceAll(v, "\x00", `\0`)
	return "'" + v + "'"
}
