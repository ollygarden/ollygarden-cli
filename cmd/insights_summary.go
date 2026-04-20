package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

type insightSummaryDetail struct {
	InsightID   string `json:"insight_id"`
	Content     string `json:"content"`
	Model       string `json:"model"`
	GeneratedAt string `json:"generated_at"`
	Cached      bool   `json:"cached"`
}

var insightsSummaryCmd = &cobra.Command{
	Use:   "summary <insight-id>",
	Short: "Show AI-generated summary for an insight",
	Args:  cobra.ExactArgs(1),
	RunE:  runInsightsSummary,
}

func init() {
	insightsCmd.AddCommand(insightsSummaryCmd)
}

func runInsightsSummary(cmd *cobra.Command, args []string) error {
	insightID := args[0]

	c := NewClient()
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	resp, err := c.Get(cmd.Context(), "/insights/"+insightID+"/summary", nil)
	if err != nil {
		return fmt.Errorf("requesting insight summary: %w", err)
	}

	apiResp, err := client.ParseResponse(resp)
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok {
			var raw json.RawMessage
			if apiErr.ErrorResponse != nil {
				raw, _ = json.Marshal(apiErr.ErrorResponse)
			}
			f.PrintError(apiErr.Error(), raw)
		}
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

	var summary insightSummaryDetail
	if err := json.Unmarshal(apiResp.Data, &summary); err != nil {
		return fmt.Errorf("parsing summary data: %w", err)
	}

	emDash := "\u2014"
	valOrDash := func(s string) string {
		if s == "" {
			return emDash
		}
		return s
	}

	cachedStr := "no"
	if summary.Cached {
		cachedStr = "yes"
	}

	pairs := []output.KVPair{
		{Key: "Insight ID", Value: summary.InsightID},
		{Key: "Model", Value: valOrDash(summary.Model)},
		{Key: "Generated At", Value: valOrDash(summary.GeneratedAt)},
		{Key: "Cached", Value: cachedStr},
		{Key: "Summary", Value: valOrDash(summary.Content)},
	}

	f.PrintKeyValue(pairs)
	return nil
}
