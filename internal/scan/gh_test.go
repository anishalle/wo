package scan

import "testing"

func TestRepoSlugFromRemote(t *testing.T) {
	cases := []struct {
		name     string
		remote   string
		wantSlug string
		wantOK   bool
	}{
		{"ssh with .git", "git@github.com:anishalle/wo.git", "anishalle/wo", true},
		{"ssh without .git", "git@github.com:anishalle/wo", "anishalle/wo", true},
		{"https with .git", "https://github.com/anishalle/wo.git", "anishalle/wo", true},
		{"https without .git", "https://github.com/anishalle/wo", "anishalle/wo", true},
		{"http", "http://github.com/anishalle/wo.git", "anishalle/wo", true},
		{"ssh scheme", "ssh://git@github.com/anishalle/wo.git", "anishalle/wo", true},
		{"trailing slash", "https://github.com/anishalle/wo/", "anishalle/wo", true},
		{"non-github ssh", "git@gitlab.com:anishalle/wo.git", "", false},
		{"non-github https", "https://bitbucket.org/anishalle/wo.git", "", false},
		{"empty", "", "", false},
		{"garbage", "not a url", "", false},
		{"missing repo", "git@github.com:anishalle", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			slug, ok := repoSlugFromRemote(c.remote)
			if ok != c.wantOK || slug != c.wantSlug {
				t.Fatalf("repoSlugFromRemote(%q) = (%q, %v), want (%q, %v)", c.remote, slug, ok, c.wantSlug, c.wantOK)
			}
		})
	}
}
