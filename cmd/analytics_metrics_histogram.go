package cmd

import (
	"github.com/spf13/cobra"
)

var analyticsMetricsHistogramCmd = &cobra.Command{
	Use:   "metrics-histogram",
	Short: "Histogram datapoints and value diversity per metric and service",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMagnoliaWidget(cmd, "/magnolia/metrics-histogram", renderMetricRows)
	},
}

func init() {
	analyticsCmd.AddCommand(analyticsMetricsHistogramCmd)
}
