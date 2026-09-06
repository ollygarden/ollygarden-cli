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

var apiKeysCmd = &cobra.Command{
	Use:   "api-keys",
	Short: "Manage API keys",
}

var webhooksCmd = &cobra.Command{
	Use:   "webhooks",
	Short: "Manage webhooks",
}

var webhooksDeliveriesCmd = &cobra.Command{
	Use:   "deliveries",
	Short: "View webhook deliveries",
}

var roseCmd = &cobra.Command{
	Use:   "rose",
	Short: "View Rose codebase findings, repositories, and executions",
}

var roseFindingsCmd = &cobra.Command{
	Use:   "findings",
	Short: "View Rose codebase findings",
}

var roseRepositoriesCmd = &cobra.Command{
	Use:   "repositories",
	Short: "View repositories connected to Rose",
}

var roseExecutionsCmd = &cobra.Command{
	Use:   "executions",
	Short: "View Rose executions",
}

func init() {
	rootCmd.AddCommand(servicesCmd)
	rootCmd.AddCommand(insightsCmd)
	rootCmd.AddCommand(analyticsCmd)
	rootCmd.AddCommand(apiKeysCmd)
	rootCmd.AddCommand(webhooksCmd)
	webhooksCmd.AddCommand(webhooksDeliveriesCmd)
	rootCmd.AddCommand(roseCmd)
	roseCmd.AddCommand(roseFindingsCmd)
	roseCmd.AddCommand(roseRepositoriesCmd)
	roseCmd.AddCommand(roseExecutionsCmd)
}
