package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/anishalle/wo/internal/model"
)

func LoadWorkspaceFile(dir string) (model.WorkspaceFileConfig, bool, error) {
	var cfg model.WorkspaceFileConfig
	path := filepath.Join(dir, ".wo")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return cfg, false, nil
	} else if err != nil {
		return cfg, false, err
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, true, fmt.Errorf("parse %s: %w", path, err)
	}
	if cfg.Enter.Shell == "" {
		cfg.Enter.Shell = "inherit"
	}
	return cfg, true, nil
}

func WorkspaceFingerprint(dir string) (string, error) {
	path := filepath.Join(dir, ".wo")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return HashWorkspaceContent(dir, b), nil
}
