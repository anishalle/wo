package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anishalle/wo/internal/model"
)

func newListCmd() *cobra.Command {
	var owner string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List indexed workspaces",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCmd(cmd)
			ctx := cmd.Context()
			if err := maybePromptRescan(ctx, app); err != nil {
				return err
			}
			var (
				ws  []model.Workspace
				err error
			)
			if owner != "" {
				ws, err = app.Store.ListWorkspacesByOwner(ctx, owner)
			} else {
				ws, err = app.Store.ListWorkspaces(ctx)
			}
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(ws)
			}
			if len(ws) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no workspaces indexed")
				return nil
			}
			owners := map[string][]string{}
			for _, w := range ws {
				owners[w.Owner] = append(owners[w.Owner], fmt.Sprintf("%s\t%s", w.RepoName, w.Path))
			}
			ownerKeys := make([]string, 0, len(owners))
			for k := range owners {
				ownerKeys = append(ownerKeys, k)
			}
			sort.Strings(ownerKeys)
			for _, k := range ownerKeys {
				fmt.Fprintln(cmd.OutOrStdout(), k)
				fmt.Fprintln(cmd.OutOrStdout(), strings.Repeat("-", len(k)))
				sort.Strings(owners[k])
				for _, line := range owners[k] {
					fmt.Fprintln(cmd.OutOrStdout(), line)
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Filter by owner")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output JSON")
	return cmd
}
