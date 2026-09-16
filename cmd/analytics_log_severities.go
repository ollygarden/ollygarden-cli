package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

type logSeverityRow struct {
	SeverityText string `json:"SeverityText"`
	Environment  string `json:"environment"`
	Count        int64  `json:"cnt"`
}

var analyticsLogSeveritiesCmd = &cobra.Command{
	Use:   "log-severities",
	Short: "Log record counts by severity and environment",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMagnoliaWidget(cmd, "/magnolia/log-severities", renderLogSeverities)
	},
}

func renderLogSeverities(f *output.Formatter, data json.RawMessage) error {
	var rows []logSeverityRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return fmt.Errorf("parsing log severities: %w", err)
	}
	table := make([][]string, len(rows))
	for i, row := range rows {
		table[i] = []string{row.SeverityText, row.Environment, fmt.Sprint(row.Count)}
	}
	f.PrintTable([]string{"SEVERITY", "ENVIRONMENT", "COUNT"}, table)
	return nil
}

func init() {
	analyticsCmd.AddCommand(analyticsLogSeveritiesCmd)
}
