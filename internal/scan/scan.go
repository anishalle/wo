package scan

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anishalle/wo/internal/config"
	"github.com/anishalle/wo/internal/db"
	"github.com/anishalle/wo/internal/model"
)

type Options struct {
	Roots          []string
	Depth          int
	FollowSymlinks bool
	Prune          bool
}

type Result struct {
	Discovered int
	Updated    int
	Removed    int64
}

type candidate struct {
	workspace model.Workspace
	aliases   []string
}

var ignoreDirNames = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	".cache":       {},
	".direnv":      {},
	"vendor":       {},
}

func Run(ctx context.Context, store *db.Store, opts Options) (Result, error) {
	var out Result
	if opts.Depth < 1 {
		opts.Depth = 1
	}
	if len(opts.Roots) == 0 {
		return out, fmt.Errorf("no roots to scan")
	}
	seen := map[string]candidate{}
	for _, root := range opts.Roots {
		root = filepath.Clean(root)
		if err := store.SaveScanRoot(ctx, root, opts.Depth); err != nil {
			return out, err
		}
		rootInfo, err := os.Stat(root)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return out, err
		}
		if !rootInfo.IsDir() {
			continue
		}
		rootCand, err := inspectDirectory(root)
		if err != nil {
			return out, err
		}
		if rootCand != nil {
			seen[root] = *rootCand
		}
		err = walkRoot(root, opts.Depth, opts.FollowSymlinks, seen)
		if err != nil {
			return out, err
		}
	}

	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		cand := seen[path]
		out.Discovered++
		if _, err := store.UpsertWorkspace(ctx, cand.workspace, cand.aliases); err != nil {
			return out, err
		}
		out.Updated++
	}
	if opts.Prune {
		removed, err := store.DeleteMissingWorkspaces(ctx, paths)
		if err != nil {
			return out, err
		}
		out.Removed = removed
	}
	return out, nil
}

type walkNode struct {
	path  string
	depth int
}

func walkRoot(root string, maxDepth int, followSymlinks bool, seen map[string]candidate) error {
	queue := []walkNode{{path: root, depth: 0}}
	visitedReal := map[string]struct{}{}
	realRoot, err := filepath.EvalSymlinks(root)
	if err == nil {
		visitedReal[realRoot] = struct{}{}
	}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		if node.depth >= maxDepth {
			continue
		}
		entries, err := os.ReadDir(node.path)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			child := filepath.Join(node.path, entry.Name())
			isDir := entry.IsDir()
			isSymlink := entry.Type()&os.ModeSymlink != 0
			if isSymlink {
				if !followSymlinks {
					continue
				}
				if st, err := os.Stat(child); err == nil && st.IsDir() {
					isDir = true
				}
			}
			if !isDir {
				continue
			}
			depth := node.depth + 1
			if shouldSkipDir(entry.Name(), depth, maxDepth) {
				continue
			}
			if depth > maxDepth {
				continue
			}
			if followSymlinks {
				if real, err := filepath.EvalSymlinks(child); err == nil {
					if _, ok := visitedReal[real]; ok {
						continue
					}
					visitedReal[real] = struct{}{}
				}
			}
			cand, err := inspectDirectory(child)
			if err == nil && cand != nil {
				seen[child] = *cand
			}
			if depth < maxDepth {
				queue = append(queue, walkNode{path: child, depth: depth})
			}
		}
	}
	return nil
}

func shouldSkipDir(name string, depth, maxDepth int) bool {
	if depth <= 0 {
		return false
	}
	if _, ok := ignoreDirNames[name]; ok {
		return true
	}
	if strings.HasPrefix(name, ".") {
		return true
	}
	if depth > maxDepth {
		return true
	}
	return false
}

func depthOf(rel string) int {
	if rel == "." || rel == "" {
		return 0
	}
	return len(strings.Split(rel, string(filepath.Separator)))
}

func inspectDirectory(path string) (*candidate, error) {
	gitInfo, err := os.Stat(filepath.Join(path, ".git"))
	hasGit := err == nil && gitInfo.IsDir()
	woCfg, hasWO, err := config.LoadWorkspaceFile(path)
	if err != nil {
		return nil, err
	}
	if !hasGit && !hasWO {
		return nil, nil
	}
	repoName := filepath.Base(path)
	if woCfg.Name != "" {
		repoName = woCfg.Name
	}
	remoteURL := ""
	owner := ""
	if hasGit {
		remoteURL, owner = extractGitRemote(path)
	}
	if woCfg.Owner != "" {
		owner = woCfg.Owner
	}
	if owner == "" {
		owner = ownerFromPath(path)
	}
	if owner == "" {
		owner = "local"
	}
	c := candidate{
		workspace: model.Workspace{
			Path:      path,
			RepoName:  repoName,
			Owner:     owner,
			Source:    sourceName(hasGit, hasWO),
			HasGit:    hasGit,
			HasWO:     hasWO,
			RemoteURL: remoteURL,
		},
	}
	if woCfg.Name != "" {
		c.aliases = append(c.aliases, filepath.Base(path))
	}
	return &c, nil
}

func sourceName(hasGit, hasWO bool) string {
	switch {
	case hasGit && hasWO:
		return "git+wo"
	case hasGit:
		return "git"
	case hasWO:
		return "wo"
	default:
		return "unknown"
	}
}

func ownerFromPath(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i := 0; i < len(parts)-2; i++ {
		if strings.Contains(parts[i], ".") {
			return parts[i+1]
		}
	}
	return ""
}
