package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/anishalle/wo/internal/shell"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init <zsh|bash|fish>",
		Short: "Print shell integration script",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			script, err := shell.Script(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), script)
			return nil
		},
	}
	return cmd
}
