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

func TestLoadWorkspaceFileProfiles(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`
name = "harp"
owner = "hackutd"

[enter]
commands = ["echo startup"]

[cursor]
command = "cursor ."

[lint]
commands = ["make lint", "make test"]
chdir = false
`)
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
	if !cfg.HasEnter {
		t.Fatalf("expected has_enter to be true")
	}
	cursor, ok := cfg.Profiles["cursor"]
	if !ok {
		t.Fatalf("expected cursor profile")
	}
	if len(cursor.Commands) != 1 || cursor.Commands[0] != "cursor ." {
		t.Fatalf("unexpected cursor profile commands: %#v", cursor.Commands)
	}
	if !cursor.Chdir {
		t.Fatalf("expected cursor chdir default to true")
	}
	lint, ok := cfg.Profiles["lint"]
	if !ok {
		t.Fatalf("expected lint profile")
	}
	if len(lint.Commands) != 2 {
		t.Fatalf("expected 2 lint commands, got %d", len(lint.Commands))
	}
	if lint.Chdir {
		t.Fatalf("expected lint chdir=false")
	}
}

func TestLoadGlobalHookFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	globalPath, err := GlobalHookConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte(`
[enter]
commands = ["echo no"]

[cursor]
command = "cursor ."
`)
	if err := os.WriteFile(globalPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, exists, err := LoadGlobalHookFile()
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected global config.wo to exist")
	}
	if !cfg.HasEnter {
		t.Fatalf("expected global config to report enter section")
	}
	if len(cfg.Enter.Commands) != 1 {
		t.Fatalf("expected enter commands in parsed global config")
	}
	if _, ok := cfg.Profiles["cursor"]; !ok {
		t.Fatalf("expected cursor profile in global config")
	}
}
