package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	webhooksCreateName         string
	webhooksCreateURL          string
	webhooksCreateEventTypes   []string
	webhooksCreateEnvironments []string
	webhooksCreateMinSeverity  string
	webhooksCreateEnabled      bool
)

var webhooksCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a webhook",
	Args:  cobra.NoArgs,
	RunE:  runWebhooksCreate,
}

func init() {
	webhooksCmd.AddCommand(webhooksCreateCmd)

	webhooksCreateCmd.Flags().StringVar(&webhooksCreateName, "name", "", "Webhook name (required)")
	webhooksCreateCmd.Flags().StringVar(&webhooksCreateURL, "url", "", "HTTPS URL for delivery (required)")
	webhooksCreateCmd.Flags().StringArrayVar(&webhooksCreateEventTypes, "event-type", nil, "Insight type IDs (repeatable)")
	webhooksCreateCmd.Flags().StringArrayVar(&webhooksCreateEnvironments, "environment", nil, "Environments (repeatable)")
	webhooksCreateCmd.Flags().StringVar(&webhooksCreateMinSeverity, "min-severity", "Low", "Minimum severity: Low, Normal, Important, Critical")
	webhooksCreateCmd.Flags().BoolVar(&webhooksCreateEnabled, "enabled", false, "Enable the webhook")

	// MarkFlagRequired only errors if the flag doesn't exist — panic here surfaces
	// programmer error at startup rather than silently skipping the requirement.
	if err := webhooksCreateCmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}
	if err := webhooksCreateCmd.MarkFlagRequired("url"); err != nil {
		panic(err)
	}
}

var validSeverities = map[string]bool{
	"Low": true, "Normal": true, "Important": true, "Critical": true,
}

type webhookCreateRequest struct {
	Name         string   `json:"name"`
	URL          string   `json:"url"`
	IsEnabled    bool     `json:"is_enabled"`
	MinSeverity  string   `json:"min_severity"`
	EventTypes   []string `json:"event_types"`
	Environments []string `json:"environments"`
}

func runWebhooksCreate(cmd *cobra.Command, args []string) error {
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	if len(webhooksCreateName) > 255 {
		return fmt.Errorf("Error: --name must be at most 255 characters")
	}
	if !validSeverities[webhooksCreateMinSeverity] {
		return fmt.Errorf("Error: --min-severity must be one of: Low, Normal, Important, Critical")
	}

	eventTypes := webhooksCreateEventTypes
	if eventTypes == nil {
		eventTypes = []string{}
	}
	environments := webhooksCreateEnvironments
	if environments == nil {
		environments = []string{}
	}

	body := webhookCreateRequest{
		Name:         webhooksCreateName,
		URL:          webhooksCreateURL,
		IsEnabled:    webhooksCreateEnabled,
		MinSeverity:  webhooksCreateMinSeverity,
		EventTypes:   eventTypes,
		Environments: environments,
	}

	c := NewClient()
	resp, err := c.Post(cmd.Context(), "/webhooks", body)
	if err != nil {
		return fmt.Errorf("creating webhook: %w", err)
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

	var wh webhookDetail
	if err := json.Unmarshal(apiResp.Data, &wh); err != nil {
		return fmt.Errorf("parsing webhook data: %w", err)
	}

	joinOrAll := func(s []string) string {
		if len(s) == 0 {
			return "all"
		}
		return strings.Join(s, ", ")
	}

	pairs := []output.KVPair{
		{Key: "ID", Value: wh.ID},
		{Key: "Name", Value: wh.Name},
		{Key: "URL", Value: wh.URL},
		{Key: "Enabled", Value: strconv.FormatBool(wh.IsEnabled)},
		{Key: "Severity", Value: wh.MinSeverity},
		{Key: "Event Types", Value: joinOrAll(wh.EventTypes)},
		{Key: "Environments", Value: joinOrAll(wh.Environments)},
		{Key: "Created", Value: wh.CreatedAt},
		{Key: "Updated", Value: wh.UpdatedAt},
	}

	f.PrintKeyValue(pairs)
	return nil
}
