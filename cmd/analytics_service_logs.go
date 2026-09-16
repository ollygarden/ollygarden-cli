package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
	"go.olly.garden/magnolia/contract"
)

var analyticsServiceLogsCmd = &cobra.Command{
	Use:   "service-logs",
	Short: "Log record counts per service",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMagnoliaWidget(cmd, "/magnolia/service-logs", renderServiceLogs)
	},
}

func renderServiceLogs(f *output.Formatter, data json.RawMessage) error {
	var rows []contract.ServicesRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return fmt.Errorf("parsing service logs: %w", err)
	}
	table := make([][]string, len(rows))
	for i, row := range rows {
		table[i] = []string{row.ServiceName, row.ServiceNamespace, row.Environment, formatCount(row.LogCount)}
	}
	f.PrintTable([]string{"SERVICE", "NAMESPACE", "ENVIRONMENT", "LOGS"}, table)
	return nil
}

func init() {
	analyticsCmd.AddCommand(analyticsServiceLogsCmd)
}
