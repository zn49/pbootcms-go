package store

import (
	"fmt"
	"strings"
)

// NormalizeDriver maps aliases to sqlite|mysql|pgsql.
func NormalizeDriver(d string) string {
	switch strings.ToLower(strings.TrimSpace(d)) {
	case "", "sqlite", "sqlite3":
		if strings.TrimSpace(d) == "" {
			return "sqlite"
		}
		return "sqlite"
	case "mysql", "mariadb":
		return "mysql"
	case "pgsql", "postgres", "postgresql", "pg":
		return "pgsql"
	default:
		return strings.ToLower(strings.TrimSpace(d))
	}
}

func validDriver(d string) bool {
	switch d {
	case "sqlite", "mysql", "pgsql":
		return true
	default:
		return false
	}
}

// Random returns the dialect random() expression.
func (d *DB) Random() string {
	if d.Driver == "mysql" {
		return "RAND()"
	}
	return "RANDOM()"
}

// IntCast wraps expr as integer for ORDER/MAX across dialects.
func (d *DB) IntCast(expr string) string {
	if !Ident(expr) && !looksSafeExpr(expr) {
		return "0"
	}
	switch d.Driver {
	case "mysql":
		return "CAST(" + expr + " AS SIGNED)"
	case "pgsql":
		return "CAST(NULLIF(" + expr + ",'') AS INTEGER)"
	default:
		return "CAST(" + expr + " AS INTEGER)"
	}
}

func looksSafeExpr(s string) bool {
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '.' {
			continue
		}
		return false
	}
	return s != ""
}

// AutoIDColumn is the id column DDL fragment for DIY tables.
func (d *DB) AutoIDColumn() string {
	switch d.Driver {
	case "mysql":
		return "id int unsigned NOT NULL AUTO_INCREMENT PRIMARY KEY"
	case "pgsql":
		return "id SERIAL PRIMARY KEY"
	default:
		return "id INTEGER PRIMARY KEY"
	}
}

// TextType is a generic string column type for the current driver.
func (d *DB) TextType() string {
	switch d.Driver {
	case "mysql":
		return "varchar(500) NULL"
	case "pgsql":
		return "TEXT"
	default:
		return "TEXT"
	}
}

// QuoteIdent quotes an identifier for the current dialect (after Ident check).
func (d *DB) QuoteIdent(name string) (string, error) {
	if !Ident(name) {
		return "", fmt.Errorf("非法标识符 %q", name)
	}
	switch d.Driver {
	case "mysql":
		return "`" + name + "`", nil
	case "pgsql":
		return `"` + name + `"`, nil
	default:
		return `"` + name + `"`, nil
	}
}
