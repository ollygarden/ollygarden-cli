package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	insightsCountsGroupBy     string
	insightsCountsServiceID   string
	insightsCountsStatus      string
	insightsCountsInsightType string
	insightsCountsSignalType  string
	insightsCountsImpact      string
	insightsCountsDateFrom    string
	insightsCountsDateTo      string
)

// insightsCountGroup is one row of the group-by aggregate: a group key and
// how many insights matched the filters within that group.
type insightsCountGroup struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

// countsGroupByHeader names the human-mode column for each --group-by value,
// matching the equivalent column name on "insights list".
var countsGroupByHeader = map[string]string{
	"insight_type": "TYPE",
	"service":      "SERVICE",
	"impact":       "IMPACT",
}

var insightsCountsCmd = &cobra.Command{
	Use:   "counts",
	Short: "Count insights grouped by type, service, or impact",
	Args:  cobra.NoArgs,
	RunE:  runInsightsCounts,
}

func init() {
	insightsCmd.AddCommand(insightsCountsCmd)
	insightsCountsCmd.Flags().StringVar(&insightsCountsGroupBy, "group-by", "", "Group counts by this field (required: insight_type, service, impact)")
	insightsCountsCmd.Flags().StringVar(&insightsCountsServiceID, "service-id", "", "Filter by service ID")
	insightsCountsCmd.Flags().StringVar(&insightsCountsStatus, "status", "", "Filter by status (comma-separated: active, archived, muted)")
	insightsCountsCmd.Flags().StringVar(&insightsCountsInsightType, "insight-type", "", "Filter by exact insight type names (comma-separated, 1-100 values, each 1-128 characters)")
	insightsCountsCmd.Flags().StringVar(&insightsCountsSignalType, "signal-type", "", "Filter by signal type (trace, metric, log, resource). Each names where the finding lives in the OTLP model, not which signal carried the evidence: a resource-attribute finding is always \"resource\", even if what it describes is a metric.")
	insightsCountsCmd.Flags().StringVar(&insightsCountsImpact, "impact", "", "Filter by impact (comma-separated: Critical, Important, Normal, Low)")
	insightsCountsCmd.Flags().StringVar(&insightsCountsDateFrom, "date-from", "", "Filter created_at >= this date (RFC3339)")
	insightsCountsCmd.Flags().StringVar(&insightsCountsDateTo, "date-to", "", "Filter created_at <= this date (RFC3339)")
}

func insightsCountsFilters() insightsFilterFlags {
	return insightsFilterFlags{
		serviceID:   insightsCountsServiceID,
		status:      insightsCountsStatus,
		insightType: insightsCountsInsightType,
		signalType:  insightsCountsSignalType,
		impact:      insightsCountsImpact,
		dateFrom:    insightsCountsDateFrom,
		dateTo:      insightsCountsDateTo,
	}
}

func runInsightsCounts(cmd *cobra.Command, args []string) error {
	header, ok := countsGroupByHeader[insightsCountsGroupBy]
	if !ok {
		return fmt.Errorf("--group-by is required and must be one of: insight_type, service, impact")
	}
	if err := insightsCountsFilters().validate(); err != nil {
		return err
	}

	c := NewClient()
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	query := url.Values{}
	query.Set("group_by", insightsCountsGroupBy)
	insightsCountsFilters().applyQuery(query)

	resp, err := c.Get(cmd.Context(), "/insights/counts", query)
	if err != nil {
		return fmt.Errorf("requesting insight counts: %w", err)
	}

	apiResp, err := client.ParseResponse(resp)
	if err != nil {
		printAPIError(f, err)
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

	var groups []insightsCountGroup
	if err := json.Unmarshal(apiResp.Data, &groups); err != nil {
		return fmt.Errorf("parsing insight counts data: %w", err)
	}

	rows := make([][]string, len(groups))
	for i, g := range groups {
		rows[i] = []string{g.Key, fmt.Sprintf("%d", g.Count)}
	}
	f.PrintTable([]string{header, "COUNT"}, rows)

	if apiResp.Meta.Capped {
		fmt.Fprintln(cmd.ErrOrStderr(), "# Warning: group cap reached; counts are truncated, not a total.")
	}

	return nil
}
