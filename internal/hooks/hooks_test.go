package hooks

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/anishalle/wo/internal/config"
	"github.com/anishalle/wo/internal/db"
	"github.com/anishalle/wo/internal/model"
)

func TestCommandsForWorkspacePrefersWorkspaceProfile(t *testing.T) {
	ctx := context.Background()
	store, ws := setupWorkspace(t, `
[enter]
commands = ["echo startup"]

[cursor]
command = "echo workspace"
`)
	defer store.Close()

	writeGlobalHookFile(t, `
[cursor]
command = "echo global"
`)

	allowWorkspaceHooks(t, ctx, store, ws.Path, ws.ID)
	svc := New(store, config.DefaultConfig())
	plan, err := svc.CommandsForWorkspace(ctx, ws, Request{Profile: "cursor"})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(plan.Commands))
	}
	if plan.Commands[0] != "echo startup" {
		t.Fatalf("unexpected first command: %q", plan.Commands[0])
	}
	if plan.Commands[1] != "echo workspace" {
		t.Fatalf("unexpected second command: %q", plan.Commands[1])
	}
	if plan.ReturnToOriginal {
		t.Fatalf("expected to stay in workspace by default")
	}
}

func TestCommandsForWorkspaceForceGlobalRunsStartupAndGlobalProfile(t *testing.T) {
	ctx := context.Background()
	store, ws := setupWorkspace(t, `
[enter]
commands = ["echo startup"]

[cursor]
command = "echo workspace"
`)
	defer store.Close()

	writeGlobalHookFile(t, `
[enter]
commands = ["echo global-startup"]

[cursor]
command = "echo global"
chdir = false
`)

	allowWorkspaceHooks(t, ctx, store, ws.Path, ws.ID)
	svc := New(store, config.DefaultConfig())
	plan, err := svc.CommandsForWorkspace(ctx, ws, Request{Profile: "cursor", ForceGlobal: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Commands) != 2 {
		t.Fatalf("expected startup + global profile, got %d commands", len(plan.Commands))
	}
	if plan.Commands[0] != "echo startup" {
		t.Fatalf("unexpected first command: %q", plan.Commands[0])
	}
	if plan.Commands[1] != "echo global" {
		t.Fatalf("unexpected second command: %q", plan.Commands[1])
	}
	if !plan.ReturnToOriginal {
		t.Fatalf("expected ReturnToOriginal for chdir=false global profile")
	}
}

func TestCommandsForWorkspaceFallsBackToGlobalProfile(t *testing.T) {
	ctx := context.Background()
	store, ws := setupWorkspace(t, `
[enter]
commands = ["echo startup"]
`)
	defer store.Close()

	writeGlobalHookFile(t, `
[cursor]
command = "echo global"
`)

	allowWorkspaceHooks(t, ctx, store, ws.Path, ws.ID)
	svc := New(store, config.DefaultConfig())
	plan, err := svc.CommandsForWorkspace(ctx, ws, Request{Profile: "cursor"})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Commands) != 2 {
		t.Fatalf("expected startup + global profile, got %d", len(plan.Commands))
	}
	if plan.Commands[0] != "echo startup" || plan.Commands[1] != "echo global" {
		t.Fatalf("unexpected commands: %#v", plan.Commands)
	}
}

func TestCommandsForWorkspaceMissingProfileErrors(t *testing.T) {
	ctx := context.Background()
	store, ws := setupWorkspace(t, `
[enter]
commands = ["echo startup"]
`)
	defer store.Close()

	setTempHome(t)
	allowWorkspaceHooks(t, ctx, store, ws.Path, ws.ID)
	svc := New(store, config.DefaultConfig())
	if _, err := svc.CommandsForWorkspace(ctx, ws, Request{Profile: "cursor"}); err == nil {
		t.Fatalf("expected missing profile error")
	}
}

func TestCommandsForWorkspaceTrustDenySkipsWorkspaceHooksButRunsGlobal(t *testing.T) {
	ctx := context.Background()
	store, ws := setupWorkspace(t, `
[enter]
commands = ["echo startup"]
`)
	defer store.Close()

	writeGlobalHookFile(t, `
[cursor]
command = "echo global"
`)

	denyWorkspaceHooks(t, ctx, store, ws.Path, ws.ID)
	svc := New(store, config.DefaultConfig())
	plan, err := svc.CommandsForWorkspace(ctx, ws, Request{Profile: "cursor", ForceGlobal: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Commands) != 1 {
		t.Fatalf("expected only global command, got %d", len(plan.Commands))
	}
	if plan.Commands[0] != "echo global" {
		t.Fatalf("unexpected command: %q", plan.Commands[0])
	}
}

func TestCommandsForWorkspaceCleanSkipsAllHooks(t *testing.T) {
	ctx := context.Background()
	store, ws := setupWorkspace(t, `
[enter]
commands = ["echo startup"]

[cursor]
command = "echo workspace"
`)
	defer store.Close()

	allowWorkspaceHooks(t, ctx, store, ws.Path, ws.ID)
	svc := New(store, config.DefaultConfig())
	plan, err := svc.CommandsForWorkspace(ctx, ws, Request{Profile: "cursor", Clean: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Commands) != 0 {
		t.Fatalf("expected clean mode to skip hooks, got %#v", plan.Commands)
	}
}

func setupWorkspace(t *testing.T, woContent string) (*db.Store, model.Workspace) {
	t.Helper()
	tmp := t.TempDir()
	wsPath := filepath.Join(tmp, "harp")
	writeFile(t, filepath.Join(wsPath, ".wo"), woContent)

	store, err := db.Open(filepath.Join(tmp, "data"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	id, err := store.UpsertWorkspace(ctx, model.Workspace{
		Path:     wsPath,
		RepoName: "harp",
		Owner:    "hackutd",
		Source:   "wo",
		HasWO:    true,
	}, nil)
	if err != nil {
		store.Close()
		t.Fatal(err)
	}
	return store, model.Workspace{
		ID:       id,
		Path:     wsPath,
		RepoName: "harp",
		Owner:    "hackutd",
		Source:   "wo",
		HasWO:    true,
	}
}

func allowWorkspaceHooks(t *testing.T, ctx context.Context, store *db.Store, workspacePath string, workspaceID int64) {
	t.Helper()
	fingerprint, err := config.WorkspaceFingerprint(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetTrust(ctx, workspaceID, db.TrustAllow, fingerprint); err != nil {
		t.Fatal(err)
	}
}

func denyWorkspaceHooks(t *testing.T, ctx context.Context, store *db.Store, workspacePath string, workspaceID int64) {
	t.Helper()
	fingerprint, err := config.WorkspaceFingerprint(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetTrust(ctx, workspaceID, db.TrustDeny, fingerprint); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setTempHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
}

func writeGlobalHookFile(t *testing.T, content string) {
	t.Helper()
	setTempHome(t)
	path, err := config.GlobalHookConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, content)
}
