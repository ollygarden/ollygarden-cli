package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

type serviceMetricsRow struct {
	ServiceName         string `json:"service_name"`
	Environment         string `json:"environment"`
	Datapoints          int64  `json:"datapoints"`
	GaugeDatapoints     int64  `json:"gauge_datapoints"`
	SumDatapoints       int64  `json:"sum_datapoints"`
	HistogramDatapoints int64  `json:"histogram_datapoints"`
}

var analyticsServiceMetricsCmd = &cobra.Command{
	Use:   "service-metrics",
	Short: "Metric datapoints per service, broken down by instrument type",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMagnoliaWidget(cmd, "/magnolia/service-metrics", renderServiceMetrics)
	},
}

func renderServiceMetrics(f *output.Formatter, data json.RawMessage) error {
	var rows []serviceMetricsRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return fmt.Errorf("parsing service metrics: %w", err)
	}
	table := make([][]string, len(rows))
	for i, row := range rows {
		table[i] = []string{
			row.ServiceName,
			row.Environment,
			fmt.Sprint(row.Datapoints),
			fmt.Sprint(row.GaugeDatapoints),
			fmt.Sprint(row.SumDatapoints),
			fmt.Sprint(row.HistogramDatapoints),
		}
	}
	f.PrintTable([]string{"SERVICE", "ENVIRONMENT", "DATAPOINTS", "GAUGE", "SUM", "HISTOGRAM"}, table)
	return nil
}

func init() {
	analyticsCmd.AddCommand(analyticsServiceMetricsCmd)
}
