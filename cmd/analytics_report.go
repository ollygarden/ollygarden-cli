package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

var legacyMagnoliaReportOrgID string

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

var analyticsReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Get the latest pre-computed Magnolia report",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAnalyticsReport(cmd, "")
	},
}

var legacyMagnoliaReportCmd = &cobra.Command{
	Use:    "report",
	Short:  "Get the latest pre-computed Magnolia report",
	Args:   cobra.NoArgs,
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if legacyMagnoliaReportOrgID == "" {
			return fmt.Errorf("--org-id is required")
		}
		return runAnalyticsReport(cmd, legacyMagnoliaReportOrgID)
	},
}

func runAnalyticsReport(cmd *cobra.Command, orgID string) error {
	f := newFormatter(cmd)
	body, err := magnoliaGet(cmd, f, "/magnolia/report", orgID)
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
}

func init() {
	analyticsCmd.AddCommand(analyticsReportCmd)
	magnoliaCmd.AddCommand(legacyMagnoliaReportCmd)
	legacyMagnoliaReportCmd.Flags().StringVar(&legacyMagnoliaReportOrgID, "org-id", "", "Organization identifier")
	_ = legacyMagnoliaReportCmd.MarkFlagRequired("org-id")
}
