// One-shot: strip Pboot demo rows from live DB + seed SQL, rebrand to PbootCMS-Go.
package main

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"

	_ "github.com/glebarez/go-sqlite"
)

func main() {
	dbPath := "site/data/pbootcms.db"
	sqlPath := "site/static/backup/sql/pbootcms.sql"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		sqlPath = os.Args[2]
	}
	if err := cleanDB(dbPath); err != nil {
		fatal(err)
	}
	if err := cleanSQL(sqlPath); err != nil {
		fatal(err)
	}
	fmt.Println("demo cleared; branded PbootCMS-Go:", dbPath, sqlPath)
}

func cleanDB(dsn string) error {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	stmts := []string{
		`DELETE FROM ay_content_ext`,
		`DELETE FROM ay_content`,
		`DELETE FROM ay_member_comment`,
		`DELETE FROM ay_message`,
		`DELETE FROM ay_slide`,
		`DELETE FROM ay_link`,
		`DELETE FROM ay_tags`,
		`DELETE FROM ay_label`,
		`DELETE FROM ay_member`,
		`DELETE FROM ay_content_sort`,
		`UPDATE ay_site SET
			title='PbootCMS-Go',
			subtitle='Go 企业网站内容管理系统',
			domain='',
			keywords='pbootcms-go,cms',
			description='PbootCMS-Go',
			icp='',
			copyright='Copyright © PbootCMS-Go All Rights Reserved.'
			WHERE acode='cn'`,
		`UPDATE ay_company SET
			name='PbootCMS-Go',
			address='', postcode='', contact='', mobile='', phone='', fax='',
			email='', qq='', weixin='', blicense='', other=''
			WHERE acode='cn'`,
		`UPDATE ay_config SET value='PbootCMS-Go' WHERE name IN ('watermark_text','cmsname')`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("db: %w", err)
		}
	}
	return nil
}

func cleanSQL(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := string(b)
	demo := []string{
		"ay_content_ext", "ay_content", "ay_content_sort", "ay_slide", "ay_link",
		"ay_message", "ay_member_comment", "ay_tags", "ay_label", "ay_member",
	}
	for _, t := range demo {
		s2, ok := dropInsert(s, t)
		if !ok {
			fmt.Fprintf(os.Stderr, "warn: no INSERT for %s\n", t)
			continue
		}
		s = s2
	}
	s = strings.Replace(s,
		"('27','watermark_text','PbootCMS','2','255','水印文本')",
		"('27','watermark_text','PbootCMS-Go','2','255','水印文本')", 1)

	siteRe := regexp.MustCompile(`(?s)INSERT INTO \` + "`" + `ay_site\` + "`" + ` \(.*?\) VALUES\s*\n\([^;]+\)\s*;`)
	s = siteRe.ReplaceAllString(s, "INSERT INTO `ay_site` (`id`,`acode`,`title`,`subtitle`,`domain`,`logo`,`keywords`,`description`,`icp`,`theme`,`statistical`,`copyright`) VALUES\n"+
		"('1','cn','PbootCMS-Go','Go 企业网站内容管理系统','','/static/images/logo.png','pbootcms-go,cms','PbootCMS-Go','','default','','Copyright © PbootCMS-Go All Rights Reserved.');")

	coRe := regexp.MustCompile(`(?s)INSERT INTO \` + "`" + `ay_company\` + "`" + ` \(.*?\) VALUES\s*\n\([^;]+\)\s*;`)
	s = coRe.ReplaceAllString(s, "INSERT INTO `ay_company` (`id`,`acode`,`name`,`address`,`postcode`,`contact`,`mobile`,`phone`,`fax`,`email`,`qq`,`weixin`,`blicense`,`other`) VALUES\n"+
		"('1','cn','PbootCMS-Go','','','','','','','','','','','');")

	return os.WriteFile(path, []byte(s), 0o644)
}

// dropInsert removes INSERT INTO `table` ... ; respecting quotes/escapes.
func dropInsert(s, table string) (string, bool) {
	needle := "INSERT INTO `" + table + "`"
	i := strings.Index(s, needle)
	if i < 0 {
		return s, false
	}
	j := i + len(needle)
	inS, inD, esc := false, false, false
	for j < len(s) {
		c := s[j]
		if esc {
			esc = false
			j++
			continue
		}
		if c == '\\' && (inS || inD) {
			esc = true
			j++
			continue
		}
		if !inD && c == '\'' {
			inS = !inS
			j++
			continue
		}
		if !inS && c == '"' {
			inD = !inD
			j++
			continue
		}
		if !inS && !inD && c == ';' {
			j++ // include ;
			for j < len(s) && (s[j] == '\n' || s[j] == '\r') {
				j++
			}
			return s[:i] + "-- cleared demo " + needle + ";\n" + s[j:], true
		}
		j++
	}
	return s, false
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
