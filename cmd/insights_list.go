package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	insightsListLimit      int
	insightsListOffset     int
	insightsListServiceID  string
	insightsListStatus     string
	insightsListSignalType string
	insightsListImpact     string
	insightsListDateFrom   string
	insightsListDateTo     string
	insightsListSort       string
)

type insightsListItem struct {
	ID          string              `json:"id"`
	Status      string              `json:"status"`
	ServiceName string              `json:"service_name"`
	InsightType *insightTypeCompact `json:"insight_type"`
	DetectedTS  string              `json:"detected_ts"`
}

var insightsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List insights",
	Args:  cobra.NoArgs,
	RunE:  runInsightsList,
}

func init() {
	insightsCmd.AddCommand(insightsListCmd)
	insightsListCmd.Flags().IntVar(&insightsListLimit, "limit", 20, "Maximum number of results (1-100)")
	insightsListCmd.Flags().IntVar(&insightsListOffset, "offset", 0, "Number of results to skip (≥0)")
	insightsListCmd.Flags().StringVar(&insightsListServiceID, "service-id", "", "Filter by service ID")
	insightsListCmd.Flags().StringVar(&insightsListStatus, "status", "", "Filter by status (comma-separated: active, archived, muted)")
	insightsListCmd.Flags().StringVar(&insightsListSignalType, "signal-type", "", "Filter by signal type (trace, metric, log)")
	insightsListCmd.Flags().StringVar(&insightsListImpact, "impact", "", "Filter by impact (comma-separated: Critical, Important, Normal, Low)")
	insightsListCmd.Flags().StringVar(&insightsListDateFrom, "date-from", "", "Filter from date (RFC3339)")
	insightsListCmd.Flags().StringVar(&insightsListDateTo, "date-to", "", "Filter to date (RFC3339)")
	insightsListCmd.Flags().StringVar(&insightsListSort, "sort", "-detected_ts", "Sort field (prefix +/- for asc/desc: detected_ts, created_at, updated_at, impact, signal_type)")
}

func runInsightsList(cmd *cobra.Command, args []string) error {
	if insightsListLimit < 1 || insightsListLimit > 100 {
		return fmt.Errorf("Error: --limit must be between 1 and 100")
	}
	if insightsListOffset < 0 {
		return fmt.Errorf("Error: --offset must be >= 0")
	}

	c := NewClient()
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	query := url.Values{}
	query.Set("limit", strconv.Itoa(insightsListLimit))
	query.Set("offset", strconv.Itoa(insightsListOffset))
	query.Set("sort", insightsListSort)
	if insightsListServiceID != "" {
		query.Set("service_id", insightsListServiceID)
	}
	if insightsListStatus != "" {
		query.Set("status", insightsListStatus)
	}
	if insightsListSignalType != "" {
		query.Set("signal_type", insightsListSignalType)
	}
	if insightsListImpact != "" {
		query.Set("impact", insightsListImpact)
	}
	if insightsListDateFrom != "" {
		query.Set("date_from", insightsListDateFrom)
	}
	if insightsListDateTo != "" {
		query.Set("date_to", insightsListDateTo)
	}

	resp, err := c.Get(cmd.Context(), "/insights", query)
	if err != nil {
		return fmt.Errorf("requesting insights: %w", err)
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

	var insights []insightsListItem
	if err := json.Unmarshal(apiResp.Data, &insights); err != nil {
		return fmt.Errorf("parsing insights data: %w", err)
	}

	headers := []string{"ID", "TYPE", "IMPACT", "SIGNAL", "SERVICE", "DETECTED"}
	rows := make([][]string, len(insights))
	for i, ins := range insights {
		displayName := "\u2014"
		impact := "\u2014"
		signalType := "\u2014"
		if ins.InsightType != nil {
			displayName = ins.InsightType.DisplayName
			impact = ins.InsightType.Impact
			signalType = ins.InsightType.SignalType
		}
		rows[i] = []string{ins.ID, displayName, impact, signalType, ins.ServiceName, ins.DetectedTS}
	}

	f.PrintTable(headers, rows)

	if apiResp.Meta.HasMore {
		f.PrintPaginationHint(apiResp.Meta.Total, insightsListOffset, insightsListLimit)
	}

	return nil
}
