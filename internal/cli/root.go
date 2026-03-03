package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var version = "dev"

type appCtxKey struct{}

func NewRootCmd(v string) *cobra.Command {
	if v != "" {
		version = v
	}
	root := &cobra.Command{
		Use:               "wo",
		Short:             "Fast workspace manager",
		SilenceUsage:      true,
		SilenceErrors:     true,
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: completeWorkspaceQuery,
		Version:           version,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if !commandNeedsApp(cmd) {
				return nil
			}
			ctx := cmd.Context()
			if appFromCtx(ctx) != nil {
				return nil
			}
			app, err := NewApp(ctx)
			if err != nil {
				return err
			}
			cmd.SetContext(context.WithValue(ctx, appCtxKey{}, app))
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, _ []string) error {
			if !commandNeedsApp(cmd) {
				return nil
			}
			app := appFromCmd(cmd)
			if app != nil {
				return app.Close()
			}
			return nil
		},
		RunE: runRoot,
	}

	root.Flags().Bool("clean", false, "Skip hooks")
	root.Flags().Bool("pick", false, "Always open picker")
	root.Flags().Bool("global", false, "Use profile from global config.wo")

	root.AddCommand(newScanCmd())
	root.AddCommand(newListCmd())
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newTrustCmd())
	root.AddCommand(newInitCmd())
	root.AddCommand(newCompletionCmd())
	root.AddCommand(newManCmd())
	root.AddCommand(newResolveCmd())
	root.AddCommand(newShellApplyCmd())
	root.AddCommand(newBrowseCmd())
	return root
}

func commandNeedsApp(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	switch cmd.Name() {
	case "init", "completion", "man", "__shell-apply", "help":
		return false
	}
	if cmd.Name() == "wo" {
		if f := cmd.Flags().Lookup("help"); f != nil && f.Changed {
			return false
		}
		if f := cmd.Flags().Lookup("version"); f != nil && f.Changed {
			return false
		}
	}
	return true
}

func Execute(v string) int {
	cmd := NewRootCmd(v)
	if err := cmd.Execute(); err != nil {
		if !errors.Is(err, errSilentExit) {
			fmt.Fprintf(cmd.ErrOrStderr(), "wo: %v\n", err)
		}
		if code := exitCode(err); code != 0 {
			return code
		}
		return 2
	}
	return 0
}

func runRoot(cmd *cobra.Command, args []string) error {
	app := appFromCmd(cmd)
	ctx := cmd.Context()
	clean, _ := cmd.Flags().GetBool("clean")
	pick, _ := cmd.Flags().GetBool("pick")
	forceGlobal, _ := cmd.Flags().GetBool("global")
	if forceGlobal && len(args) < 2 {
		return fmt.Errorf("--global requires a profile argument")
	}

	if err := maybePromptRescan(ctx, app); err != nil {
		return err
	}

	if len(args) == 0 {
		resp, err := runBrowseFlow(ctx, app, clean)
		if err != nil {
			return err
		}
		if resp.Status != "ok" {
			return exitErr{code: resp.ExitCode, err: errSilentExit}
		}
		fmt.Fprintln(cmd.OutOrStdout(), resp.Path)
		return nil
	}
	workspace, profile, err := parseWorkspaceProfileArgs(args)
	if err != nil {
		return err
	}
	resp, err := runResolveFlow(ctx, app, workspace, profile, clean, pick, forceGlobal)
	if err != nil {
		return err
	}
	if resp.Status != "ok" {
		if resp.Message != "" {
			fmt.Fprintln(cmd.ErrOrStderr(), "wo:", resp.Message)
		}
		return exitErr{code: resp.ExitCode, err: errSilentExit}
	}
	fmt.Fprintln(cmd.OutOrStdout(), resp.Path)
	return nil
}

func parseWorkspaceProfileArgs(args []string) (string, string, error) {
	if len(args) == 0 {
		return "", "", nil
	}
	if len(args) > 2 {
		return "", "", fmt.Errorf("expected usage: wo [workspace] [profile]")
	}
	workspace := strings.TrimSpace(args[0])
	if workspace == "" {
		return "", "", fmt.Errorf("workspace argument is required")
	}
	if len(args) == 1 {
		return workspace, "", nil
	}
	profile := strings.TrimSpace(args[1])
	if profile == "" {
		return "", "", fmt.Errorf("profile argument cannot be empty")
	}
	return workspace, profile, nil
}

func appFromCmd(cmd *cobra.Command) *App {
	if cmd == nil {
		return nil
	}
	if app := appFromCtx(cmd.Context()); app != nil {
		return app
	}
	return appFromCtx(cmd.Root().Context())
}

func appFromCtx(ctx context.Context) *App {
	if ctx == nil {
		return nil
	}
	app, _ := ctx.Value(appCtxKey{}).(*App)
	return app
}

var errSilentExit = errors.New("silent exit")

type exitErr struct {
	code int
	err  error
}

func (e exitErr) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e exitErr) Unwrap() error {
	return e.err
}

func exitCode(err error) int {
	var ex exitErr
	if errors.As(err, &ex) {
		return ex.code
	}
	return 0
}
