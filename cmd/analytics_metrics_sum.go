package cmd

import (
	"github.com/spf13/cobra"
)

var analyticsMetricsSumCmd = &cobra.Command{
	Use:   "metrics-sum",
	Short: "Sum datapoints and value diversity per metric and service",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMagnoliaWidget(cmd, "/magnolia/metrics-sum", renderMetricRows)
	},
}

func init() {
	analyticsCmd.AddCommand(analyticsMetricsSumCmd)
}
