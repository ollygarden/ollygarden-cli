package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
	"go.olly.garden/magnolia/contract"
)

var analyticsSpanNamesCmd = &cobra.Command{
	Use:   "span-names",
	Short: "Span counts by service, kind, and span name",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMagnoliaWidget(cmd, "/magnolia/span-names", renderSpanNames)
	},
}

func renderSpanNames(f *output.Formatter, data json.RawMessage) error {
	var rows []contract.SpanNameRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return fmt.Errorf("parsing span names: %w", err)
	}
	table := make([][]string, len(rows))
	for i, row := range rows {
		table[i] = []string{row.ServiceName, row.SpanKind, row.SpanName, row.ServiceNamespace, row.Environment, fmt.Sprint(row.Cnt)}
	}
	f.PrintTable([]string{"SERVICE", "KIND", "NAME", "NAMESPACE", "ENVIRONMENT", "COUNT"}, table)
	return nil
}

func init() {
	analyticsCmd.AddCommand(analyticsSpanNamesCmd)
}
