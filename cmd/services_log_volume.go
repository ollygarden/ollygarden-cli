package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var servicesLogVolumePeriod string

var servicesLogVolumeCmd = &cobra.Command{
	Use:   "log-volume <service-id>",
	Short: "Show service log volume by severity",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLogVolume(cmd, fmt.Sprintf("/services/%s/analytics/log-volume", args[0]), servicesLogVolumePeriod)
	},
}

func init() {
	servicesCmd.AddCommand(servicesLogVolumeCmd)
	servicesLogVolumeCmd.Flags().StringVar(&servicesLogVolumePeriod, "period", "24h", "Time period: 1h, 6h, 12h, 24h, 7d, or 30d")
}
