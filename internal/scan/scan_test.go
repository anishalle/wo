package scan

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/anishalle/wo/internal/db"
)

func TestRunDepthLimit(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "workspaces")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	depth1 := filepath.Join(root, "github.com", "hackutd")
	if err := os.MkdirAll(depth1, 0o755); err != nil {
		t.Fatal(err)
	}
	depth1Repo := filepath.Join(depth1, "harp")
	if err := os.MkdirAll(filepath.Join(depth1Repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	depth2Repo := filepath.Join(depth1Repo, "nested")
	if err := os.MkdirAll(depth2Repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(depth2Repo, ".wo"), []byte("name = \"nested\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := db.Open(filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()
	_, err = Run(ctx, store, Options{Roots: []string{root}, Depth: 4})
	if err != nil {
		t.Fatal(err)
	}
	all, err := store.ListWorkspaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 workspaces with depth 4, got %d", len(all))
	}

	_, err = Run(ctx, store, Options{Roots: []string{root}, Depth: 3, Prune: true})
	if err != nil {
		t.Fatal(err)
	}
	all, err = store.ListWorkspaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 workspace with depth 3, got %d", len(all))
	}

	_, err = Run(ctx, store, Options{Roots: []string{root}, Depth: 1, Prune: true})
	if err != nil {
		t.Fatal(err)
	}
	all, err = store.ListWorkspaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 0 {
		t.Fatalf("expected 0 workspaces with depth 1, got %d", len(all))
	}
}

func TestRunResolvesRelativeRootsToAbsolutePaths(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	root := filepath.Join(home, "workspaces")
	repo := filepath.Join(root, "github.com", "anishalle", "wo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	resolvedRepo, err := filepath.EvalSymlinks(repo)
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

	store, err := db.Open(filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()
	if _, err := Run(ctx, store, Options{Roots: []string{"workspaces/"}, Depth: 3}); err != nil {
		t.Fatal(err)
	}

	workspaces, err := store.ListWorkspaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(workspaces))
	}
	if workspaces[0].Path != resolvedRepo {
		t.Fatalf("expected absolute workspace path %q, got %q", resolvedRepo, workspaces[0].Path)
	}

	roots, err := store.ListScanRoots(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 {
		t.Fatalf("expected 1 scan root, got %d", len(roots))
	}
	if roots[0].Path != resolvedRoot {
		t.Fatalf("expected absolute scan root %q, got %q", resolvedRoot, roots[0].Path)
	}
}
