package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

var analyticsLogVolumePeriod string

var validLogVolumePeriods = map[string]struct{}{
	"1h": {}, "6h": {}, "12h": {}, "24h": {}, "7d": {}, "30d": {},
}

type logVolumeData struct {
	Period     string              `json:"period"`
	TotalCount int64               `json:"total_count"`
	Severities []logVolumeSeverity `json:"severities"`
}

type logVolumeSeverity struct {
	SeverityText string  `json:"severity_text"`
	RecordCount  int64   `json:"record_count"`
	Percent      float64 `json:"percent"`
}

var analyticsLogVolumeCmd = &cobra.Command{
	Use:   "log-volume",
	Short: "Show organization log volume by severity",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLogVolume(cmd, "/analytics/log-volume", analyticsLogVolumePeriod)
	},
}

func init() {
	analyticsCmd.AddCommand(analyticsLogVolumeCmd)
	analyticsLogVolumeCmd.Flags().StringVar(&analyticsLogVolumePeriod, "period", "24h", "Time period: 1h, 6h, 12h, 24h, 7d, or 30d")
}

func runLogVolume(cmd *cobra.Command, path, period string) error {
	if _, ok := validLogVolumePeriods[period]; !ok {
		return fmt.Errorf("--period must be one of: 1h, 6h, 12h, 24h, 7d, 30d")
	}

	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)
	query := url.Values{"period": []string{period}}
	resp, err := NewClient().Get(cmd.Context(), path, query)
	if err != nil {
		return fmt.Errorf("requesting log volume: %w", err)
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

	var data logVolumeData
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		return fmt.Errorf("parsing log volume data: %w", err)
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Period: %s\nTotal records: %d\n", data.Period, data.TotalCount)
	rows := make([][]string, len(data.Severities))
	for i, severity := range data.Severities {
		rows[i] = []string{severity.SeverityText, fmt.Sprintf("%d", severity.RecordCount), fmt.Sprintf("%.2f%%", severity.Percent)}
	}
	f.PrintTable([]string{"SEVERITY", "RECORDS", "SHARE"}, rows)
	return nil
}
