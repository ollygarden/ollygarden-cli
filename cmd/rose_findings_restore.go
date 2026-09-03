package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var roseFindingsRestoreCmd = &cobra.Command{
	Use:   "restore <repository-id> <finding-id>",
	Short: "Restore a dismissed Rose finding",
	Args:  cobra.ExactArgs(2),
	RunE:  runRoseFindingsRestore,
}

func init() {
	roseFindingsCmd.AddCommand(roseFindingsRestoreCmd)
}

func runRoseFindingsRestore(cmd *cobra.Command, args []string) error {
	f := newFormatter(cmd)
	if err := validateRoseRepositoryID(f, args[0]); err != nil {
		return err
	}
	if err := validateRoseFindingID(f, args[1]); err != nil {
		return err
	}
	apiResp, err := rosePatch(cmd.Context(), f, "/rose/repositories/"+args[0]+"/findings/"+args[1], roseFindingDismissalRequest{Dismissed: false})
	if err != nil {
		return err
	}
	if f.IsJSON() {
		raw, _ := json.Marshal(apiResp)
		f.PrintJSON(raw)
		return nil
	}
	if f.IsQuiet() {
		return nil
	}
	var result roseFindingDismissalResult
	if err := json.Unmarshal(apiResp.Data, &result); err != nil {
		return fmt.Errorf("parsing finding dismissal data: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Finding %s is active.\n", result.FindingID)
	return nil
}
