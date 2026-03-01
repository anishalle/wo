package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Roots      []string         `toml:"roots"`
	Scan       ScanConfig       `toml:"scan"`
	Search     SearchConfig     `toml:"search"`
	UI         UIConfig         `toml:"ui"`
	Hooks      HooksConfig      `toml:"hooks"`
	Correction CorrectionConfig `toml:"correction"`
}

type ScanConfig struct {
	DepthDefault  int  `toml:"depth_default"`
	FollowSymlink bool `toml:"follow_symlink"`
}

type SearchConfig struct {
	Backend string `toml:"backend"`
}

type UIConfig struct {
	Theme string `toml:"theme"`
}

type HooksConfig struct {
	Enabled bool `toml:"enabled"`
}

type CorrectionConfig struct {
	Enabled  bool    `toml:"enabled"`
	MinScore float64 `toml:"min_score"`
	MinGap   float64 `toml:"min_gap"`
}

func DefaultConfig() Config {
	cfg := Config{
		Scan: ScanConfig{
			DepthDefault: 1,
		},
		Search: SearchConfig{
			Backend: "auto",
		},
		UI:    UIConfig{Theme: "gh"},
		Hooks: HooksConfig{Enabled: true},
		Correction: CorrectionConfig{
			Enabled:  true,
			MinScore: 0.72,
			MinGap:   0.10,
		},
	}
	if root := defaultWorkspaceRoot(); root != "" {
		cfg.Roots = append(cfg.Roots, root)
	}
	return cfg
}

func defaultWorkspaceRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	candidate := filepath.Join(home, "workspaces")
	if st, err := os.Stat(candidate); err == nil && st.IsDir() {
		return candidate
	}
	return ""
}

func ConfigPath() (string, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfgDir, "wo", "config.toml"), nil
}

func DataDir() (string, error) {
	if runtime.GOOS == "linux" {
		if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
			return filepath.Join(dataHome, "wo"), nil
		}
	}
	base, err := os.UserCacheDir()
	if err != nil {
		base, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(base, "wo"), nil
}

func Load() (Config, string, error) {
	cfg := DefaultConfig()
	path, err := ConfigPath()
	if err != nil {
		return cfg, "", err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return cfg, path, nil
	} else if err != nil {
		return cfg, path, err
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, path, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Normalize(); err != nil {
		return cfg, path, err
	}
	return cfg, path, nil
}

func (c *Config) Normalize() error {
	for i, root := range c.Roots {
		expanded, err := expandPath(root)
		if err != nil {
			return err
		}
		c.Roots[i] = expanded
	}
	if c.Scan.DepthDefault < 1 {
		c.Scan.DepthDefault = 1
	}
	if c.Search.Backend == "" {
		c.Search.Backend = "auto"
	}
	if c.Search.Backend != "auto" && c.Search.Backend != "internal" && c.Search.Backend != "fzf" {
		return fmt.Errorf("search.backend must be one of auto|internal|fzf")
	}
	if c.Correction.MinScore <= 0 || c.Correction.MinScore > 1 {
		return fmt.Errorf("correction.min_score must be in (0,1]")
	}
	if c.Correction.MinGap < 0 || c.Correction.MinGap > 1 {
		return fmt.Errorf("correction.min_gap must be in [0,1]")
	}
	if len(c.Roots) == 0 {
		if root := defaultWorkspaceRoot(); root != "" {
			c.Roots = []string{root}
		}
	}
	return nil
}

func EnsureConfigFile(path string, cfg Config) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := toml.NewEncoder(f)
	return enc.Encode(cfg)
}

func expandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return filepath.Clean(path), nil
}
