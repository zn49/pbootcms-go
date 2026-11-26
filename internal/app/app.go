package app

import (
	"path/filepath"

	"github.com/zn49/pbootcms-go/internal/auth"
	"github.com/zn49/pbootcms-go/internal/cache"
	"github.com/zn49/pbootcms-go/internal/config"
	"github.com/zn49/pbootcms-go/internal/fsperm"
	"github.com/zn49/pbootcms-go/internal/store"
)

// App is the running site. New only stores config; Bootstrap opens resources.
type App struct {
	Cfg     config.File
	CfgPath string // path to config.json for fingerprint/db renames
	DB      *store.DB
	Admin   *auth.Store
	Front   *auth.Store
	Pages   cache.Page

	bootstrapped bool
}

func New(cfg config.File) *App {
	return &App{Cfg: cfg}
}

func (a *App) IsBootstrapped() bool { return a.bootstrapped }

// Bootstrap opens the database, sessions and page cache.
// A second call releases the previous resources first.
func (a *App) Bootstrap() error {
	if a.bootstrapped {
		if err := a.ClearBootstrap(); err != nil {
			return err
		}
	}
	fsperm.HardenSite(a.Cfg.SiteRoot)
	schema := filepath.Join(a.Cfg.SiteRoot, "static", "backup", "sql", "pbootcms.sql")
	db, err := store.Open(a.Cfg.Driver, a.Cfg.DSN, a.Cfg.SiteRoot, schema)
	if err != nil {
		return err
	}
	data := filepath.Join(a.Cfg.SiteRoot, "data")
	a.DB = db
	a.Admin = auth.NewStore(filepath.Join(data, "session", "admin"))
	a.Front = auth.NewStore(filepath.Join(data, "session", "front"))
	a.Pages = cache.Page{Dir: filepath.Join(data, "cache")}
	if a.Cfg.Driver == "sqlite" && a.Cfg.DSN != "" {
		fsperm.HardenFile(a.Cfg.DSN, fsperm.FilePrivate)
	}
	a.bootstrapped = true
	return nil
}

// ClearBootstrap closes the database. Sessions and the cache dir are left on disk.
func (a *App) ClearBootstrap() error {
	if !a.bootstrapped {
		return nil
	}
	a.bootstrapped = false
	var err error
	if a.DB != nil && a.DB.SQL != nil {
		err = a.DB.SQL.Close()
	}
	a.DB = nil
	return err
}
