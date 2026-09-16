package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
	"go.olly.garden/magnolia/contract"
)

var analyticsTracesCmd = &cobra.Command{
	Use:   "traces",
	Short: "Largest traces by span count",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMagnoliaWidget(cmd, "/magnolia/traces", renderTraces)
	},
}

func renderTraces(f *output.Formatter, data json.RawMessage) error {
	var rows []contract.TraceRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return fmt.Errorf("parsing traces: %w", err)
	}
	table := make([][]string, len(rows))
	for i, row := range rows {
		table[i] = []string{row.TraceID, row.RootSpanName, row.ServiceName, fmt.Sprint(row.SpanCount)}
	}
	f.PrintTable([]string{"TRACE ID", "ROOT SPAN", "SERVICE", "SPANS"}, table)
	return nil
}

func init() {
	analyticsCmd.AddCommand(analyticsTracesCmd)
}
