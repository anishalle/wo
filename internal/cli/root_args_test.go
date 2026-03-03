package cli

import "testing"

func TestParseWorkspaceProfileArgs(t *testing.T) {
	workspace, profile, err := parseWorkspaceProfileArgs([]string{"harp"})
	if err != nil {
		t.Fatal(err)
	}
	if workspace != "harp" || profile != "" {
		t.Fatalf("unexpected parse result: workspace=%q profile=%q", workspace, profile)
	}

	workspace, profile, err = parseWorkspaceProfileArgs([]string{"harp", "cursor"})
	if err != nil {
		t.Fatal(err)
	}
	if workspace != "harp" || profile != "cursor" {
		t.Fatalf("unexpected parse result: workspace=%q profile=%q", workspace, profile)
	}
}

func TestParseWorkspaceProfileArgsRejectsTooManyArgs(t *testing.T) {
	if _, _, err := parseWorkspaceProfileArgs([]string{"one", "two", "three"}); err == nil {
		t.Fatalf("expected parse error for too many args")
	}
}
