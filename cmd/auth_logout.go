package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ollygarden/ollygarden-cli/internal/auth"
	"github.com/ollygarden/ollygarden-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	authLogoutContext string
	authLogoutAll     bool
	authLogoutConfirm bool
)

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove a saved context (or all of them)",
	Long: `Remove a saved context from disk.

  Default                 Remove the current-context and unset the pointer.
  --context NAME          Remove a specific context.
  --all                   Remove every context. Requires --confirm in non-TTY mode.

When the last context is removed, the config file is deleted entirely.`,
	Args: cobra.NoArgs,
	RunE: runAuthLogout,
}

func init() {
	// --context here SHADOWS the persistent flag at this command's scope.
	// We use the same flag name because it carries the same intent ("name a
	// context"); the --context value is read from this command's own flag set.
	authLogoutCmd.Flags().StringVar(&authLogoutContext, "context", "", "Name of the context to remove")
	authLogoutCmd.Flags().BoolVar(&authLogoutAll, "all", false, "Remove every saved context")
	authLogoutCmd.Flags().BoolVar(&authLogoutConfirm, "confirm", false, "Required for --all in non-interactive mode")
	authCmd.AddCommand(authLogoutCmd)
}

func runAuthLogout(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		var ue *config.UnreadableError
		if errors.As(err, &ue) {
			return auth.ErrConfigUnreadable(ue.Path, ue.Err)
		}
		return auth.ErrConfigUnreadable("", err)
	}

	switch {
	case authLogoutAll:
		if !authLogoutConfirm && !isTerminal(os.Stdin) {
			return auth.ErrConfirmRequired()
		}
		// On a TTY without --confirm, prompt y/N (default No).
		if !authLogoutConfirm && isTerminal(os.Stdin) {
			fmt.Fprintf(cmd.ErrOrStderr(), "Remove all %d saved contexts? [y/N]: ", len(cfg.Contexts))
			b := make([]byte, 1)
			_, _ = os.Stdin.Read(b)
			fmt.Fprintln(cmd.ErrOrStderr())
			if !strings.EqualFold(string(b), "y") {
				return fmt.Errorf("aborted")
			}
		}
		cfg.Contexts = map[string]*config.Context{}
		cfg.CurrentContext = ""
	case authLogoutContext != "":
		if _, ok := cfg.Contexts[authLogoutContext]; !ok {
			return auth.ErrContextNotFound(authLogoutContext)
		}
		delete(cfg.Contexts, authLogoutContext)
		if cfg.CurrentContext == authLogoutContext {
			cfg.CurrentContext = ""
		}
	default:
		if cfg.CurrentContext == "" {
			return auth.ErrNoCredentials()
		}
		removed := cfg.CurrentContext
		delete(cfg.Contexts, removed)
		cfg.CurrentContext = ""
		_ = removed
	}

	if err := config.Write(cfg); err != nil {
		var we *config.WriteFailedError
		if errors.As(err, &we) {
			return auth.ErrConfigWriteFailed(we.Path, we.Err)
		}
		return auth.ErrConfigWriteFailed("", err)
	}

	if jsonMode {
		out := map[string]any{
			"data": map[string]any{
				"removed_all":     authLogoutAll,
				"current_context": cfg.CurrentContext,
				"remaining":       sortedContextNames(cfg.Contexts),
			},
			"meta": map[string]any{},
		}
		raw, _ := json.Marshal(out)
		fmt.Fprintln(cmd.OutOrStdout(), string(raw))
		return nil
	}

	if quiet {
		return nil
	}

	switch {
	case authLogoutAll:
		fmt.Fprintln(cmd.OutOrStdout(), "✓ Removed all saved contexts.")
	case authLogoutContext != "":
		fmt.Fprintf(cmd.OutOrStdout(), "✓ Removed context %q.\n", authLogoutContext)
	default:
		fmt.Fprintln(cmd.OutOrStdout(), "✓ Logged out.")
	}

	if !authLogoutAll && len(cfg.Contexts) > 0 && cfg.CurrentContext == "" {
		names := sortedContextNames(cfg.Contexts)
		fmt.Fprintf(cmd.OutOrStdout(),
			"No current context set. Available: %s. Activate with `ollygarden auth use-context NAME`.\n",
			strings.Join(names, ", "),
		)
	}
	return nil
}

func sortedContextNames(m map[string]*config.Context) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
