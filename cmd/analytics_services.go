package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

var analyticsServicesLimit int

type analyticsServicesData struct {
	PeriodStart string                 `json:"period_start"`
	PeriodEnd   string                 `json:"period_end"`
	Services    []analyticsServiceItem `json:"services"`
}

type analyticsServiceItem struct {
	Name          string                   `json:"name"`
	Namespace     string                   `json:"namespace"`
	Environment   string                   `json:"environment"`
	TotalBytes    int64                    `json:"total_bytes"`
	TotalPercent  float64                  `json:"total_percent"`
	LatestVersion *analyticsServiceVersion `json:"latest_version"`
}

type analyticsServiceVersion struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

var analyticsServicesCmd = &cobra.Command{
	Use:   "services",
	Short: "List service analytics",
	Args:  cobra.NoArgs,
	RunE:  runAnalyticsServices,
}

func init() {
	analyticsCmd.AddCommand(analyticsServicesCmd)
	analyticsServicesCmd.Flags().IntVar(&analyticsServicesLimit, "limit", 50, "Maximum number of results (1-100)")
}

func runAnalyticsServices(cmd *cobra.Command, args []string) error {
	if analyticsServicesLimit < 1 || analyticsServicesLimit > 100 {
		return fmt.Errorf("Error: --limit must be between 1 and 100")
	}

	c := NewClient()
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	query := url.Values{}
	query.Set("limit", strconv.Itoa(analyticsServicesLimit))

	resp, err := c.Get(cmd.Context(), "/analytics/services", query)
	if err != nil {
		return fmt.Errorf("requesting analytics services: %w", err)
	}

	apiResp, err := client.ParseResponse(resp)
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok {
			var raw json.RawMessage
			if apiErr.ErrorResponse != nil {
				raw, _ = json.Marshal(apiErr.ErrorResponse)
			}
			f.PrintError(apiErr.Error(), raw)
		}
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

	var data analyticsServicesData
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		return fmt.Errorf("parsing analytics services data: %w", err)
	}

	// Period header on stderr
	fmt.Fprintf(cmd.ErrOrStderr(), "Period: %s to %s\n", data.PeriodStart, data.PeriodEnd)

	headers := []string{"NAME", "ENVIRONMENT", "TOTAL", "TOTAL %", "VERSION"}
	rows := make([][]string, len(data.Services))
	for i, svc := range data.Services {
		version := "\u2014"
		if svc.LatestVersion != nil {
			version = svc.LatestVersion.Version
		}
		rows[i] = []string{
			svc.Name,
			svc.Environment,
			formatBytes(svc.TotalBytes),
			fmt.Sprintf("%.1f%%", svc.TotalPercent),
			version,
		}
	}

	f.PrintTable(headers, rows)
	return nil
}

// formatBytes converts bytes to a human-readable string.
func formatBytes(b int64) string {
	const (
		kb = 1000
		mb = 1000 * kb
		gb = 1000 * mb
		tb = 1000 * gb
	)
	switch {
	case b >= tb:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(tb))
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
