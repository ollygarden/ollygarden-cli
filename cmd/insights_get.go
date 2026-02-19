package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

type insightDetail struct {
	ID                 string              `json:"id"`
	Status             string              `json:"status"`
	ServiceID          string              `json:"service_id"`
	ServiceName        string              `json:"service_name"`
	ServiceEnvironment string              `json:"service_environment"`
	InsightType        *insightTypeCompact `json:"insight_type"`
	DetectedTS         string              `json:"detected_ts"`
	CreatedAt          string              `json:"created_at"`
	UpdatedAt          string              `json:"updated_at"`
}

var insightsGetCmd = &cobra.Command{
	Use:   "get <insight-id>",
	Short: "Show insight details",
	Args:  cobra.ExactArgs(1),
	RunE:  runInsightsGet,
}

func init() {
	insightsCmd.AddCommand(insightsGetCmd)
}

func runInsightsGet(cmd *cobra.Command, args []string) error {
	insightID := args[0]

	c := NewClient()
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	resp, err := c.Get(cmd.Context(), "/insights/"+insightID, nil)
	if err != nil {
		return fmt.Errorf("requesting insight: %w", err)
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

	var ins insightDetail
	if err := json.Unmarshal(apiResp.Data, &ins); err != nil {
		return fmt.Errorf("parsing insight data: %w", err)
	}

	emDash := "\u2014"
	valOrDash := func(s string) string {
		if s == "" {
			return emDash
		}
		return s
	}

	displayName := emDash
	impact := emDash
	signalType := emDash
	if ins.InsightType != nil {
		displayName = ins.InsightType.DisplayName
		impact = ins.InsightType.Impact
		signalType = ins.InsightType.SignalType
	}

	pairs := []output.KVPair{
		{Key: "ID", Value: ins.ID},
		{Key: "Status", Value: ins.Status},
		{Key: "Type", Value: displayName},
		{Key: "Impact", Value: impact},
		{Key: "Signal", Value: signalType},
		{Key: "Service", Value: ins.ServiceName + " (" + ins.ServiceID + ")"},
		{Key: "Environment", Value: valOrDash(ins.ServiceEnvironment)},
		{Key: "Detected", Value: ins.DetectedTS},
		{Key: "Created", Value: ins.CreatedAt},
		{Key: "Updated", Value: ins.UpdatedAt},
	}

	f.PrintKeyValue(pairs)
	return nil
}
