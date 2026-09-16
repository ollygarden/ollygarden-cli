package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

// metricRow is shared by the metrics-gauge, metrics-sum, and metrics-histogram
// commands — the three endpoints return the same row shape.
type metricRow struct {
	MetricName        string  `json:"metric_name"`
	ServiceName       string  `json:"service_name"`
	Environment       string  `json:"environment"`
	Datapoints        int64   `json:"datapoints"`
	ValueDiversityPct float64 `json:"value_diversity_pct"`
}

func renderMetricRows(f *output.Formatter, data json.RawMessage) error {
	var rows []metricRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return fmt.Errorf("parsing metrics: %w", err)
	}
	table := make([][]string, len(rows))
	for i, row := range rows {
		table[i] = []string{
			row.MetricName,
			row.ServiceName,
			row.Environment,
			fmt.Sprint(row.Datapoints),
			fmt.Sprintf("%.1f%%", row.ValueDiversityPct),
		}
	}
	f.PrintTable([]string{"METRIC", "SERVICE", "ENVIRONMENT", "DATAPOINTS", "DIVERSITY"}, table)
	return nil
}

var analyticsMetricsGaugeCmd = &cobra.Command{
	Use:   "metrics-gauge",
	Short: "Gauge datapoints and value diversity per metric and service",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMagnoliaWidget(cmd, "/magnolia/metrics-gauge", renderMetricRows)
	},
}

func init() {
	analyticsCmd.AddCommand(analyticsMetricsGaugeCmd)
}
