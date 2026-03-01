package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWorkspaceFile(t *testing.T) {
	dir := t.TempDir()
	content := []byte("name = \"harp\"\nowner = \"hackutd\"\n[enter]\ncommands = [\"nvim .\", \"make dev\"]\n")
	if err := os.WriteFile(filepath.Join(dir, ".wo"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, exists, err := LoadWorkspaceFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected .wo to exist")
	}
	if cfg.Name != "harp" || cfg.Owner != "hackutd" {
		t.Fatalf("unexpected parsed values: %+v", cfg)
	}
	if len(cfg.Enter.Commands) != 2 {
		t.Fatalf("expected 2 enter commands")
	}
}
