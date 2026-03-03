package cli

import (
	"context"
	"fmt"
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
				hookPlan, err := hookSvc.CommandsForWorkspace(ctx, selected, hooks.Request{Clean: clean, Profile: profile, ForceGlobal: forceGlobal})
				if err != nil {
					return out, err
				}
				_ = app.Store.TouchUsage(ctx, selected.ID)
				out = model.ResolveResponse{Status: model.ResolveOK, Path: selected.Path, HookCommands: hookPlan.Commands, ReturnToOriginal: hookPlan.ReturnToOriginal, ExitCode: model.ExitOK}
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
					hookPlan, err := hookSvc.CommandsForWorkspace(ctx, candidate, hooks.Request{Clean: clean, Profile: profile, ForceGlobal: forceGlobal})
					if err != nil {
						return out, err
					}
					_ = app.Store.TouchUsage(ctx, candidate.ID)
					out = model.ResolveResponse{Status: model.ResolveOK, Path: candidate.Path, HookCommands: hookPlan.Commands, ReturnToOriginal: hookPlan.ReturnToOriginal, ExitCode: model.ExitOK}
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
					hookPlan, err := hookSvc.CommandsForWorkspace(ctx, picked, hooks.Request{Clean: clean, Profile: profile, ForceGlobal: forceGlobal})
					if err != nil {
						return out, err
					}
					_ = app.Store.TouchUsage(ctx, picked.ID)
					out = model.ResolveResponse{Status: model.ResolveOK, Path: picked.Path, HookCommands: hookPlan.Commands, ReturnToOriginal: hookPlan.ReturnToOriginal, ExitCode: model.ExitOK}
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

	if err := app.Store.TouchUsage(ctx, selected.ID); err != nil {
		return out, err
	}
	hookPlan, err := hookSvc.CommandsForWorkspace(ctx, selected, hooks.Request{Clean: clean, Profile: profile, ForceGlobal: forceGlobal})
	if err != nil {
		return out, err
	}
	out = model.ResolveResponse{
		Status:           model.ResolveOK,
		Path:             selected.Path,
		HookCommands:     hookPlan.Commands,
		ReturnToOriginal: hookPlan.ReturnToOriginal,
		ExitCode:         model.ExitOK,
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
	if err := app.Store.TouchUsage(ctx, picked.ID); err != nil {
		return out, err
	}
	hookSvc := hooks.New(app.Store, app.Config)
	hookPlan, err := hookSvc.CommandsForWorkspace(ctx, picked, hooks.Request{Clean: clean})
	if err != nil {
		return out, err
	}
	out = model.ResolveResponse{
		Status:           model.ResolveOK,
		Path:             picked.Path,
		HookCommands:     hookPlan.Commands,
		ReturnToOriginal: hookPlan.ReturnToOriginal,
		ExitCode:         model.ExitOK,
	}
	return out, nil
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
