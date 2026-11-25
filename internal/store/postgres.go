package store

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

// ImportPostgres loads a phpMyAdmin MySQL dump into PostgreSQL.
func ImportPostgres(db *sql.DB, dump string) error {
	dump = strings.ReplaceAll(dump, "'0000-00-00 00:00:00'", "'1970-01-01 00:00:00'")
	dump = strings.ReplaceAll(dump, "'0000-00-00'", "'1970-01-01'")
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for _, stmt := range splitSQL(dump) {
		up := strings.ToUpper(strings.TrimSpace(stmt))
		if strings.HasPrefix(up, "SET ") || strings.HasPrefix(up, "LOCK ") || strings.HasPrefix(up, "UNLOCK ") || strings.HasPrefix(up, "/*") {
			continue
		}
		var q string
		if strings.HasPrefix(up, "CREATE TABLE") || strings.HasPrefix(up, "DROP TABLE") {
			q = toPostgresCreate(stmt)
		} else {
			q = strings.ReplaceAll(stmt, "`", "")
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

var (
	rePGAutoInt = regexp.MustCompile(`(?i)\bint\(\d+\)[^,\n]*AUTO_INCREMENT`)
	rePGInt     = regexp.MustCompile(`(?i)\b(?:tiny|small|medium|big)?int\(\d+\)`)
)

func toPostgresCreate(stmt string) string {
	stmt = rePGAutoInt.ReplaceAllString(stmt, "SERIAL")
	stmt = reEngine.ReplaceAllString(stmt, ")")
	stmt = reComment.ReplaceAllString(stmt, "")
	stmt = reUnique.ReplaceAllString(stmt, "UNIQUE $1")
	stmt = reKey.ReplaceAllString(stmt, "")
	stmt = reAuto.ReplaceAllString(stmt, "")
	stmt = reUnsign.ReplaceAllString(stmt, "")
	// BIGINT: Pboot uses unsigned ints (e.g. score 4294967295) that overflow INTEGER
	stmt = rePGInt.ReplaceAllString(stmt, "BIGINT")
	stmt = reChar.ReplaceAllString(stmt, "TEXT")
	stmt = reText.ReplaceAllString(stmt, "TEXT")
	stmt = reDT.ReplaceAllString(stmt, "TIMESTAMP")
	stmt = strings.ReplaceAll(stmt, "`", "")
	stmt = regexp.MustCompile(`,\s*\)`).ReplaceAllString(stmt, ")")
	return addDefaults(stmt)
}
