package cli

import (
	"strings"
	"testing"

	"github.com/anishalle/wo/internal/model"
)

func TestPosixShellScriptRestoresCWDWhenRequested(t *testing.T) {
	resp := model.ResolveResponse{
		Status:           model.ResolveOK,
		Path:             "/tmp/workspace",
		HookCommands:     []string{"echo hi"},
		ReturnToOriginal: true,
		ExitCode:         model.ExitOK,
	}
	script := posixShellScript(resp, "zsh")
	if !strings.Contains(script, "__wo_prev_dir=$(pwd)") {
		t.Fatalf("expected previous directory capture in script:\n%s", script)
	}
	if !strings.Contains(script, "cd -- \"$__wo_prev_dir\" || return 1") {
		t.Fatalf("expected restore directory command in script:\n%s", script)
	}
}

func TestFishShellScriptRestoresCWDWhenRequested(t *testing.T) {
	resp := model.ResolveResponse{
		Status:           model.ResolveOK,
		Path:             "/tmp/workspace",
		HookCommands:     []string{"echo hi"},
		ReturnToOriginal: true,
		ExitCode:         model.ExitOK,
	}
	script := fishShellScript(resp)
	if !strings.Contains(script, "set -l __wo_prev_dir (pwd)") {
		t.Fatalf("expected previous directory capture in fish script:\n%s", script)
	}
	if !strings.Contains(script, "cd -- \"$__wo_prev_dir\"; or return 1") {
		t.Fatalf("expected restore directory command in fish script:\n%s", script)
	}
}

func TestNormalizeResolveQueryProfileSplitsLegacyQuery(t *testing.T) {
	query, profile, err := normalizeResolveQueryProfile("harp cursor", "")
	if err != nil {
		t.Fatal(err)
	}
	if query != "harp" || profile != "cursor" {
		t.Fatalf("unexpected normalize output query=%q profile=%q", query, profile)
	}
}

func TestNormalizeResolveQueryProfileRejectsTooManyTokens(t *testing.T) {
	if _, _, err := normalizeResolveQueryProfile("one two three", ""); err == nil {
		t.Fatalf("expected error for too many tokens")
	}
}
