package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/zn49/pbootcms-go/internal/app"
	"github.com/zn49/pbootcms-go/internal/config"
	"github.com/zn49/pbootcms-go/internal/httpx"
	"github.com/zn49/pbootcms-go/internal/install"
)

func main() {
	cfgPath := "config.json"
	if v := os.Getenv("GBOOT_CONFIG"); v != "" {
		cfgPath = v
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatal(err)
		}
		cfg = config.Default()
		slog.Info("no config.json; install mode", "addr", cfg.Addr)
	}
	a := app.New(cfg)
	a.CfgPath = cfgPath
	if install.Needed(a) {
		slog.Info("awaiting install at /install")
	} else if err := a.Bootstrap(); err != nil {
		log.Fatal(err)
	}
	defer a.ClearBootstrap()
	if err := httpx.Listen(a); err != nil {
		log.Fatal(err)
	}
}
