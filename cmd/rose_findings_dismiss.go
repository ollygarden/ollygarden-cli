package cmd

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/spf13/cobra"
)

const roseDismissReasonMaxLength = 1000

var roseFindingsDismissReason string

type roseFindingDismissalRequest struct {
	Dismissed       bool    `json:"dismissed"`
	DismissedReason *string `json:"dismissed_reason,omitempty"`
}

type roseFindingDismissalResult struct {
	FindingID       string  `json:"finding_id"`
	Dismissed       bool    `json:"dismissed"`
	DismissedReason *string `json:"dismissed_reason"`
}

var roseFindingsDismissCmd = &cobra.Command{
	Use:   "dismiss <repository-id> <finding-id>",
	Short: "Dismiss a Rose finding",
	Args:  cobra.ExactArgs(2),
	RunE:  runRoseFindingsDismiss,
}

func init() {
	roseFindingsCmd.AddCommand(roseFindingsDismissCmd)
	roseFindingsDismissCmd.Flags().StringVar(&roseFindingsDismissReason, "reason", "", "Optional dismissal reason (maximum 1000 characters)")
}

func runRoseFindingsDismiss(cmd *cobra.Command, args []string) error {
	f := newFormatter(cmd)
	if err := validateRoseRepositoryID(f, args[0]); err != nil {
		return err
	}
	if err := validateRoseFindingID(f, args[1]); err != nil {
		return err
	}
	if utf8.RuneCountInString(roseFindingsDismissReason) > roseDismissReasonMaxLength {
		return roseInvalidParameters(f, "reason must be at most 1000 characters")
	}

	body := roseFindingDismissalRequest{Dismissed: true}
	if cmd.Flags().Changed("reason") {
		body.DismissedReason = &roseFindingsDismissReason
	}
	apiResp, err := rosePatch(cmd.Context(), f, "/rose/repositories/"+args[0]+"/findings/"+args[1], body)
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
	fmt.Fprintf(cmd.OutOrStdout(), "Finding %s is dismissed.\n", result.FindingID)
	return nil
}
