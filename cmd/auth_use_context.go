package cmd

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/auth"
	"github.com/ollygarden/ollygarden-cli/internal/config"
	"github.com/spf13/cobra"
)

var authUseContextCmd = &cobra.Command{
	Use:   "use-context <name>",
	Short: "Set the current-context to a saved context by name",
	Args:  cobra.ExactArgs(1),
	RunE:  runAuthUseContext,
}

func init() {
	authCmd.AddCommand(authUseContextCmd)
}

func runAuthUseContext(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		var ue *config.UnreadableError
		if errors.As(err, &ue) {
			return auth.ErrConfigUnreadable(ue.Path, ue.Err)
		}
		return auth.ErrConfigUnreadable("", err)
	}
	if _, ok := cfg.Contexts[name]; !ok {
		return auth.ErrContextNotFound(name)
	}
	cfg.CurrentContext = name
	if err := config.Write(cfg); err != nil {
		var we *config.WriteFailedError
		if errors.As(err, &we) {
			return auth.ErrConfigWriteFailed(we.Path, we.Err)
		}
		return auth.ErrConfigWriteFailed("", err)
	}

	if jsonMode {
		out := map[string]any{
			"data": map[string]any{"current_context": name},
			"meta": map[string]any{},
		}
		raw, _ := json.Marshal(out)
		fmt.Fprintln(cmd.OutOrStdout(), string(raw))
		return nil
	}
	if quiet {
		return nil
	}
	fmt.Fprintf(cmd.OutOrStdout(), "✓ Switched to context %q.\n", name)
	return nil
}
