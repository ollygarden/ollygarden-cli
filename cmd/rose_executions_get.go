package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

type roseExecutionDetailData struct {
	Execution    roseExecutionSummary `json:"execution"`
	CurrentPhase *roseExecutionPhase  `json:"currentPhase"`
	Phases       []roseExecutionPhase `json:"phases"`
	Running      bool                 `json:"running"`
	LastSeenAt   *string              `json:"lastSeenAt"`
}

var roseExecutionsGetCmd = &cobra.Command{
	Use:   "get <execution-id>",
	Short: "Show a Rose execution's status and phases",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoseExecutionsGet,
}

func init() {
	roseExecutionsCmd.AddCommand(roseExecutionsGetCmd)
}

func runRoseExecutionsGet(cmd *cobra.Command, args []string) error {
	f := newFormatter(cmd)

	apiResp, err := roseGet(cmd.Context(), f, "/rose/executions/"+args[0], nil)
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

	var detail roseExecutionDetailData
	if err := json.Unmarshal(apiResp.Data, &detail); err != nil {
		return fmt.Errorf("parsing execution data: %w", err)
	}

	ex := detail.Execution
	currentPhase := emDash
	if detail.CurrentPhase != nil {
		currentPhase = detail.CurrentPhase.Label + " (" + detail.CurrentPhase.State + ")"
	}
	started := ex.StartedAt
	commit := ex.CommitSHA
	pairs := []output.KVPair{
		{Key: "ID", Value: ex.ID},
		{Key: "Type", Value: ex.ExecutionType},
		{Key: "Status", Value: strOrDash(ex.Status)},
		{Key: "Trigger", Value: strOrDash(ex.TriggerSource)},
		{Key: "Repository", Value: ex.RepoOwner + "/" + ex.RepoName + " (" + ex.RepositoryID + ")"},
		{Key: "Ref", Value: orDash(ex.Ref)},
		{Key: "Commit", Value: shortSHA(&commit)},
		{Key: "Started", Value: roseTime(&started)},
		{Key: "Completed", Value: roseTime(ex.CompletedAt)},
		{Key: "Running", Value: boolYesNo(detail.Running)},
		{Key: "Last seen", Value: roseTime(detail.LastSeenAt)},
		{Key: "Current phase", Value: currentPhase},
	}
	f.PrintKeyValue(pairs)

	if len(detail.Phases) > 0 {
		fmt.Fprintln(cmd.OutOrStdout())
		headers := []string{"PHASE", "STATUS", "STARTED", "COMPLETED"}
		rows := make([][]string, len(detail.Phases))
		for i, ph := range detail.Phases {
			rows[i] = []string{ph.Label, ph.State, roseTime(ph.StartedAt), roseTime(ph.CompletedAt)}
		}
		f.PrintTable(headers, rows)
	}
	return nil
}
