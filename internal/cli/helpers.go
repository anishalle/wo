package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/anishalle/wo/internal/config"
	"github.com/anishalle/wo/internal/model"
	"github.com/anishalle/wo/internal/scan"
)

const defaultScanDepth = 1

func isInteractive() bool {
	// Shell wrappers capture stdout for machine-readable responses, so treat stderr as the
	// interactive output stream for prompts/TUI compatibility checks.
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stderr.Fd()))
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
	targets, err := rescanTargets(ctx, app)
	if err != nil {
		return err
	}
	opts := scan.Options{
		Targets:        targets,
		FollowSymlinks: app.Config.Scan.FollowSymlink,
		Prune:          true,
	}
	_, err = scan.Run(ctx, app.Store, opts)
	return err
}

func rescanTargets(ctx context.Context, app *App) ([]scan.Target, error) {
	if app != nil && app.Store != nil {
		roots, err := app.Store.ListScanRoots(ctx)
		if err != nil {
			return nil, err
		}
		if len(roots) > 0 {
			out := make([]scan.Target, 0, len(roots))
			for _, root := range roots {
				out = append(out, scan.Target{Path: root.Path, Depth: root.Depth})
			}
			return out, nil
		}
	}
	defaultFile, err := config.ScanPathsFilePath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(defaultFile); err == nil {
		return loadScanTargetsFile(defaultFile, effectiveScanDepth(app.Config.Scan.DepthDefault))
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if len(app.Config.Roots) == 0 {
		return nil, fmt.Errorf("no saved scan roots and no scan file at %s", defaultFile)
	}
	out := make([]scan.Target, 0, len(app.Config.Roots))
	for _, root := range app.Config.Roots {
		path, err := config.ResolvePath(root, "")
		if err != nil {
			return nil, err
		}
		out = append(out, scan.Target{Path: path, Depth: effectiveScanDepth(app.Config.Scan.DepthDefault)})
	}
	return out, nil
}

func scanTargetsFromArgs(args []string, app *App) ([]scan.Target, error) {
	depthDefault := effectiveScanDepth(app.Config.Scan.DepthDefault)
	switch len(args) {
	case 0:
		path, err := config.ScanPathsFilePath()
		if err != nil {
			return nil, err
		}
		return loadScanTargetsFile(path, depthDefault)
	case 1:
		path, err := config.ResolvePath(args[0], "")
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(path)
		if err == nil && info.Mode().IsRegular() {
			return loadScanTargetsFile(path, depthDefault)
		}
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		return []scan.Target{{Path: path, Depth: depthDefault}}, nil
	case 2:
		path, err := config.ResolvePath(args[0], "")
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(path)
		if err == nil && info.Mode().IsRegular() {
			return nil, fmt.Errorf("scan file %q cannot be combined with a positional depth", args[0])
		}
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		depth, err := strconv.Atoi(strings.TrimSpace(args[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid scan depth %q", args[1])
		}
		if depth < 1 {
			return nil, fmt.Errorf("scan depth must be >= 1")
		}
		return []scan.Target{{Path: path, Depth: depth}}, nil
	default:
		return nil, fmt.Errorf("expected usage: wo scan [path|scan-file] [depth]")
	}
}

func loadScanTargetsFile(path string, defaultDepth int) ([]scan.Target, error) {
	resolvedPath, err := config.ResolvePath(path, "")
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("scan file not found: %s", resolvedPath)
		}
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	baseDir := filepath.Dir(resolvedPath)
	targets := make([]scan.Target, 0, len(lines))
	seen := map[string]int{}
	for idx, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		pathPart, depth, err := parseScanTargetLine(line, defaultDepth)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", resolvedPath, idx+1, err)
		}
		resolvedTarget, err := config.ResolvePath(pathPart, baseDir)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", resolvedPath, idx+1, err)
		}
		if seenIdx, ok := seen[resolvedTarget]; ok {
			targets[seenIdx].Depth = depth
			continue
		}
		seen[resolvedTarget] = len(targets)
		targets = append(targets, scan.Target{Path: resolvedTarget, Depth: depth})
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("scan file has no paths: %s", resolvedPath)
	}
	return targets, nil
}

func parseScanTargetLine(line string, defaultDepth int) (string, int, error) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", 0, fmt.Errorf("empty scan entry")
	}
	depth := defaultDepth
	pathPart := line
	last := fields[len(fields)-1]
	if parsedDepth, err := strconv.Atoi(last); err == nil {
		if parsedDepth < 1 {
			return "", 0, fmt.Errorf("scan depth must be >= 1")
		}
		depth = parsedDepth
		pathPart = strings.TrimSpace(strings.TrimSuffix(line, last))
	}
	if pathPart == "" {
		return "", 0, fmt.Errorf("scan path is required")
	}
	return pathPart, depth, nil
}

func effectiveScanDepth(configured int) int {
	if configured < 1 {
		return defaultScanDepth
	}
	return configured
}

func groupByOwner(workspaces []model.Workspace) map[string][]model.Workspace {
	out := map[string][]model.Workspace{}
	for _, ws := range workspaces {
		out[ws.Owner] = append(out[ws.Owner], ws)
	}
	return out
}
