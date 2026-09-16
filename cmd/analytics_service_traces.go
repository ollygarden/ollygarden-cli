package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
	"go.olly.garden/magnolia/contract"
)

var analyticsServiceTracesCmd = &cobra.Command{
	Use:   "service-traces",
	Short: "Span and trace counts per service",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMagnoliaWidget(cmd, "/magnolia/service-traces", renderServiceTraces)
	},
}

func renderServiceTraces(f *output.Formatter, data json.RawMessage) error {
	var rows []contract.ServicesRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return fmt.Errorf("parsing service traces: %w", err)
	}
	table := make([][]string, len(rows))
	for i, row := range rows {
		table[i] = []string{row.ServiceName, row.ServiceNamespace, row.Environment, formatCount(row.SpanCount), formatCount(row.TraceCount)}
	}
	f.PrintTable([]string{"SERVICE", "NAMESPACE", "ENVIRONMENT", "SPANS", "TRACES"}, table)
	return nil
}

func init() {
	analyticsCmd.AddCommand(analyticsServiceTracesCmd)
}
