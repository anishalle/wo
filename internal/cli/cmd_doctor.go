package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run health checks for wo",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCmd(cmd)
			ctx := cmd.Context()
			checks := []struct {
				Name string
				Err  error
				Soft bool
			}{
				{Name: "config", Err: app.Config.Normalize()},
				{Name: "database", Err: pingDB(ctx, app)},
				{Name: "fzf", Err: checkBinary("fzf"), Soft: true},
			}
			for _, root := range app.Config.Roots {
				if _, err := os.Stat(root); err != nil {
					checks = append(checks, struct {
						Name string
						Err  error
						Soft bool
					}{Name: "root:" + root, Err: err, Soft: true})
				}
			}
			failed := 0
			for _, c := range checks {
				if c.Err != nil {
					if c.Soft {
						fmt.Fprintf(cmd.OutOrStdout(), "[WARN] %s: %v\n", c.Name, c.Err)
					} else {
						failed++
						fmt.Fprintf(cmd.OutOrStdout(), "[FAIL] %s: %v\n", c.Name, c.Err)
					}
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "[ OK ] %s\n", c.Name)
				}
			}
			if failed > 0 {
				return exitErr{code: 2, err: errSilentExit}
			}
			return nil
		},
	}
}

func pingDB(ctx context.Context, app *App) error {
	_, err := app.Store.ListWorkspaces(ctx)
	return err
}

func checkBinary(name string) error {
	_, err := exec.LookPath(name)
	return err
}
