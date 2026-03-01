package db

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/anishalle/wo/internal/model"
)

func TestOpenAndUpsertWorkspace(t *testing.T) {
	ctx := context.Background()
	dataDir := filepath.Join(t.TempDir(), "data")
	store, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ws := model.Workspace{
		Path:      "/tmp/harp",
		RepoName:  "harp",
		Owner:     "hackutd",
		Source:    "git",
		HasGit:    true,
		HasWO:     false,
		RemoteURL: "git@github.com:hackutd/harp.git",
	}
	id1, err := store.UpsertWorkspace(ctx, ws, []string{"hp"})
	if err != nil {
		t.Fatal(err)
	}
	if id1 == 0 {
		t.Fatalf("expected non-zero workspace id")
	}
	id2, err := store.UpsertWorkspace(ctx, ws, []string{"harp"})
	if err != nil {
		t.Fatal(err)
	}
	if id2 != id1 {
		t.Fatalf("expected stable workspace id, got %d then %d", id1, id2)
	}
	byAlias, err := store.FindByAlias(ctx, "harp")
	if err != nil {
		t.Fatal(err)
	}
	if len(byAlias) != 1 {
		t.Fatalf("expected 1 alias workspace, got %d", len(byAlias))
	}
}

func TestTrustRoundTrip(t *testing.T) {
	ctx := context.Background()
	dataDir := filepath.Join(t.TempDir(), "data")
	store, err := Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	id, err := store.UpsertWorkspace(ctx, model.Workspace{
		Path:     "/tmp/project",
		RepoName: "project",
		Owner:    "local",
		Source:   "wo",
		HasWO:    true,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetTrust(ctx, id, TrustAllow, "fp1"); err != nil {
		t.Fatal(err)
	}
	rec, err := store.GetTrust(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Decision != TrustAllow || rec.Fingerprint != "fp1" {
		t.Fatalf("unexpected trust record: %+v", rec)
	}
	if err := store.ResetTrust(ctx, id); err != nil {
		t.Fatal(err)
	}
}
