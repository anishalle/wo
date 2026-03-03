package cli

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anishalle/wo/internal/config"
	"github.com/anishalle/wo/internal/model"
)

func completeWorkspaceQuery(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	app, cleanup := appForCompletion(cmd)
	if cleanup != nil {
		defer cleanup()
	}
	if app == nil || app.Store == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ws, err := app.Store.ListWorkspaces(cmd.Context())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	if len(args) == 1 {
		out := hookProfileCompletionsForWorkspaceToken(ws, args[0], toComplete)
		return out, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
	}
	return workspaceNameCompletions(ws, toComplete), cobra.ShellCompDirectiveNoFileComp
}

func completeWorkspaceToken(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	app, cleanup := appForCompletion(cmd)
	if cleanup != nil {
		defer cleanup()
	}
	if app == nil || app.Store == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ws, err := app.Store.ListWorkspaces(cmd.Context())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return workspaceTokenCompletions(ws, toComplete), cobra.ShellCompDirectiveNoFileComp
}

func workspaceNameCompletions(workspaces []model.Workspace, toComplete string) []string {
	query := strings.ToLower(strings.TrimSpace(toComplete))
	type candidate struct {
		name string
		desc string
	}
	seen := map[string]candidate{}
	for _, ws := range workspaces {
		name := strings.TrimSpace(ws.RepoName)
		if name == "" {
			continue
		}
		if query != "" && !strings.HasPrefix(strings.ToLower(name), query) {
			continue
		}
		key := strings.ToLower(name)
		desc := ws.Owner + " · " + ws.Path
		if existing, ok := seen[key]; ok {
			if existing.desc != desc {
				seen[key] = candidate{name: name, desc: "multiple workspaces"}
			}
			continue
		}
		seen[key] = candidate{name: name, desc: desc}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		c := seen[k]
		out = append(out, c.name+"\t"+c.desc)
	}
	return out
}

func workspaceTokenCompletions(workspaces []model.Workspace, toComplete string) []string {
	query := strings.ToLower(strings.TrimSpace(toComplete))
	tokens := map[string]string{}
	for _, ws := range workspaces {
		repo := strings.TrimSpace(ws.RepoName)
		path := strings.TrimSpace(ws.Path)
		if repo != "" {
			if query == "" || strings.HasPrefix(strings.ToLower(repo), query) {
				tokens[repo] = ws.Owner + " · " + ws.Path
			}
		}
		if path != "" {
			if query == "" || strings.HasPrefix(strings.ToLower(path), query) {
				tokens[path] = "path"
			}
		}
	}
	keys := make([]string, 0, len(tokens))
	for token := range tokens {
		keys = append(keys, token)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, token := range keys {
		out = append(out, token+"\t"+tokens[token])
	}
	return out
}

func hookProfileCompletionsForWorkspaceToken(workspaces []model.Workspace, workspaceToken, toComplete string) []string {
	selected := workspacesForToken(workspaces, workspaceToken)
	if len(selected) == 0 {
		return nil
	}
	query := strings.ToLower(strings.TrimSpace(toComplete))

	internal := map[string]string{}
	sort.SliceStable(selected, func(i, j int) bool {
		return strings.ToLower(selected[i].Path) < strings.ToLower(selected[j].Path)
	})
	for _, ws := range selected {
		cfg, exists, err := config.LoadWorkspaceFile(ws.Path)
		if err != nil || !exists {
			continue
		}
		for name := range cfg.Profiles {
			if !matchesCompletionPrefix(name, query) {
				continue
			}
			key := strings.ToLower(name)
			if _, seen := internal[key]; !seen {
				internal[key] = name
			}
		}
	}

	global := map[string]string{}
	globalCfg, globalExists, err := config.LoadGlobalHookFile()
	if err == nil && globalExists {
		for name := range globalCfg.Profiles {
			if !matchesCompletionPrefix(name, query) {
				continue
			}
			key := strings.ToLower(name)
			if _, shadowed := internal[key]; shadowed {
				continue
			}
			if _, seen := global[key]; !seen {
				global[key] = name
			}
		}
	}

	internalNames := mapValuesSorted(internal)
	globalNames := mapValuesSorted(global)
	out := make([]string, 0, len(internalNames)+len(globalNames))
	for _, name := range internalNames {
		out = append(out, name+"\tworkspace profile")
	}
	for _, name := range globalNames {
		out = append(out, name+"\tglobal profile")
	}
	return out
}

func workspacesForToken(workspaces []model.Workspace, token string) []model.Workspace {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	byPath := make([]model.Workspace, 0)
	for _, ws := range workspaces {
		if ws.Path == token {
			byPath = append(byPath, ws)
		}
	}
	if len(byPath) > 0 {
		return byPath
	}
	byRepo := make([]model.Workspace, 0)
	for _, ws := range workspaces {
		if strings.EqualFold(ws.RepoName, token) {
			byRepo = append(byRepo, ws)
		}
	}
	return byRepo
}

func matchesCompletionPrefix(name, query string) bool {
	if query == "" {
		return true
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(name)), query)
}

func mapValuesSorted(in map[string]string) []string {
	out := make([]string, 0, len(in))
	for _, value := range in {
		out = append(out, value)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
}

func appForCompletion(cmd *cobra.Command) (*App, func()) {
	app := appFromCmd(cmd)
	if app != nil {
		return app, nil
	}
	tmpApp, err := NewApp(cmd.Context())
	if err != nil {
		return nil, nil
	}
	return tmpApp, func() {
		_ = tmpApp.Close()
	}
}
