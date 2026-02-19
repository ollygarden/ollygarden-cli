package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

type deliveryDetail struct {
	ID              string  `json:"id"`
	Status          string  `json:"status"`
	HTTPStatusCode  int     `json:"http_status_code"`
	AttemptNumber   int     `json:"attempt_number"`
	ErrorMessage    *string `json:"error_message"`
	IdempotencyKey  string  `json:"idempotency_key"`
	InsightID       string  `json:"insight_id"`
	WebhookConfigID string  `json:"webhook_config_id"`
	OrganizationID  string  `json:"organization_id"`
	CreatedAt       string  `json:"created_at"`
	CompletedAt     *string `json:"completed_at"`
}

var webhooksDeliveriesGetCmd = &cobra.Command{
	Use:   "get <webhook-id> <delivery-id>",
	Short: "Show delivery details",
	Args:  cobra.ExactArgs(2),
	RunE:  runWebhooksDeliveriesGet,
}

func init() {
	webhooksDeliveriesCmd.AddCommand(webhooksDeliveriesGetCmd)
}

func runWebhooksDeliveriesGet(cmd *cobra.Command, args []string) error {
	webhookID := args[0]
	deliveryID := args[1]

	c := NewClient()
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	resp, err := c.Get(cmd.Context(), "/webhooks/"+webhookID+"/deliveries/"+deliveryID, nil)
	if err != nil {
		return fmt.Errorf("requesting delivery: %w", err)
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

	var d deliveryDetail
	if err := json.Unmarshal(apiResp.Data, &d); err != nil {
		return fmt.Errorf("parsing delivery data: %w", err)
	}

	httpStatus := "—"
	if d.HTTPStatusCode != 0 {
		httpStatus = strconv.Itoa(d.HTTPStatusCode)
	}

	errorMsg := "—"
	if d.ErrorMessage != nil && *d.ErrorMessage != "" {
		errorMsg = *d.ErrorMessage
	}

	completedAt := "—"
	if d.CompletedAt != nil && *d.CompletedAt != "" {
		completedAt = *d.CompletedAt
	}

	pairs := []output.KVPair{
		{Key: "ID", Value: d.ID},
		{Key: "Status", Value: d.Status},
		{Key: "HTTP Status", Value: httpStatus},
		{Key: "Attempts", Value: strconv.Itoa(d.AttemptNumber)},
		{Key: "Error", Value: errorMsg},
		{Key: "Insight ID", Value: d.InsightID},
		{Key: "Webhook ID", Value: d.WebhookConfigID},
		{Key: "Idempotency Key", Value: d.IdempotencyKey},
		{Key: "Created", Value: d.CreatedAt},
		{Key: "Completed", Value: completedAt},
	}

	f.PrintKeyValue(pairs)
	return nil
}
