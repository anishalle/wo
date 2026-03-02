package resolve

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/anishalle/wo/internal/config"
	"github.com/anishalle/wo/internal/db"
	"github.com/anishalle/wo/internal/model"
)

func TestResolveExactAndFuzzyConfirmationFlow(t *testing.T) {
	ctx := context.Background()
	store, err := db.Open(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	fixtures := []model.Workspace{
		{Path: "/tmp/hacktx", RepoName: "hacktx", Owner: "iampritisee", Source: "git", HasGit: true},
		{Path: "/tmp/hackuta", RepoName: "hackuta", Owner: "purvajpatel", Source: "git", HasGit: true},
		{Path: "/tmp/hackuta 2", RepoName: "hackuta 2", Owner: "purvajpatel", Source: "git", HasGit: true},
	}
	for _, ws := range fixtures {
		if _, err := store.UpsertWorkspace(ctx, ws, nil); err != nil {
			t.Fatal(err)
		}
	}

	svc := New(store, config.DefaultConfig())

	exact, err := svc.Resolve(ctx, "hacktx")
	if err != nil {
		t.Fatal(err)
	}
	if exact.Stage != "exact" || len(exact.Matches) != 1 {
		t.Fatalf("expected exact stage with one match, got stage=%q matches=%d", exact.Stage, len(exact.Matches))
	}

	fuzzy, err := svc.Resolve(ctx, "hck")
	if err != nil {
		t.Fatal(err)
	}
	if fuzzy.Stage != "fuzzy" {
		t.Fatalf("expected fuzzy stage, got %q", fuzzy.Stage)
	}
	if len(fuzzy.Matches) != 0 {
		t.Fatalf("expected zero auto matches for fuzzy query, got %d", len(fuzzy.Matches))
	}
	if len(fuzzy.TopSuggestions) == 0 {
		t.Fatalf("expected fuzzy suggestions")
	}
}
