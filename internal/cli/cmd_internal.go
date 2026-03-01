package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anishalle/wo/internal/model"
)

func newResolveCmd() *cobra.Command {
	var query string
	var clean bool
	var pick bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:    "__resolve",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(query) == "" {
				return fmt.Errorf("--query is required")
			}
			app := appFromCmd(cmd)
			ctx := cmd.Context()
			if err := maybePromptRescan(ctx, app); err != nil {
				return err
			}
			resp, err := runResolveFlow(ctx, app, query, clean, pick)
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				if err := enc.Encode(resp); err != nil {
					return err
				}
				if resp.Status != model.ResolveOK {
					return exitErr{code: resp.ExitCode, err: errSilentExit}
				}
				return nil
			}
			if resp.Status != model.ResolveOK {
				if resp.Message != "" {
					fmt.Fprintln(cmd.ErrOrStderr(), resp.Message)
				}
				return exitErr{code: resp.ExitCode, err: errSilentExit}
			}
			fmt.Fprintln(cmd.OutOrStdout(), resp.Path)
			return nil
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "workspace query")
	cmd.Flags().BoolVar(&clean, "clean", false, "Skip enter hooks")
	cmd.Flags().BoolVar(&pick, "pick", false, "Force picker")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	return cmd
}

func newBrowseCmd() *cobra.Command {
	var asJSON bool
	var clean bool
	cmd := &cobra.Command{
		Use:    "__browse",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := appFromCmd(cmd)
			ctx := cmd.Context()
			if err := maybePromptRescan(ctx, app); err != nil {
				return err
			}
			resp, err := runBrowseFlow(ctx, app, clean)
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				if err := enc.Encode(resp); err != nil {
					return err
				}
				if resp.Status != model.ResolveOK {
					return exitErr{code: resp.ExitCode, err: errSilentExit}
				}
				return nil
			}
			if resp.Status != model.ResolveOK {
				if resp.Message != "" {
					fmt.Fprintln(cmd.ErrOrStderr(), resp.Message)
				}
				return exitErr{code: resp.ExitCode, err: errSilentExit}
			}
			fmt.Fprintln(cmd.OutOrStdout(), resp.Path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	cmd.Flags().BoolVar(&clean, "clean", false, "Skip hooks")
	return cmd
}

func newShellApplyCmd() *cobra.Command {
	var shell string
	cmd := &cobra.Command{
		Use:    "__shell-apply",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			b, err := io.ReadAll(bufio.NewReader(os.Stdin))
			if err != nil {
				return err
			}
			if len(strings.TrimSpace(string(b))) == 0 {
				return nil
			}
			var resp model.ResolveResponse
			if err := json.Unmarshal(b, &resp); err != nil {
				return err
			}
			script := shellScriptFromResolve(resp, shell)
			fmt.Fprint(cmd.OutOrStdout(), script)
			return nil
		},
	}
	cmd.Flags().StringVar(&shell, "shell", "zsh", "target shell")
	return cmd
}

func shellScriptFromResolve(resp model.ResolveResponse, shell string) string {
	if shell == "fish" {
		return fishShellScript(resp)
	}
	return posixShellScript(resp, shell)
}

func posixShellScript(resp model.ResolveResponse, shell string) string {
	var sb strings.Builder
	emitReturn := func(code int) {
		sb.WriteString("return ")
		sb.WriteString(fmt.Sprintf("%d", code))
		sb.WriteString("\n")
	}
	if resp.Status != model.ResolveOK {
		msg := resp.Message
		if msg == "" {
			msg = "command failed"
		}
		sb.WriteString("printf '%s\\n' ")
		sb.WriteString(shellQuote("wo: "+msg, shell))
		sb.WriteString(" >&2\n")
		emitReturn(resp.ExitCode)
		return sb.String()
	}
	sb.WriteString("cd -- ")
	sb.WriteString(shellQuote(resp.Path, shell))
	sb.WriteString(" || return 1\n")
	for _, cmd := range resp.HookCommands {
		sb.WriteString(cmd)
		sb.WriteString("\n")
		sb.WriteString("__wo_hook_status=$?\n")
		sb.WriteString("if [ $__wo_hook_status -ne 0 ]; then printf '%s\\n' ")
		sb.WriteString(shellQuote("wo: hook failed: "+cmd, shell))
		sb.WriteString(" >&2; fi\n")
	}
	emitReturn(0)
	return sb.String()
}

func fishShellScript(resp model.ResolveResponse) string {
	var sb strings.Builder
	emitReturn := func(code int) {
		sb.WriteString(fmt.Sprintf("return %d\n", code))
	}
	if resp.Status != model.ResolveOK {
		msg := resp.Message
		if msg == "" {
			msg = "command failed"
		}
		sb.WriteString("printf '%s\\n' ")
		sb.WriteString(shellQuote("wo: "+msg, "fish"))
		sb.WriteString(" >&2\n")
		emitReturn(resp.ExitCode)
		return sb.String()
	}
	sb.WriteString("cd -- ")
	sb.WriteString(shellQuote(resp.Path, "fish"))
	sb.WriteString("; or return 1\n")
	for _, cmd := range resp.HookCommands {
		sb.WriteString(cmd)
		sb.WriteString("\n")
		sb.WriteString("set __wo_hook_status $status\n")
		sb.WriteString("if test $__wo_hook_status -ne 0\n")
		sb.WriteString("  printf '%s\\n' ")
		sb.WriteString(shellQuote("wo: hook failed: "+cmd, "fish"))
		sb.WriteString(" >&2\n")
		sb.WriteString("end\n")
	}
	emitReturn(0)
	return sb.String()
}

func shellQuote(s string, shell string) string {
	escaped := strings.ReplaceAll(s, "'", `'"'"'`)
	switch shell {
	case "fish":
		return "'" + escaped + "'"
	default:
		return "'" + escaped + "'"
	}
}
