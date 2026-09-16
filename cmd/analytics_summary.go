package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

type magnoliaSummary struct {
	Totals struct {
		Spans            int64 `json:"spans"`
		LogRecords       int64 `json:"log_records"`
		MetricDatapoints int64 `json:"metric_datapoints"`
		MetricDPByType   struct {
			Gauge     int64 `json:"gauge"`
			Sum       int64 `json:"sum"`
			Histogram int64 `json:"histogram"`
		} `json:"metric_datapoints_by_type"`
	} `json:"totals"`
	ServiceCountsBySignal []struct {
		Signal   string `json:"signal"`
		Services int64  `json:"services"`
	} `json:"service_counts_by_signal"`
}

var analyticsSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Per-signal totals and service counts from the latest Magnolia run",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMagnoliaWidget(cmd, "/magnolia/summary", renderMagnoliaSummary)
	},
}

func renderMagnoliaSummary(f *output.Formatter, data json.RawMessage) error {
	var summary magnoliaSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return fmt.Errorf("parsing summary: %w", err)
	}
	f.PrintKeyValue([]output.KVPair{
		{Key: "Spans", Value: fmt.Sprint(summary.Totals.Spans)},
		{Key: "Log records", Value: fmt.Sprint(summary.Totals.LogRecords)},
		{Key: "Metric datapoints", Value: fmt.Sprint(summary.Totals.MetricDatapoints)},
		{Key: "Gauge datapoints", Value: fmt.Sprint(summary.Totals.MetricDPByType.Gauge)},
		{Key: "Sum datapoints", Value: fmt.Sprint(summary.Totals.MetricDPByType.Sum)},
		{Key: "Histogram datapoints", Value: fmt.Sprint(summary.Totals.MetricDPByType.Histogram)},
	})
	rows := make([][]string, len(summary.ServiceCountsBySignal))
	for i, entry := range summary.ServiceCountsBySignal {
		rows[i] = []string{entry.Signal, fmt.Sprint(entry.Services)}
	}
	f.PrintTable([]string{"SIGNAL", "SERVICES"}, rows)
	return nil
}

func init() {
	analyticsCmd.AddCommand(analyticsSummaryCmd)
}
