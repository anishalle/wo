package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func newManCmd() *cobra.Command {
	var outDir string
	cmd := &cobra.Command{
		Use:    "man",
		Hidden: true,
		Short:  "Generate man pages",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if outDir == "" {
				outDir = "./man"
			}
			if err := os.MkdirAll(outDir, 0o755); err != nil {
				return err
			}
			header := &doc.GenManHeader{
				Title:   "WO",
				Section: "1",
			}
			return doc.GenManTree(cmd.Root(), header, filepath.Clean(outDir))
		},
	}
	cmd.Flags().StringVar(&outDir, "dir", "", "Output directory for man pages")
	return cmd
}
