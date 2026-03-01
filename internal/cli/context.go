package cli

import (
	"context"
	"fmt"

	"github.com/anishalle/wo/internal/config"
	"github.com/anishalle/wo/internal/db"
)

type App struct {
	Config     config.Config
	ConfigPath string
	Store      *db.Store
}

func NewApp(ctx context.Context) (*App, error) {
	cfg, cfgPath, err := config.Load()
	if err != nil {
		return nil, err
	}
	if err := cfg.Normalize(); err != nil {
		return nil, err
	}
	if err := config.EnsureConfigFile(cfgPath, cfg); err != nil {
		return nil, fmt.Errorf("ensure config: %w", err)
	}
	dataDir, err := config.DataDir()
	if err != nil {
		return nil, err
	}
	store, err := db.Open(dataDir)
	if err != nil {
		return nil, err
	}
	return &App{Config: cfg, ConfigPath: cfgPath, Store: store}, nil
}

func (a *App) Close() error {
	if a == nil || a.Store == nil {
		return nil
	}
	return a.Store.Close()
}
