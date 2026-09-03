package cmd

import "github.com/spf13/cobra"

var roseExecutionsInstrumentationType string

type roseInstrumentationExecutionRequest struct {
	RepositoryID string `json:"repo_id"`
	Type         string `json:"type"`
}

var roseExecutionsInstrumentCmd = &cobra.Command{
	Use:   "instrument <repository-id>",
	Short: "Start a Rose instrumentation execution",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		f := newFormatter(cmd)
		if err := validateRoseRepositoryID(f, args[0]); err != nil {
			return err
		}
		if roseExecutionsInstrumentationType != "minimum" && roseExecutionsInstrumentationType != "general" {
			return roseInvalidParameters(f, "--type must be one of: minimum, general")
		}
		response, err := rosePost(cmd.Context(), f, "/rose/codebase/instrumentation/execute", roseInstrumentationExecutionRequest{RepositoryID: args[0], Type: roseExecutionsInstrumentationType})
		if err != nil {
			return err
		}
		return printRoseExecutionTrigger(cmd, f, response)
	},
}

func init() {
	roseExecutionsCmd.AddCommand(roseExecutionsInstrumentCmd)
	roseExecutionsInstrumentCmd.Flags().StringVar(&roseExecutionsInstrumentationType, "type", "minimum", "Instrumentation type (minimum or general)")
}
