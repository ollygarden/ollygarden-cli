package cmd

import "github.com/spf13/cobra"

var servicesCmd = &cobra.Command{
	Use:   "services",
	Short: "Manage services",
}

var insightsCmd = &cobra.Command{
	Use:   "insights",
	Short: "Manage insights",
}

var analyticsCmd = &cobra.Command{
	Use:   "analytics",
	Short: "View analytics",
}

var webhooksCmd = &cobra.Command{
	Use:   "webhooks",
	Short: "Manage webhooks",
}

var webhooksDeliveriesCmd = &cobra.Command{
	Use:   "deliveries",
	Short: "View webhook deliveries",
}

func init() {
	rootCmd.AddCommand(servicesCmd)
	rootCmd.AddCommand(insightsCmd)
	rootCmd.AddCommand(analyticsCmd)
	rootCmd.AddCommand(webhooksCmd)
	webhooksCmd.AddCommand(webhooksDeliveriesCmd)
}
