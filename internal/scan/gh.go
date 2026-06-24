package scan

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	ghOnce      sync.Once
	ghAvailable bool
)

// hasGH reports whether the `gh` CLI is available on PATH. The result is cached
// for the lifetime of the process.
func hasGH() bool {
	ghOnce.Do(func() {
		_, err := exec.LookPath("gh")
		ghAvailable = err == nil
	})
	return ghAvailable
}

// repoSlugFromRemote returns the "owner/repo" slug for a github.com remote URL,
// or ok=false for non-github remotes or unparseable URLs. It handles SSH
// (git@github.com:owner/repo.git) and HTTP(S) (https://github.com/owner/repo.git)
// forms.
func repoSlugFromRemote(remoteURL string) (slug string, ok bool) {
	remote := strings.TrimSuffix(strings.TrimSpace(remoteURL), ".git")
	if remote == "" {
		return "", false
	}

	var path string
	switch {
	case strings.HasPrefix(remote, "git@"):
		parts := strings.SplitN(remote, ":", 2)
		if len(parts) != 2 {
			return "", false
		}
		host := strings.TrimPrefix(parts[0], "git@")
		if !isGitHubHost(host) {
			return "", false
		}
		path = parts[1]
	case strings.HasPrefix(remote, "https://"), strings.HasPrefix(remote, "http://"), strings.HasPrefix(remote, "ssh://"):
		withoutScheme := remote[strings.Index(remote, "//")+2:]
		// Drop any user@ prefix on ssh:// URLs.
		if at := strings.Index(withoutScheme, "@"); at != -1 {
			withoutScheme = withoutScheme[at+1:]
		}
		slash := strings.Index(withoutScheme, "/")
		if slash == -1 {
			return "", false
		}
		host := withoutScheme[:slash]
		if !isGitHubHost(host) {
			return "", false
		}
		path = withoutScheme[slash+1:]
	default:
		return "", false
	}

	segs := strings.Split(strings.Trim(path, "/"), "/")
	if len(segs) < 2 {
		return "", false
	}
	owner := segs[len(segs)-2]
	repo := segs[len(segs)-1]
	if owner == "" || repo == "" {
		return "", false
	}
	return owner + "/" + repo, true
}

func isGitHubHost(host string) bool {
	// Strip a possible :port suffix.
	if i := strings.Index(host, ":"); i != -1 {
		host = host[:i]
	}
	return strings.EqualFold(host, "github.com")
}

// fetchDescriptions populates the Description field on candidates that have a
// github.com remote, using a bounded pool of gh workers. It is a no-op when gh
// is unavailable. The seen map is mutated in place.
func fetchDescriptions(ctx context.Context, seen map[string]candidate, paths []string) {
	if !hasGH() {
		return
	}

	type job struct {
		path string
		slug string
	}
	jobs := make([]job, 0, len(paths))
	for _, path := range paths {
		cand := seen[path]
		if slug, ok := repoSlugFromRemote(cand.workspace.RemoteURL); ok {
			jobs = append(jobs, job{path: path, slug: slug})
		}
	}
	if len(jobs) == 0 {
		return
	}

	const workers = 8
	jobCh := make(chan job)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				desc := fetchDescription(ctx, j.slug)
				if desc == "" {
					continue
				}
				mu.Lock()
				cand := seen[j.path]
				cand.workspace.Description = desc
				seen[j.path] = cand
				mu.Unlock()
			}
		}()
	}
	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)
	wg.Wait()
}

// fetchDescription returns the GitHub description for a "owner/repo" slug via the
// gh CLI. It returns "" on any error or timeout.
func fetchDescription(ctx context.Context, slug string) string {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "repo", "view", slug, "--json", "description", "-q", ".description")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(out.String())
}
