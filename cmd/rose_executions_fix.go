package cmd

import "github.com/spf13/cobra"

var (
	roseExecutionsFixFindingIDs  []string
	roseExecutionsFixIssueNumber int
)

type roseFixExecutionRequest struct {
	RepositoryID string   `json:"repo_id"`
	FindingIDs   []string `json:"finding_ids"`
	IssueNumber  *int     `json:"issue_number,omitempty"`
}

var roseExecutionsFixCmd = &cobra.Command{
	Use:   "fix <repository-id>",
	Short: "Start a Rose fix execution",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		f := newFormatter(cmd)
		if err := validateRoseRepositoryID(f, args[0]); err != nil {
			return err
		}
		if len(roseExecutionsFixFindingIDs) < 1 || len(roseExecutionsFixFindingIDs) > 10 {
			return roseInvalidParameters(f, "--finding-id must be repeated between 1 and 10 times")
		}
		for _, id := range roseExecutionsFixFindingIDs {
			if err := validateRoseFindingID(f, id); err != nil {
				return err
			}
		}
		request := roseFixExecutionRequest{RepositoryID: args[0], FindingIDs: roseExecutionsFixFindingIDs}
		if cmd.Flags().Changed("issue-number") {
			if roseExecutionsFixIssueNumber < 1 {
				return roseInvalidParameters(f, "--issue-number must be >= 1")
			}
			request.IssueNumber = &roseExecutionsFixIssueNumber
		}
		response, err := rosePost(cmd.Context(), f, "/rose/codebase/fix/execute", request)
		if err != nil {
			return err
		}
		return printRoseExecutionTrigger(cmd, f, response)
	},
}

func init() {
	roseExecutionsCmd.AddCommand(roseExecutionsFixCmd)
	roseExecutionsFixCmd.Flags().StringSliceVar(&roseExecutionsFixFindingIDs, "finding-id", nil, "Finding ID to fix (repeat 1-10 times)")
	roseExecutionsFixCmd.Flags().IntVar(&roseExecutionsFixIssueNumber, "issue-number", 0, "GitHub issue number associated with the fix")
}
