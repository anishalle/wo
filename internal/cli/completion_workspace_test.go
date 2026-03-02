package cli

import (
	"testing"

	"github.com/anishalle/wo/internal/model"
)

func TestWorkspaceNameCompletionsDedup(t *testing.T) {
	items := []model.Workspace{
		{RepoName: "wo", Owner: "anishalle", Path: "/a/wo"},
		{RepoName: "wo", Owner: "hackutd", Path: "/b/wo"},
		{RepoName: "website", Owner: "anishalle", Path: "/a/website"},
	}
	got := workspaceNameCompletions(items, "w")
	if len(got) != 2 {
		t.Fatalf("expected 2 completions, got %d: %#v", len(got), got)
	}
	if got[0] != "website\tanishalle · /a/website" {
		t.Fatalf("unexpected first completion: %q", got[0])
	}
	if got[1] != "wo\tmultiple workspaces" {
		t.Fatalf("unexpected second completion: %q", got[1])
	}
}

func TestWorkspaceTokenCompletionsIncludesPath(t *testing.T) {
	items := []model.Workspace{
		{RepoName: "harp", Owner: "hackutd", Path: "/Users/ani/workspaces/github.com/hackutd/harp"},
	}
	got := workspaceTokenCompletions(items, "/Users/ani/workspaces")
	if len(got) != 1 {
		t.Fatalf("expected 1 path completion, got %d: %#v", len(got), got)
	}
	expected := "/Users/ani/workspaces/github.com/hackutd/harp\tpath"
	if got[0] != expected {
		t.Fatalf("unexpected path completion: %q", got[0])
	}
}
