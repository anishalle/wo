package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anishalle/wo/internal/config"
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

func TestHookProfileCompletionsWorkspaceFirstAndGlobalFallback(t *testing.T) {
	workspaceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspaceDir, ".wo"), []byte(`
[enter]
commands = ["echo startup"]

[cursor]
command = "cursor ."

[zed]
command = "zed ."
`), 0o644); err != nil {
		t.Fatal(err)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	globalPath, err := config.GlobalHookConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(globalPath, []byte(`
[enter]
commands = ["echo global startup should not autocomplete"]

[code]
command = "code ."

[cursor]
command = "global cursor should be hidden"

[vim]
command = "vim ."
`), 0o644); err != nil {
		t.Fatal(err)
	}

	items := []model.Workspace{
		{RepoName: "arlost", Owner: "anishalle", Path: workspaceDir},
	}
	got := hookProfileCompletionsForWorkspaceToken(items, "arlost", "")
	expected := []string{
		"cursor\tworkspace profile",
		"zed\tworkspace profile",
		"code\tglobal profile",
		"vim\tglobal profile",
	}
	if len(got) != len(expected) {
		t.Fatalf("unexpected count: got=%d want=%d values=%#v", len(got), len(expected), got)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("unexpected completion at %d: got=%q want=%q", i, got[i], expected[i])
		}
	}
}

func TestHookProfileCompletionsPrefixFilter(t *testing.T) {
	workspaceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspaceDir, ".wo"), []byte(`
[cursor]
command = "cursor ."
`), 0o644); err != nil {
		t.Fatal(err)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	globalPath, err := config.GlobalHookConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(globalPath, []byte(`
[code]
command = "code ."

[vim]
command = "vim ."
`), 0o644); err != nil {
		t.Fatal(err)
	}

	items := []model.Workspace{
		{RepoName: "arlost", Owner: "anishalle", Path: workspaceDir},
	}
	got := hookProfileCompletionsForWorkspaceToken(items, "arlost", "co")
	if len(got) != 1 || got[0] != "code\tglobal profile" {
		t.Fatalf("unexpected filtered completions: %#v", got)
	}
}
