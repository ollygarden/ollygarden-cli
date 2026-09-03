package cmd

import "github.com/spf13/cobra"

type roseReviewExecutionRequest struct {
	RepositoryID string `json:"repo_id"`
}

var roseExecutionsReviewCmd = &cobra.Command{
	Use:   "review <repository-id>",
	Short: "Start a Rose instrumentation review",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		f := newFormatter(cmd)
		if err := validateRoseRepositoryID(f, args[0]); err != nil {
			return err
		}
		response, err := rosePost(cmd.Context(), f, "/rose/codebase/review/execute", roseReviewExecutionRequest{RepositoryID: args[0]})
		if err != nil {
			return err
		}
		return printRoseExecutionTrigger(cmd, f, response)
	},
}

func init() { roseExecutionsCmd.AddCommand(roseExecutionsReviewCmd) }
