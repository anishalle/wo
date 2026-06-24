package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/anishalle/wo/internal/scan"
)

func newScanCmd() *cobra.Command {
	var followSymlinks bool
	var prune bool
	var noDescriptions bool
	cmd := &cobra.Command{
		Use:   "scan [path|scan-file] [depth]",
		Short: "Scan filesystem roots and index workspaces",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFromCmd(cmd)
			ctx := cmd.Context()
			if !cmd.Flags().Changed("follow-symlinks") {
				followSymlinks = app.Config.Scan.FollowSymlink
			}
			fetchDescriptions := app.Config.Scan.FetchDescriptions
			if noDescriptions {
				fetchDescriptions = false
			}
			targets, err := scanTargetsFromArgs(args, app)
			if err != nil {
				return err
			}
			result, err := scan.Run(ctx, app.Store, scan.Options{
				Targets:           targets,
				FollowSymlinks:    followSymlinks,
				Prune:             prune,
				FetchDescriptions: fetchDescriptions,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "scanned roots=%d discovered=%d updated=%d removed=%d\n", len(targets), result.Discovered, result.Updated, result.Removed)
			return nil
		},
	}
	cmd.Flags().BoolVar(&followSymlinks, "follow-symlinks", false, "Follow symlinked directories")
	cmd.Flags().BoolVar(&prune, "prune", false, "Prune indexed workspaces no longer found")
	cmd.Flags().BoolVar(&noDescriptions, "no-descriptions", false, "Skip fetching repo descriptions from gh")
	return cmd
}
