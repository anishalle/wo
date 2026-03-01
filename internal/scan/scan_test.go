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
