package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/term"

	"github.com/anishalle/wo/internal/model"
	"github.com/anishalle/wo/internal/scan"
)

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func promptYesNo(question string, defaultYes bool) (bool, error) {
	if !isInteractive() {
		return defaultYes, nil
	}
	fmt.Fprint(os.Stderr, question)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return defaultYes, nil
	}
	if line == "y" || line == "yes" {
		return true, nil
	}
	if line == "n" || line == "no" {
		return false, nil
	}
	return defaultYes, nil
}

func maybePromptRescan(ctx context.Context, app *App) error {
	workspaces, err := app.Store.ListWorkspaces(ctx)
	if err != nil {
		return err
	}
	if len(workspaces) < 10 {
		return nil
	}
	missing := 0
	for _, ws := range workspaces {
		if _, err := os.Stat(ws.Path); err != nil {
			missing++
		}
	}
	ratio := float64(missing) / float64(len(workspaces))
	if ratio < 0.10 {
		return nil
	}
	ok, err := promptYesNo("wo: index looks stale. Run wo scan now? (Y/n) ", true)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	opts := scan.Options{
		Roots:          app.Config.Roots,
		Depth:          app.Config.Scan.DepthDefault,
		FollowSymlinks: app.Config.Scan.FollowSymlink,
		Prune:          true,
	}
	_, err = scan.Run(ctx, app.Store, opts)
	return err
}

func normalizeRoots(roots []string, defaults []string) []string {
	if len(roots) == 0 {
		roots = append(roots, defaults...)
	}
	set := map[string]struct{}{}
	out := make([]string, 0, len(roots))
	for _, root := range roots {
		if root == "" {
			continue
		}
		if strings.HasPrefix(root, "~/") || root == "~" {
			home, err := os.UserHomeDir()
			if err == nil {
				if root == "~" {
					root = home
				} else {
					root = filepath.Join(home, strings.TrimPrefix(root, "~/"))
				}
			}
		}
		root = filepath.Clean(root)
		if _, ok := set[root]; ok {
			continue
		}
		set[root] = struct{}{}
		out = append(out, root)
	}
	sort.Strings(out)
	return out
}

func groupByOwner(workspaces []model.Workspace) map[string][]model.Workspace {
	out := map[string][]model.Workspace{}
	for _, ws := range workspaces {
		out[ws.Owner] = append(out[ws.Owner], ws)
	}
	return out
}
