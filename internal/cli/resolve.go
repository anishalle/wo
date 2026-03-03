package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/anishalle/wo/internal/hooks"
	"github.com/anishalle/wo/internal/model"
	"github.com/anishalle/wo/internal/resolve"
	"github.com/anishalle/wo/internal/tui"
)

func runResolveFlow(ctx context.Context, app *App, query, profile string, clean, forcePick, forceGlobal bool) (model.ResolveResponse, error) {
	var out model.ResolveResponse
	service := resolve.New(app.Store, app.Config)
	hookSvc := hooks.New(app.Store, app.Config)
	result, err := service.Resolve(ctx, query)
	if err != nil {
		return out, err
	}

	if len(result.Matches) == 0 {
		if result.Correction != nil {
			ok, err := promptYesNo(fmt.Sprintf("wo: you typed %s. Did you mean %s? (y/N) ", query, result.Correction.Workspace.RepoName), false)
			if err != nil {
				return out, err
			}
			if ok {
				selected := result.Correction.Workspace
				resolved, ok, err := finalizeSelectedWorkspace(ctx, app, hookSvc, selected, clean, profile, forceGlobal)
				if err != nil {
					return out, err
				}
				out = resolved
				if !ok {
					return out, nil
				}
				return out, nil
			}
		}
		if result.Stage == "fuzzy" && len(result.TopSuggestions) > 0 {
			// Fuzzy matches always require explicit user confirmation.
			if len(result.TopSuggestions) == 1 {
				candidate := result.TopSuggestions[0].Workspace
				ok, err := promptYesNo(fmt.Sprintf("wo: did you mean %s? (y/N) ", candidate.RepoName), false)
				if err != nil {
					return out, err
				}
				if ok {
					resolved, ok, err := finalizeSelectedWorkspace(ctx, app, hookSvc, candidate, clean, profile, forceGlobal)
					if err != nil {
						return out, err
					}
					out = resolved
					if !ok {
						return out, nil
					}
					return out, nil
				}
			} else if isInteractive() {
				candidates := make([]model.Workspace, 0, len(result.TopSuggestions))
				for _, m := range result.TopSuggestions {
					candidates = append(candidates, m.Workspace)
				}
				picked, ok, err := pickWorkspaceInteractive(candidates, app.Config.Search.Backend, "Select workspace (fuzzy match)")
				if err != nil {
					return out, err
				}
				if ok {
					resolved, ok, err := finalizeSelectedWorkspace(ctx, app, hookSvc, picked, clean, profile, forceGlobal)
					if err != nil {
						return out, err
					}
					out = resolved
					if !ok {
						return out, nil
					}
					return out, nil
				}
				out.Status = model.ResolveUserCancel
				out.Message = "selection cancelled"
				out.ExitCode = model.ExitCanceled
				return out, nil
			}
		}
		out.Status = model.ResolveNoMatch
		out.Message = noMatchMessage(query, result.TopSuggestions)
		out.ExitCode = model.ExitNoMatch
		out.Candidates = toCandidates(result.TopSuggestions)
		return out, nil
	}

	matches := result.Matches
	var selected model.Workspace
	if len(matches) == 1 && !forcePick {
		selected = matches[0].Workspace
	} else {
		list := make([]model.Workspace, 0, len(matches))
		for _, m := range matches {
			list = append(list, m.Workspace)
		}
		picked, ok, err := pickWorkspaceInteractive(list, app.Config.Search.Backend, "Select workspace")
		if err != nil {
			return out, err
		}
		if !ok {
			out.Status = model.ResolveUserCancel
			out.Message = "selection cancelled"
			out.ExitCode = model.ExitCanceled
			return out, nil
		}
		selected = picked
	}

	resolved, ok, err := finalizeSelectedWorkspace(ctx, app, hookSvc, selected, clean, profile, forceGlobal)
	if err != nil {
		return out, err
	}
	out = resolved
	if !ok {
		return out, nil
	}
	return out, nil
}

func runBrowseFlow(ctx context.Context, app *App, clean bool) (model.ResolveResponse, error) {
	var out model.ResolveResponse
	workspaces, err := app.Store.ListWorkspaces(ctx)
	if err != nil {
		return out, err
	}
	if len(workspaces) == 0 {
		out.Status = model.ResolveNoMatch
		out.Message = "no workspaces indexed. Run: wo scan"
		out.ExitCode = model.ExitNoMatch
		return out, nil
	}
	picked, ok, err := pickWorkspaceInteractive(workspaces, app.Config.Search.Backend, "wo projects", true)
	if err != nil {
		return out, err
	}
	if !ok {
		out.Status = model.ResolveUserCancel
		out.Message = "selection cancelled"
		out.ExitCode = model.ExitCanceled
		return out, nil
	}
	hookSvc := hooks.New(app.Store, app.Config)
	resolved, ok, err := finalizeSelectedWorkspace(ctx, app, hookSvc, picked, clean, "", false)
	if err != nil {
		return out, err
	}
	out = resolved
	if !ok {
		return out, nil
	}
	return out, nil
}

func finalizeSelectedWorkspace(ctx context.Context, app *App, hookSvc *hooks.Service, selected model.Workspace, clean bool, profile string, forceGlobal bool) (model.ResolveResponse, bool, error) {
	var out model.ResolveResponse
	exists, removed, err := ensureWorkspaceExistsOrPrune(ctx, app, selected)
	if err != nil {
		return out, false, err
	}
	if !exists {
		out.Status = model.ResolveNoMatch
		out.ExitCode = model.ExitNoMatch
		if removed {
			out.Message = fmt.Sprintf("removed missing workspace from index: %s", selected.Path)
		} else {
			out.Message = fmt.Sprintf("workspace path does not exist: %s", selected.Path)
		}
		return out, false, nil
	}
	if err := app.Store.TouchUsage(ctx, selected.ID); err != nil {
		return out, false, err
	}
	hookPlan, err := hookSvc.CommandsForWorkspace(ctx, selected, hooks.Request{Clean: clean, Profile: profile, ForceGlobal: forceGlobal})
	if err != nil {
		return out, false, err
	}
	out = model.ResolveResponse{
		Status:           model.ResolveOK,
		Path:             selected.Path,
		HookCommands:     hookPlan.Commands,
		ReturnToOriginal: hookPlan.ReturnToOriginal,
		ExitCode:         model.ExitOK,
	}
	return out, true, nil
}

func ensureWorkspaceExistsOrPrune(ctx context.Context, app *App, ws model.Workspace) (bool, bool, error) {
	if _, err := os.Stat(ws.Path); err == nil {
		return true, false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, false, err
	}
	ok, err := promptYesNo(fmt.Sprintf("wo: workspace path is missing: %s. Remove from index? (y/N) ", ws.Path), false)
	if err != nil {
		return false, false, err
	}
	if !ok {
		return false, false, nil
	}
	deleted, err := app.Store.DeleteWorkspaceByID(ctx, ws.ID)
	if err != nil {
		return false, false, err
	}
	return false, deleted, nil
}

func pickWorkspaceInteractive(candidates []model.Workspace, backend, title string, grouped ...bool) (model.Workspace, bool, error) {
	group := false
	if len(grouped) > 0 {
		group = grouped[0]
	}
	if backend == "fzf" && tui.HasFZF() {
		prompt := "wo> "
		if group {
			prompt = "wo browse> "
		}
		return tui.PickWithFZF(candidates, prompt)
	}
	return tui.PickWorkspace(title, candidates, group)
}

func toCandidates(matches []resolve.Match) []model.ResolveCandidate {
	out := make([]model.ResolveCandidate, 0, len(matches))
	for _, m := range matches {
		out = append(out, model.ResolveCandidate{
			ID:       m.Workspace.ID,
			Path:     m.Workspace.Path,
			RepoName: m.Workspace.RepoName,
			Owner:    m.Workspace.Owner,
			Score:    m.Score,
		})
	}
	return out
}

func noMatchMessage(query string, suggestions []resolve.Match) string {
	if len(suggestions) == 0 {
		return fmt.Sprintf("no workspace found for %q", query)
	}
	names := make([]string, 0, minInt(len(suggestions), 3))
	for i := 0; i < len(suggestions) && i < 3; i++ {
		names = append(names, suggestions[i].Workspace.RepoName)
	}
	return fmt.Sprintf("no workspace found for %q. maybe: %s", query, strings.Join(names, ", "))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
