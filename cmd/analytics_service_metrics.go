package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
	"go.olly.garden/magnolia/contract"
)

var analyticsServiceMetricsCmd = &cobra.Command{
	Use:   "service-metrics",
	Short: "Metric datapoints per service, broken down by instrument type",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMagnoliaWidget(cmd, "/magnolia/service-metrics", renderServiceMetrics)
	},
}

func renderServiceMetrics(f *output.Formatter, data json.RawMessage) error {
	var rows []contract.ServicesRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return fmt.Errorf("parsing service metrics: %w", err)
	}
	table := make([][]string, len(rows))
	for i, row := range rows {
		table[i] = []string{
			row.ServiceName,
			row.Environment,
			formatCount(row.Datapoints),
			formatCount(row.GaugeDatapoints),
			formatCount(row.SumDatapoints),
			formatCount(row.HistogramDatapoints),
		}
	}
	f.PrintTable([]string{"SERVICE", "ENVIRONMENT", "DATAPOINTS", "GAUGE", "SUM", "HISTOGRAM"}, table)
	return nil
}

func init() {
	analyticsCmd.AddCommand(analyticsServiceMetricsCmd)
}
