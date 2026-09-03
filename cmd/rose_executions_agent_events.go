package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var (
	roseExecutionAgentEventsAfterSeq int64
	roseExecutionAgentEventsLimit    int
)

type roseAgentEventPayload struct {
	Type         string `json:"type"`
	ToolName     string `json:"toolName"`
	Role         string `json:"role"`
	Preview      string `json:"preview"`
	ErrorMessage string `json:"errorMessage"`
	Reason       string `json:"reason"`
	StopReason   string `json:"stopReason"`
}

type roseAgentEvent struct {
	Seq            int64                 `json:"seq"`
	CreatedAt      string                `json:"createdAt"`
	AgentSessionID string                `json:"agentSessionId"`
	Payload        roseAgentEventPayload `json:"payload"`
}

type roseAgentEventsData struct {
	Events    []roseAgentEvent `json:"events"`
	LatestSeq *int64           `json:"latestSeq"`
}

var roseExecutionsAgentEventsCmd = &cobra.Command{
	Use:   "agent-events <execution-id>",
	Short: "List the ordered agent event timeline for a Rose execution",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoseExecutionsAgentEvents,
}

func init() {
	roseExecutionsCmd.AddCommand(roseExecutionsAgentEventsCmd)
	roseExecutionsAgentEventsCmd.Flags().Int64Var(&roseExecutionAgentEventsAfterSeq, "after-seq", -1, "Only events with a sequence greater than this value (≥-1)")
	roseExecutionsAgentEventsCmd.Flags().IntVar(&roseExecutionAgentEventsLimit, "limit", 500, "Maximum number of events (1-500)")
}

func runRoseExecutionsAgentEvents(cmd *cobra.Command, args []string) error {
	f := newFormatter(cmd)
	if !roseUUIDPattern.MatchString(args[0]) {
		return roseInvalidParameters(f, "execution-id must be a UUID")
	}
	if roseExecutionAgentEventsAfterSeq < -1 {
		return roseInvalidParameters(f, "--after-seq must be >= -1")
	}
	if roseExecutionAgentEventsLimit < 1 || roseExecutionAgentEventsLimit > 500 {
		return roseInvalidParameters(f, "--limit must be between 1 and 500")
	}

	query := url.Values{}
	query.Set("afterSeq", strconv.FormatInt(roseExecutionAgentEventsAfterSeq, 10))
	query.Set("limit", strconv.Itoa(roseExecutionAgentEventsLimit))
	apiResp, err := roseGet(cmd.Context(), f, "/rose/executions/"+args[0]+"/agent-events", query)
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

	var timeline roseAgentEventsData
	if err := json.Unmarshal(apiResp.Data, &timeline); err != nil {
		return fmt.Errorf("parsing agent events data: %w", err)
	}
	rows := make([][]string, len(timeline.Events))
	for i, event := range timeline.Events {
		created := event.CreatedAt
		rows[i] = []string{
			strconv.FormatInt(event.Seq, 10),
			roseTime(&created),
			event.AgentSessionID,
			event.Payload.Type,
			roseAgentEventDetail(event.Payload),
		}
	}
	f.PrintTable([]string{"SEQ", "CREATED", "SESSION", "TYPE", "DETAIL"}, rows)
	return nil
}

func roseAgentEventDetail(payload roseAgentEventPayload) string {
	parts := make([]string, 0, 2)
	if payload.ToolName != "" {
		parts = append(parts, payload.ToolName)
	}
	if payload.Role != "" {
		parts = append(parts, payload.Role)
	}
	for _, detail := range []string{payload.Preview, payload.ErrorMessage, payload.Reason, payload.StopReason} {
		if detail != "" {
			parts = append(parts, detail)
			break
		}
	}
	if len(parts) == 0 {
		return emDash
	}
	return strings.Join(parts, ": ")
}
