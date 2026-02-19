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

type webhookDetail struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	URL          string   `json:"url"`
	IsEnabled    bool     `json:"is_enabled"`
	MinSeverity  string   `json:"min_severity"`
	EventTypes   []string `json:"event_types"`
	Environments []string `json:"environments"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

var webhooksGetCmd = &cobra.Command{
	Use:   "get <webhook-id>",
	Short: "Show webhook details",
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhooksGet,
}

func init() {
	webhooksCmd.AddCommand(webhooksGetCmd)
}

func runWebhooksGet(cmd *cobra.Command, args []string) error {
	webhookID := args[0]

	c := NewClient()
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	resp, err := c.Get(cmd.Context(), "/webhooks/"+webhookID, nil)
	if err != nil {
		return fmt.Errorf("requesting webhook: %w", err)
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
