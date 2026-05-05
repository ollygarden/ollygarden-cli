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
	webhooksUpdateName         string
	webhooksUpdateURL          string
	webhooksUpdateEventTypes   []string
	webhooksUpdateEnvironments []string
	webhooksUpdateMinSeverity  string
	webhooksUpdateEnabled      bool
)

var webhooksUpdateCmd = &cobra.Command{
	Use:   "update <webhook-id>",
	Short: "Update a webhook",
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhooksUpdate,
}

func init() {
	webhooksCmd.AddCommand(webhooksUpdateCmd)

	webhooksUpdateCmd.Flags().StringVar(&webhooksUpdateName, "name", "", "Webhook name")
	webhooksUpdateCmd.Flags().StringVar(&webhooksUpdateURL, "url", "", "HTTPS URL for delivery")
	webhooksUpdateCmd.Flags().StringArrayVar(&webhooksUpdateEventTypes, "event-type", nil, "Insight type IDs (repeatable)")
	webhooksUpdateCmd.Flags().StringArrayVar(&webhooksUpdateEnvironments, "environment", nil, "Environments (repeatable)")
	webhooksUpdateCmd.Flags().StringVar(&webhooksUpdateMinSeverity, "min-severity", "", "Minimum severity: Low, Normal, Important, Critical")
	webhooksUpdateCmd.Flags().BoolVar(&webhooksUpdateEnabled, "enabled", false, "Enable/disable the webhook")
}

func runWebhooksUpdate(cmd *cobra.Command, args []string) error {
	webhookID := args[0]
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	// Check at least one flag was provided
	flags := []string{"name", "url", "event-type", "environment", "min-severity", "enabled"}
	anyChanged := false
	for _, flag := range flags {
		if cmd.Flags().Changed(flag) {
			anyChanged = true
			break
		}
	}
	if !anyChanged {
		return fmt.Errorf("at least one flag is required")
	}

	// Build partial request body — only include changed fields
	body := make(map[string]any)

	if cmd.Flags().Changed("name") {
		if len(webhooksUpdateName) > 255 {
			return fmt.Errorf("--name must be at most 255 characters")
		}
		body["name"] = webhooksUpdateName
	}

	if cmd.Flags().Changed("url") {
		body["url"] = webhooksUpdateURL
	}

	if cmd.Flags().Changed("event-type") {
		eventTypes := webhooksUpdateEventTypes
		if eventTypes == nil {
			eventTypes = []string{}
		}
		body["event_types"] = eventTypes
	}

	if cmd.Flags().Changed("environment") {
		environments := webhooksUpdateEnvironments
		if environments == nil {
			environments = []string{}
		}
		body["environments"] = environments
	}

	if cmd.Flags().Changed("min-severity") {
		if !validSeverities[webhooksUpdateMinSeverity] {
			return fmt.Errorf("--min-severity must be one of: Low, Normal, Important, Critical")
		}
		body["min_severity"] = webhooksUpdateMinSeverity
	}

	if cmd.Flags().Changed("enabled") {
		body["is_enabled"] = webhooksUpdateEnabled
	}

	c := NewClient()
	resp, err := c.Put(cmd.Context(), "/webhooks/"+webhookID, body)
	if err != nil {
		return fmt.Errorf("updating webhook: %w", err)
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
