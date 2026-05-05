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
	webhooksListLimit  int
	webhooksListOffset int
)

type webhookItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	IsEnabled   bool   `json:"is_enabled"`
	MinSeverity string `json:"min_severity"`
}

var webhooksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List webhooks",
	Args:  cobra.NoArgs,
	RunE:  runWebhooksList,
}

func init() {
	webhooksCmd.AddCommand(webhooksListCmd)
	webhooksListCmd.Flags().IntVar(&webhooksListLimit, "limit", 50, "Maximum number of results (1-100)")
	webhooksListCmd.Flags().IntVar(&webhooksListOffset, "offset", 0, "Number of results to skip (≥0)")
}

func runWebhooksList(cmd *cobra.Command, args []string) error {
	if webhooksListLimit < 1 || webhooksListLimit > 100 {
		return fmt.Errorf("--limit must be between 1 and 100")
	}
	if webhooksListOffset < 0 {
		return fmt.Errorf("--offset must be >= 0")
	}

	c := NewClient()
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	query := url.Values{}
	query.Set("limit", strconv.Itoa(webhooksListLimit))
	query.Set("offset", strconv.Itoa(webhooksListOffset))

	resp, err := c.Get(cmd.Context(), "/webhooks", query)
	if err != nil {
		return fmt.Errorf("requesting webhooks: %w", err)
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

	var webhooks []webhookItem
	if err := json.Unmarshal(apiResp.Data, &webhooks); err != nil {
		return fmt.Errorf("parsing webhooks data: %w", err)
	}

	headers := []string{"ID", "NAME", "URL", "ENABLED", "SEVERITY"}
	rows := make([][]string, len(webhooks))
	for i, w := range webhooks {
		rows[i] = []string{w.ID, w.Name, w.URL, strconv.FormatBool(w.IsEnabled), w.MinSeverity}
	}

	f.PrintTable(headers, rows)

	if apiResp.Meta.HasMore {
		f.PrintPaginationHint(apiResp.Meta.Total, webhooksListOffset, webhooksListLimit)
	}

	return nil
}
