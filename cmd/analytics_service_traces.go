package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

type serviceTracesRow struct {
	ServiceName      string `json:"service_name"`
	ServiceNamespace string `json:"service_namespace"`
	Environment      string `json:"environment"`
	SpanCount        int64  `json:"span_count"`
	TraceCount       int64  `json:"trace_count"`
}

var analyticsServiceTracesCmd = &cobra.Command{
	Use:   "service-traces",
	Short: "Span and trace counts per service",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMagnoliaWidget(cmd, "/magnolia/service-traces", renderServiceTraces)
	},
}

func renderServiceTraces(f *output.Formatter, data json.RawMessage) error {
	var rows []serviceTracesRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return fmt.Errorf("parsing service traces: %w", err)
	}
	table := make([][]string, len(rows))
	for i, row := range rows {
		table[i] = []string{row.ServiceName, row.ServiceNamespace, row.Environment, fmt.Sprint(row.SpanCount), fmt.Sprint(row.TraceCount)}
	}
	f.PrintTable([]string{"SERVICE", "NAMESPACE", "ENVIRONMENT", "SPANS", "TRACES"}, table)
	return nil
}

func init() {
	analyticsCmd.AddCommand(analyticsServiceTracesCmd)
}
