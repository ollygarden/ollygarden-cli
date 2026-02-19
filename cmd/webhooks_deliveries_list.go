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

var (
	webhooksDeliveriesListLimit  int
	webhooksDeliveriesListOffset int
)

type deliveryItem struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	HTTPStatusCode int    `json:"http_status_code"`
	AttemptNumber  int    `json:"attempt_number"`
	CreatedAt      string `json:"created_at"`
}

var webhooksDeliveriesListCmd = &cobra.Command{
	Use:   "list <webhook-id>",
	Short: "List deliveries for a webhook",
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhooksDeliveriesList,
}

func init() {
	webhooksDeliveriesCmd.AddCommand(webhooksDeliveriesListCmd)
	webhooksDeliveriesListCmd.Flags().IntVar(&webhooksDeliveriesListLimit, "limit", 50, "Maximum number of results (1-100)")
	webhooksDeliveriesListCmd.Flags().IntVar(&webhooksDeliveriesListOffset, "offset", 0, "Number of results to skip (≥0)")
}

func runWebhooksDeliveriesList(cmd *cobra.Command, args []string) error {
	if webhooksDeliveriesListLimit < 1 || webhooksDeliveriesListLimit > 100 {
		return fmt.Errorf("Error: --limit must be between 1 and 100")
	}
	if webhooksDeliveriesListOffset < 0 {
		return fmt.Errorf("Error: --offset must be >= 0")
	}

	webhookID := args[0]

	c := NewClient()
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	query := url.Values{}
	query.Set("limit", strconv.Itoa(webhooksDeliveriesListLimit))
	query.Set("offset", strconv.Itoa(webhooksDeliveriesListOffset))

	resp, err := c.Get(cmd.Context(), "/webhooks/"+webhookID+"/deliveries", query)
	if err != nil {
		return fmt.Errorf("requesting webhook deliveries: %w", err)
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

	var deliveries []deliveryItem
	if err := json.Unmarshal(apiResp.Data, &deliveries); err != nil {
		return fmt.Errorf("parsing deliveries data: %w", err)
	}

	headers := []string{"ID", "STATUS", "HTTP", "ATTEMPTS", "CREATED"}
	rows := make([][]string, len(deliveries))
	for i, d := range deliveries {
		httpCode := "—"
		if d.HTTPStatusCode != 0 {
			httpCode = strconv.Itoa(d.HTTPStatusCode)
		}
		rows[i] = []string{d.ID, d.Status, httpCode, strconv.Itoa(d.AttemptNumber), d.CreatedAt}
	}

	f.PrintTable(headers, rows)

	if apiResp.Meta.HasMore {
		f.PrintPaginationHint(apiResp.Meta.Total, webhooksDeliveriesListOffset, webhooksDeliveriesListLimit)
	}

	return nil
}
