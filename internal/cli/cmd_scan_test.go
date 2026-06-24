package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anishalle/wo/internal/config"
)

func TestScanTargetsFromArgsUsesAbsolutePathAndDefaultDepth(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	root := filepath.Join(home, "workspaces")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(home); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})

	app := &App{Config: config.Config{Scan: config.ScanConfig{DepthDefault: 1}}}
	targets, err := scanTargetsFromArgs([]string{"workspaces/"}, app)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 scan target, got %d", len(targets))
	}
	if targets[0].Path != resolvedRoot {
		t.Fatalf("expected absolute root %q, got %q", resolvedRoot, targets[0].Path)
	}
	if targets[0].Depth != 1 {
		t.Fatalf("expected default depth 1, got %d", targets[0].Depth)
	}
}

func TestScanTargetsFromArgsLoadsCustomScanFile(t *testing.T) {
	tmp := t.TempDir()
	scanFile := filepath.Join(tmp, "paths.wo")
	if err := os.WriteFile(scanFile, []byte("./workspaces 4\n# comment\n~/src 5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(tmp, "home")
	t.Setenv("HOME", home)

	app := &App{Config: config.Config{Scan: config.ScanConfig{DepthDefault: 1}}}
	targets, err := scanTargetsFromArgs([]string{scanFile}, app)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 2 {
		t.Fatalf("expected 2 scan targets, got %d", len(targets))
	}
	if targets[0].Path != filepath.Join(tmp, "workspaces") || targets[0].Depth != 4 {
		t.Fatalf("unexpected first target: %+v", targets[0])
	}
	if targets[1].Path != filepath.Join(home, "src") || targets[1].Depth != 5 {
		t.Fatalf("unexpected second target: %+v", targets[1])
	}
}

func TestScanTargetsFromArgsUsesDefaultScanFile(t *testing.T) {
	tmp := t.TempDir()
	xdg := filepath.Join(tmp, ".config")
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	if err := os.MkdirAll(filepath.Join(xdg, "wo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xdg, "wo", "paths.wo"), []byte("/tmp/workspaces 6\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := &App{Config: config.Config{Scan: config.ScanConfig{DepthDefault: 1}}}
	targets, err := scanTargetsFromArgs(nil, app)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 scan target, got %d", len(targets))
	}
	if targets[0].Path != "/tmp/workspaces" || targets[0].Depth != 6 {
		t.Fatalf("unexpected target: %+v", targets[0])
	}
}
