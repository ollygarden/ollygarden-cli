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
	servicesInsightsStatus string
	servicesInsightsLimit  int
	servicesInsightsOffset int
)

type insightItem struct {
	ID          string              `json:"id"`
	Status      string              `json:"status"`
	InsightType *insightTypeCompact `json:"insight_type"`
	DetectedTS  string              `json:"detected_ts"`
}

type insightTypeCompact struct {
	DisplayName string `json:"display_name"`
	Impact      string `json:"impact"`
	SignalType  string `json:"signal_type"`
}

var servicesInsightsCmd = &cobra.Command{
	Use:   "insights <service-id>",
	Short: "List insights for a service",
	Args:  cobra.ExactArgs(1),
	RunE:  runServicesInsights,
}

func init() {
	servicesCmd.AddCommand(servicesInsightsCmd)
	servicesInsightsCmd.Flags().StringVar(&servicesInsightsStatus, "status", "active", "Filter by status (comma-separated: active, archived, muted)")
	servicesInsightsCmd.Flags().IntVar(&servicesInsightsLimit, "limit", 50, "Maximum number of results (1-100)")
	servicesInsightsCmd.Flags().IntVar(&servicesInsightsOffset, "offset", 0, "Number of results to skip (≥0)")
}

func runServicesInsights(cmd *cobra.Command, args []string) error {
	if servicesInsightsLimit < 1 || servicesInsightsLimit > 100 {
		return fmt.Errorf("--limit must be between 1 and 100")
	}
	if servicesInsightsOffset < 0 {
		return fmt.Errorf("--offset must be >= 0")
	}

	serviceID := args[0]

	c := NewClient()
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	query := url.Values{}
	query.Set("status", servicesInsightsStatus)
	query.Set("limit", strconv.Itoa(servicesInsightsLimit))
	query.Set("offset", strconv.Itoa(servicesInsightsOffset))

	resp, err := c.Get(cmd.Context(), "/services/"+serviceID+"/insights", query)
	if err != nil {
		return fmt.Errorf("requesting service insights: %w", err)
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

	var insights []insightItem
	if err := json.Unmarshal(apiResp.Data, &insights); err != nil {
		return fmt.Errorf("parsing insights data: %w", err)
	}

	headers := []string{"ID", "TYPE", "IMPACT", "SIGNAL", "DETECTED"}
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
		rows[i] = []string{ins.ID, displayName, impact, signalType, ins.DetectedTS}
	}

	f.PrintTable(headers, rows)

	if apiResp.Meta.HasMore {
		f.PrintPaginationHint(apiResp.Meta.Total, servicesInsightsOffset, servicesInsightsLimit)
	}

	return nil
}
