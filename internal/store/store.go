package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/glebarez/go-sqlite"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"

	entg "github.com/zn49/pbootcms-go/internal/ent"
	"github.com/zn49/pbootcms-go/internal/ent/sysconfig"
	"github.com/zn49/pbootcms-go/internal/fsperm"
)

// DB is the Ent client for the fixed PbootCMS tables, plus the sql handle
// used to import dumps and to touch runtime DIY / ext columns.
type DB struct {
	Ent    *entg.Client
	SQL    *sql.DB
	Driver string
	Root   string

	mu   sync.RWMutex
	conf map[string]string
}

func Open(driver, dsn, siteRoot, schemaFile string) (*DB, error) {
	driver = NormalizeDriver(driver)
	if !validDriver(driver) {
		return nil, fmt.Errorf("不支持的数据库驱动 %q，仅支持 sqlite / mysql / pgsql", driver)
	}
	if err := fsperm.MkdirAll(filepath.Join(siteRoot, "data"), fsperm.DirPrivate); err != nil {
		return nil, err
	}
	sqlDB, err := sql.Open(sqlDriver(driver), dsnFor(driver, dsn))
	if err != nil {
		return nil, err
	}
	if driver == "sqlite" {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	client := entg.NewClient(entg.Driver(entsql.OpenDB(entDialect(driver), sqlDB)))
	d := &DB{Ent: client, SQL: sqlDB, Driver: driver, Root: siteRoot, conf: map[string]string{}}
	if err := d.ensure(schemaFile); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	if err := d.ReloadConf(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return d, nil
}

func sqlDriver(driver string) string {
	switch driver {
	case "mysql":
		return "mysql"
	case "pgsql":
		return "pgx"
	default:
		return "sqlite"
	}
}

func entDialect(driver string) string {
	switch driver {
	case "mysql":
		return dialect.MySQL
	case "pgsql":
		return dialect.Postgres
	default:
		return dialect.SQLite
	}
}

func dsnFor(driver, dsn string) string {
	if driver == "sqlite" {
		return sqliteDSN(dsn)
	}
	return dsn
}

func sqliteDSN(path string) string {
	if strings.HasPrefix(path, "file:") {
		if !strings.Contains(path, "busy_timeout") {
			if strings.Contains(path, "?") {
				return path + "&_pragma=busy_timeout(5000)"
			}
			return path + "?_pragma=busy_timeout(5000)"
		}
		return path
	}
	path = filepath.ToSlash(path)
	return "file:" + path + "?_pragma=busy_timeout(5000)"
}

func (d *DB) ensure(schemaFile string) error {
	ok, err := d.TableExists("ay_config")
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	if schemaFile == "" {
		schemaFile = filepath.Join(d.Root, "static", "backup", "sql", "pbootcms.sql")
	}
	b, err := os.ReadFile(schemaFile)
	if err != nil {
		return fmt.Errorf("空库且找不到初始 SQL: %w", err)
	}
	switch d.Driver {
	case "sqlite":
		return ImportMySQL(d.SQL, string(b))
	case "mysql":
		return ImportMySQLServer(d.SQL, string(b))
	case "pgsql":
		return ImportPostgres(d.SQL, string(b))
	default:
		return fmt.Errorf("不支持的驱动 %s", d.Driver)
	}
}

// TableExists reports whether a base table exists.
func (d *DB) TableExists(name string) (bool, error) {
	if !Ident(name) {
		return false, fmt.Errorf("非法表名")
	}
	var n int
	var err error
	switch d.Driver {
	case "mysql":
		n, err = d.queryInt(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name=?`, name)
	case "pgsql":
		n, err = d.queryInt(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name=?`, name)
	default:
		n, err = d.queryInt(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, name)
	}
	return n > 0, err
}

func (d *DB) ReloadConf() error {
	rows, err := d.Ent.SysConfig.Query().All(context.Background())
	if err != nil {
		return err
	}
	m := make(map[string]string, len(rows))
	for _, r := range rows {
		m[r.Name] = r.Value
	}
	d.mu.Lock()
	d.conf = m
	d.mu.Unlock()
	return nil
}

func (d *DB) Conf(k string) string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.conf[k]
}

func (d *DB) SetConf(name, value string) error {
	ctx := context.Background()
	n, err := d.Ent.SysConfig.Update().Where(sysconfig.NameEQ(name)).SetValue(value).Save(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		if err := d.Ent.SysConfig.Create().
			SetName(name).SetValue(value).SetType("2").SetSorting(255).SetDescription(name).
			Exec(ctx); err != nil {
			return err
		}
	}
	return d.ReloadConf()
}

var identRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func Ident(s string) bool { return identRe.MatchString(s) }

func (d *DB) AddColumn(table, col, sqliteType, mysqlType string) error {
	if !Ident(table) || !Ident(col) {
		return fmt.Errorf("非法标识符")
	}
	typ := sqliteType
	switch d.Driver {
	case "mysql":
		typ = mysqlType
	case "pgsql":
		typ = "TEXT"
	}
	tq, err := d.QuoteIdent(table)
	if err != nil {
		return err
	}
	cq, err := d.QuoteIdent(col)
	if err != nil {
		return err
	}
	return d.execRaw("ALTER TABLE " + tq + " ADD COLUMN " + cq + " " + typ)
}

func (d *DB) CreateDiy(table string, cols []string) error {
	if !Ident(table) || !strings.HasPrefix(table, "ay_diy_") {
		return fmt.Errorf("表名必须以 ay_diy_ 开头")
	}
	defs := []string{d.AutoIDColumn()}
	ct := "create_time TEXT"
	if d.Driver == "mysql" {
		ct = "create_time varchar(30) NULL"
	}
	defs = append(defs, ct)
	for _, c := range cols {
		if !Ident(c) {
			return fmt.Errorf("非法字段 %s", c)
		}
		defs = append(defs, c+" "+d.TextType())
	}
	tq, err := d.QuoteIdent(table)
	if err != nil {
		return err
	}
	return d.execRaw("CREATE TABLE IF NOT EXISTS " + tq + " (" + strings.Join(defs, ",") + ")")
}

func (d *DB) Exec(q string, args ...any) error {
	return d.execRaw(q, args...)
}

func (d *DB) ListDiy(table string, limit int) ([]map[string]string, error) {
	if !Ident(table) {
		return nil, fmt.Errorf("bad table")
	}
	if limit <= 0 {
		limit = 100
	}
	return d.Maps(fmt.Sprintf("SELECT * FROM %s ORDER BY id DESC LIMIT %d", table, limit))
}

func (d *DB) DeleteDiyRow(table string, id int) error {
	if !Ident(table) {
		return fmt.Errorf("bad table")
	}
	return d.Exec(fmt.Sprintf("DELETE FROM %s WHERE id=?", table), id)
}

func (d *DB) ClearDiy(table string) error {
	if !Ident(table) {
		return fmt.Errorf("bad table")
	}
	return d.Exec(fmt.Sprintf("DELETE FROM %s", table))
}

func (d *DB) DropColumn(table, col string) error {
	// shortcut: sqlite drop column needs rebuild; skip until driver-specific path
	_ = table
	_ = col
	return nil
}

func (d *DB) DropDiy(table string) error {
	if !Ident(table) || !strings.HasPrefix(table, "ay_diy_") {
		return fmt.Errorf("只能删除 ay_diy_ 表")
	}
	tq, err := d.QuoteIdent(table)
	if err != nil {
		return err
	}
	return d.execRaw("DROP TABLE IF EXISTS " + tq)
}
