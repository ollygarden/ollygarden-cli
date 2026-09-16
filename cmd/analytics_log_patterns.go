package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
	"go.olly.garden/magnolia/contract"
)

var analyticsLogPatternsCmd = &cobra.Command{
	Use:   "log-patterns",
	Short: "Counts of normalized log body previews per service and severity",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMagnoliaWidget(cmd, "/magnolia/log-patterns", renderLogPatterns)
	},
}

func renderLogPatterns(f *output.Formatter, data json.RawMessage) error {
	var rows []contract.LogPatternRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return fmt.Errorf("parsing log patterns: %w", err)
	}
	table := make([][]string, len(rows))
	for i, row := range rows {
		table[i] = []string{row.ServiceName, row.SeverityText, row.Environment, fmt.Sprint(row.Cnt), row.BodyPreview}
	}
	f.PrintTable([]string{"SERVICE", "SEVERITY", "ENVIRONMENT", "COUNT", "PREVIEW"}, table)
	return nil
}

func init() {
	analyticsCmd.AddCommand(analyticsLogPatternsCmd)
}
