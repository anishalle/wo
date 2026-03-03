package cli

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/anishalle/wo/internal/config"
	"github.com/anishalle/wo/internal/db"
	"github.com/anishalle/wo/internal/hooks"
	"github.com/anishalle/wo/internal/model"
)

func TestEnsureWorkspaceExistsOrPruneMissingKeepsByDefault(t *testing.T) {
	ctx := context.Background()
	missing := filepath.Join(t.TempDir(), "missing-workspace")
	store, ws := testWorkspaceWithPath(t, missing)
	defer store.Close()

	app := &App{Config: config.DefaultConfig(), Store: store}
	exists, removed, err := ensureWorkspaceExistsOrPrune(ctx, app, ws)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatalf("expected missing workspace to report exists=false")
	}
	if removed {
		t.Fatalf("expected missing workspace to remain indexed by default in non-interactive mode")
	}
}

func TestFinalizeSelectedWorkspaceOK(t *testing.T) {
	ctx := context.Background()
	existing := t.TempDir()
	store, ws := testWorkspaceWithPath(t, existing)
	defer store.Close()

	app := &App{Config: config.DefaultConfig(), Store: store}
	hookSvc := hooks.New(store, app.Config)

	out, ok, err := finalizeSelectedWorkspace(ctx, app, hookSvc, ws, true, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("expected resolve to succeed for existing workspace")
	}
	if out.Status != model.ResolveOK {
		t.Fatalf("expected status ok, got %q", out.Status)
	}
	if out.Path != existing {
		t.Fatalf("unexpected path: %q", out.Path)
	}
}

func testWorkspaceWithPath(t *testing.T, path string) (*db.Store, model.Workspace) {
	t.Helper()
	store, err := db.Open(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	id, err := store.UpsertWorkspace(ctx, model.Workspace{
		Path:     path,
		RepoName: filepath.Base(path),
		Owner:    "local",
		Source:   "wo",
		HasWO:    true,
	}, nil)
	if err != nil {
		store.Close()
		t.Fatal(err)
	}
	return store, model.Workspace{
		ID:       id,
		Path:     path,
		RepoName: filepath.Base(path),
		Owner:    "local",
		Source:   "wo",
		HasWO:    true,
	}
}
