package store

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

var (
	reComment = regexp.MustCompile(`(?i)\s+COMMENT\s+'[^']*'`)
	reEngine  = regexp.MustCompile(`(?i)\)\s*ENGINE\s*=[\s\S]*$`)
	reKey     = regexp.MustCompile(`(?m)^\s*KEY\s+[^\n]*,?\n`)
	reUnique  = regexp.MustCompile(`(?i)UNIQUE\s+KEY\s+\x60?[A-Za-z0-9_]+\x60?\s*(\([^)]+\))`)
	reInt     = regexp.MustCompile(`(?i)\b(?:tiny|small|medium|big)?int\(\d+\)\s+unsigned`)
	reChar    = regexp.MustCompile(`(?i)\b(?:var)?char\(\d+\)`)
	reText    = regexp.MustCompile(`(?i)\b(?:medium|long)?text\b`)
	reDT      = regexp.MustCompile(`(?i)\bdatetime\b`)
	reAuto    = regexp.MustCompile(`(?i)\s+AUTO_INCREMENT\b`)
	reUnsign  = regexp.MustCompile(`(?i)\s+unsigned\b`)
)

// ImportMySQLServer loads a phpMyAdmin dump into a real MySQL/MariaDB database.
func ImportMySQLServer(db *sql.DB, dump string) error {
	// MySQL 8 rejects Pboot's zero datetimes unless session mode is relaxed.
	_, _ = db.Exec(`SET SESSION sql_mode = REPLACE(REPLACE(@@SESSION.sql_mode,'NO_ZERO_DATE',''),'NO_ZERO_IN_DATE','')`)
	dump = strings.ReplaceAll(dump, "'0000-00-00 00:00:00'", "'1970-01-01 00:00:00'")
	dump = strings.ReplaceAll(dump, "'0000-00-00'", "'1970-01-01'")
	for _, stmt := range splitSQL(dump) {
		up := strings.ToUpper(strings.TrimSpace(stmt))
		if strings.HasPrefix(up, "SET ") || strings.HasPrefix(up, "LOCK ") || strings.HasPrefix(up, "UNLOCK ") || strings.HasPrefix(up, "/*") {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			snip := strings.TrimSpace(stmt)
			if len(snip) > 160 {
				snip = snip[:160]
			}
			return fmt.Errorf("%w: %s", err, snip)
		}
	}
	return nil
}

// ImportMySQL loads a phpMyAdmin dump into SQLite.
func ImportMySQL(db *sql.DB, dump string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for _, stmt := range splitSQL(dump) {
		up := strings.ToUpper(strings.TrimSpace(stmt))
		if strings.HasPrefix(up, "SET ") || strings.HasPrefix(up, "LOCK ") || strings.HasPrefix(up, "UNLOCK ") || strings.HasPrefix(up, "/*") {
			continue
		}
		q := stmt
		if strings.HasPrefix(up, "CREATE TABLE") || strings.HasPrefix(up, "DROP TABLE") {
			q = toSQLiteCreate(stmt)
		}
		if _, err := tx.Exec(q); err != nil {
			tx.Rollback()
			snip := strings.TrimSpace(q)
			if len(snip) > 160 {
				snip = snip[:160]
			}
			return fmt.Errorf("%w: %s", err, snip)
		}
	}
	return tx.Commit()
}

func toSQLiteCreate(stmt string) string {
	stmt = regexp.MustCompile(`(?i)\bint\(\d+\)[^,\n]*AUTO_INCREMENT`).ReplaceAllString(stmt, "INTEGER NOT NULL")
	stmt = reEngine.ReplaceAllString(stmt, ")")
	stmt = reComment.ReplaceAllString(stmt, "")
	stmt = reUnique.ReplaceAllString(stmt, "UNIQUE $1")
	stmt = reKey.ReplaceAllString(stmt, "")
	stmt = reAuto.ReplaceAllString(stmt, "")
	stmt = reUnsign.ReplaceAllString(stmt, "")
	stmt = reInt.ReplaceAllString(stmt, "INTEGER")
	stmt = reChar.ReplaceAllString(stmt, "TEXT")
	stmt = reText.ReplaceAllString(stmt, "TEXT")
	stmt = reDT.ReplaceAllString(stmt, "TEXT")
	stmt = strings.ReplaceAll(stmt, "`", "")
	stmt = regexp.MustCompile(`,\s*\)`).ReplaceAllString(stmt, ")")
	return addDefaults(stmt)
}

// Seed inserts omit some NOT NULL columns. SQLite rejects that unless they have a default.
func addDefaults(stmt string) string {
	if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(stmt)), "CREATE TABLE") {
		return stmt
	}
	lines := strings.Split(stmt, "\n")
	for i, line := range lines {
		u := strings.ToUpper(line)
		if !strings.Contains(u, "NOT NULL") || strings.Contains(u, "DEFAULT") || strings.Contains(u, "PRIMARY") || strings.Contains(u, "INTEGER") || strings.Contains(u, "BIGINT") || strings.Contains(u, "TIMESTAMP") || strings.Contains(u, "SERIAL") {
			continue
		}
		line = strings.TrimRight(line, " \t\r")
		comma := ""
		if strings.HasSuffix(line, ",") {
			line = line[:len(line)-1]
			comma = ","
		}
		lines[i] = line + " DEFAULT ''" + comma
	}
	return strings.Join(lines, "\n")
}

func splitSQL(s string) []string {
	var out []string
	var b strings.Builder
	inQ := false
	esc := false
	for _, r := range s {
		if inQ {
			b.WriteRune(r)
			if esc {
				esc = false
				continue
			}
			if r == '\\' {
				esc = true
				continue
			}
			if r == '\'' {
				inQ = false
			}
			continue
		}
		if r == '\'' {
			inQ = true
			b.WriteRune(r)
			continue
		}
		if r == ';' {
			stmt := strings.TrimSpace(b.String())
			stmt = strings.TrimLeft(stmt, "\n\r\t ")
			for strings.HasPrefix(stmt, "--") || strings.HasPrefix(stmt, "/*") {
				if i := strings.Index(stmt, "\n"); i >= 0 {
					stmt = strings.TrimSpace(stmt[i+1:])
				} else {
					stmt = ""
					break
				}
			}
			if stmt != "" {
				out = append(out, stmt)
			}
			b.Reset()
			continue
		}
		b.WriteRune(r)
	}
	return out
}
