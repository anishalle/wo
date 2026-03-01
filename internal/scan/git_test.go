package scan

import "testing"

func TestOwnerFromRemote(t *testing.T) {
	cases := []struct {
		remote string
		owner  string
	}{
		{"git@github.com:hackutd/harp.git", "hackutd"},
		{"https://github.com/anishalle/wo.git", "anishalle"},
		{"http://example.com/team/repo.git", "team"},
		{"invalid", ""},
	}
	for _, tc := range cases {
		if got := ownerFromRemote(tc.remote); got != tc.owner {
			t.Fatalf("ownerFromRemote(%q)=%q want %q", tc.remote, got, tc.owner)
		}
	}
}

func TestOwnerFromPath(t *testing.T) {
	path := "/Users/ani/workspaces/github.com/hackutd/harp"
	if got := ownerFromPath(path); got != "hackutd" {
		t.Fatalf("ownerFromPath()=%q", got)
	}
}
