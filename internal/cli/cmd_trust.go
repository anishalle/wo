package cli

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anishalle/wo/internal/config"
	"github.com/anishalle/wo/internal/db"
	"github.com/anishalle/wo/internal/model"
)

func newTrustCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trust",
		Short: "Manage workspace hook trust decisions",
	}
	cmd.AddCommand(newTrustListCmd())
	cmd.AddCommand(newTrustAllowCmd())
	cmd.AddCommand(newTrustDenyCmd())
	cmd.AddCommand(newTrustResetCmd())
	return cmd
}

func newTrustListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List trust decisions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCmd(cmd)
			recs, err := app.Store.ListTrust(cmd.Context())
			if err != nil {
				return err
			}
			if len(recs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no trust decisions recorded")
				return nil
			}
			for _, rec := range recs {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", rec.Decision, rec.Path, rec.UpdatedAt.Format("2006-01-02 15:04:05"))
			}
			return nil
		},
	}
}

func newTrustAllowCmd() *cobra.Command {
	return trustMutateCmd("allow", db.TrustAllow)
}

func newTrustDenyCmd() *cobra.Command {
	return trustMutateCmd("deny", db.TrustDeny)
}

func trustMutateCmd(name string, decision db.TrustDecision) *cobra.Command {
	return &cobra.Command{
		Use:               name + " <workspace>",
		Short:             strings.Title(name) + " hooks for a workspace",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeWorkspaceToken,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCmd(cmd)
			ws, err := resolveWorkspaceToken(cmd.Context(), app, args[0])
			if err != nil {
				return err
			}
			fingerprint, err := config.WorkspaceFingerprint(ws.Path)
			if err != nil {
				return err
			}
			if err := app.Store.SetTrust(cmd.Context(), ws.ID, decision, fingerprint); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", decision, ws.Path)
			return nil
		},
	}
}

func newTrustResetCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:               "reset [workspace]",
		Short:             "Reset trust for one workspace or all",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeWorkspaceToken,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCmd(cmd)
			if all {
				if err := app.Store.ResetAllTrust(cmd.Context()); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "trust reset for all workspaces")
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("provide workspace token or use --all")
			}
			ws, err := resolveWorkspaceToken(cmd.Context(), app, args[0])
			if err != nil {
				return err
			}
			if err := app.Store.ResetTrust(cmd.Context(), ws.ID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "trust reset %s\n", ws.Path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Reset all trust entries")
	return cmd
}

func resolveWorkspaceToken(ctx context.Context, app *App, token string) (model.Workspace, error) {
	var empty model.Workspace
	token = filepath.Clean(token)
	all, err := app.Store.ListWorkspaces(ctx)
	if err != nil {
		return empty, err
	}
	for _, ws := range all {
		if ws.Path == token {
			return ws, nil
		}
	}
	exact := make([]model.Workspace, 0)
	for _, ws := range all {
		if strings.EqualFold(ws.RepoName, token) {
			exact = append(exact, ws)
		}
	}
	if len(exact) == 1 {
		return exact[0], nil
	}
	if len(exact) > 1 {
		return empty, fmt.Errorf("workspace token %q is ambiguous", token)
	}
	return empty, fmt.Errorf("%w: workspace %q not found", sql.ErrNoRows, token)
}
