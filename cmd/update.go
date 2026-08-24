package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/selfupdate"
	"github.com/spf13/cobra"
)

var (
	updateForce    bool
	runSelfUpdater = selfupdate.Run
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update ollygarden to the latest stable release",
	Long: `Check GitHub for the latest stable ollygarden release and install it
when it is newer than the running version. The release checksum and staged
binary version are verified before the current executable is replaced.

Package-manager installations exposed through a symlink must be updated with
the package manager that installed them. Development builds cannot self-update.`,
	Args: cobra.NoArgs,
	RunE: runUpdate,
}

func init() {
	updateCmd.Flags().BoolVar(&updateForce, "force", false, "Reinstall the latest release when it is already current")
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, _ []string) error {
	var progress func(string)
	if !quiet && !jsonMode {
		progress = func(message string) {
			fmt.Fprintln(cmd.ErrOrStderr(), message)
		}
	}

	result, err := runSelfUpdater(cmd.Context(), selfupdate.Options{
		CurrentVersion: version,
		Force:          updateForce,
		Progress:       progress,
	})
	if err != nil {
		message := fmt.Sprintf("updating ollygarden: %v", err)
		if jsonMode {
			envelope := map[string]any{
				"error": map[string]any{
					"code":    "UPDATE_FAILED",
					"message": message,
				},
				"meta": map[string]any{},
			}
			raw, _ := json.Marshal(envelope)
			fmt.Fprintln(cmd.ErrOrStderr(), string(raw))
			return &reportedError{err: err}
		}
		return fmt.Errorf("Error: %s", message)
	}

	if jsonMode {
		envelope := map[string]any{
			"data": result,
			"meta": map[string]any{},
		}
		raw, _ := json.Marshal(envelope)
		fmt.Fprintln(cmd.OutOrStdout(), string(raw))
		return nil
	}
	if quiet {
		return nil
	}

	switch {
	case result.Updated && result.CurrentVersion == result.LatestVersion:
		fmt.Fprintf(cmd.OutOrStdout(), "Reinstalled ollygarden v%s.\n", result.LatestVersion)
	case result.Updated:
		fmt.Fprintf(cmd.OutOrStdout(), "Updated ollygarden from v%s to v%s.\n", result.CurrentVersion, result.LatestVersion)
	case result.CurrentIsNewer:
		fmt.Fprintf(cmd.OutOrStdout(), "Current ollygarden v%s is newer than the latest stable release v%s; no update applied.\n", result.CurrentVersion, result.LatestVersion)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "ollygarden is already up to date (v%s).\n", result.CurrentVersion)
	}
	return nil
}

// reportedError marks an error whose JSON representation was already written
// by the command, preventing Execute from emitting a second human-format line.
type reportedError struct {
	err error
}

func (e *reportedError) Error() string { return e.err.Error() }
