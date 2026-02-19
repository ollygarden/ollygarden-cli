package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

type webhookTestResponse struct {
	Success      bool   `json:"success"`
	StatusCode   int    `json:"status_code"`
	ResponseBody string `json:"response_body"`
}

var webhooksTestCmd = &cobra.Command{
	Use:   "test <webhook-id>",
	Short: "Test a webhook",
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhooksTest,
}

func init() {
	webhooksCmd.AddCommand(webhooksTestCmd)
}

func runWebhooksTest(cmd *cobra.Command, args []string) error {
	webhookID := args[0]

	c := NewClient()
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	resp, err := c.Post(cmd.Context(), "/webhooks/"+webhookID+"/test", nil)
	if err != nil {
		return fmt.Errorf("testing webhook: %w", err)
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

	var tr webhookTestResponse
	if err := json.Unmarshal(apiResp.Data, &tr); err != nil {
		return fmt.Errorf("parsing test response: %w", err)
	}

	responseBody := tr.ResponseBody
	if responseBody == "" {
		responseBody = "(empty)"
	}

	pairs := []output.KVPair{
		{Key: "Success", Value: strconv.FormatBool(tr.Success)},
		{Key: "Status Code", Value: strconv.Itoa(tr.StatusCode)},
		{Key: "Response Body", Value: responseBody},
	}

	f.PrintKeyValue(pairs)
	return nil
}
