package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/anishalle/wo/internal/scan"
)

func newScanCmd() *cobra.Command {
	var roots []string
	var depth int
	var followSymlinks bool
	var prune bool
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan filesystem roots and index workspaces",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCmd(cmd)
			ctx := cmd.Context()
			roots = normalizeRoots(roots, app.Config.Roots)
			if depth <= 0 {
				depth = app.Config.Scan.DepthDefault
			}
			if !cmd.Flags().Changed("follow-symlinks") {
				followSymlinks = app.Config.Scan.FollowSymlink
			}
			result, err := scan.Run(ctx, app.Store, scan.Options{
				Roots:          roots,
				Depth:          depth,
				FollowSymlinks: followSymlinks,
				Prune:          prune,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "scanned roots=%d discovered=%d updated=%d removed=%d\n", len(roots), result.Discovered, result.Updated, result.Removed)
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&roots, "root", nil, "Root directory to scan (repeatable)")
	cmd.Flags().IntVar(&depth, "depth", 1, "Scan depth")
	cmd.Flags().BoolVar(&followSymlinks, "follow-symlinks", false, "Follow symlinked directories")
	cmd.Flags().BoolVar(&prune, "prune", false, "Prune indexed workspaces no longer found")
	return cmd
}
