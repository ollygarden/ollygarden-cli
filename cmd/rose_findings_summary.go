package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

type roseFindingsCount struct {
	Severity string `json:"severity"`
	Category string `json:"category"`
	Count    int    `json:"count"`
}

type roseFindingsSummaryData struct {
	Total      int                 `json:"total"`
	BySeverity []roseFindingsCount `json:"by_severity"`
	ByCategory []roseFindingsCount `json:"by_category"`
}

var roseFindingsSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show active finding counts by severity and category",
	Args:  cobra.NoArgs,
	RunE:  runRoseFindingsSummary,
}

func init() {
	roseFindingsCmd.AddCommand(roseFindingsSummaryCmd)
}

func runRoseFindingsSummary(cmd *cobra.Command, args []string) error {
	f := newFormatter(cmd)

	apiResp, err := roseGet(cmd.Context(), f, "/rose/findings/summary", nil)
	if err != nil {
		return err
	}

	if f.IsJSON() {
		raw, _ := json.Marshal(apiResp)
		f.PrintJSON(raw)
		return nil
	}
	if f.IsQuiet() {
		return nil
	}

	var summary roseFindingsSummaryData
	if err := json.Unmarshal(apiResp.Data, &summary); err != nil {
		return fmt.Errorf("parsing findings summary: %w", err)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Total active findings:  %d\n\n", summary.Total)

	rows := make([][]string, len(summary.BySeverity))
	for i, c := range summary.BySeverity {
		rows[i] = []string{c.Severity, strconv.Itoa(c.Count)}
	}
	f.PrintTable([]string{"SEVERITY", "COUNT"}, rows)

	fmt.Fprintln(out)

	rows = make([][]string, len(summary.ByCategory))
	for i, c := range summary.ByCategory {
		rows[i] = []string{c.Category, strconv.Itoa(c.Count)}
	}
	f.PrintTable([]string{"CATEGORY", "COUNT"}, rows)
	return nil
}
