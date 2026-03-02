package cli

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anishalle/wo/internal/model"
)

func completeWorkspaceQuery(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
