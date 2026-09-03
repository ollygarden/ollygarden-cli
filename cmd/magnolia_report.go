package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

var magnoliaReportOrgID string

type magnoliaReport struct {
	OrgID       string `json:"orgId"`
	GeneratedAt string `json:"generatedAt"`
	Window      struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"window"`
	Data struct {
		Summary struct {
			Totals struct {
				Spans      int64 `json:"spans"`
				Logs       int64 `json:"logs"`
				Datapoints int64 `json:"datapoints"`
			} `json:"totals"`
		} `json:"summary"`
	} `json:"data"`
}

var magnoliaReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Get the latest pre-computed Magnolia report",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		f := newFormatter(cmd)
		body, err := magnoliaGet(cmd, f, "/magnolia/report", magnoliaReportOrgID)
		if err != nil {
			return err
		}
		if f.IsJSON() {
			f.PrintJSON(body)
			return nil
		}
		if f.IsQuiet() {
			return nil
		}
		var report magnoliaReport
		if err := json.Unmarshal(body, &report); err != nil {
			return fmt.Errorf("parsing Magnolia report: %w", err)
		}
		f.PrintKeyValue([]output.KVPair{
			{Key: "Organization", Value: report.OrgID},
			{Key: "Generated", Value: report.GeneratedAt},
			{Key: "Window", Value: report.Window.From + " to " + report.Window.To},
			{Key: "Spans", Value: fmt.Sprint(report.Data.Summary.Totals.Spans)},
			{Key: "Logs", Value: fmt.Sprint(report.Data.Summary.Totals.Logs)},
			{Key: "Datapoints", Value: fmt.Sprint(report.Data.Summary.Totals.Datapoints)},
		})
		return nil
	},
}

func init() {
	magnoliaCmd.AddCommand(magnoliaReportCmd)
	magnoliaReportCmd.Flags().StringVar(&magnoliaReportOrgID, "org-id", "", "Organization identifier")
	_ = magnoliaReportCmd.MarkFlagRequired("org-id")
}
