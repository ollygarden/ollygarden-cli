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
	apiKeysListLimit  int
	apiKeysListOffset int
)

type apiKeyMetadata struct {
	ID          string `json:"id"`
	KeyPreview  string `json:"key_preview"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	LastSeenAt  string `json:"last_seen_at"`
}

var apiKeysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List API keys",
	Args:  cobra.NoArgs,
	RunE:  runAPIKeysList,
}

func init() {
	apiKeysCmd.AddCommand(apiKeysListCmd)
	apiKeysListCmd.Flags().IntVar(&apiKeysListLimit, "limit", 50, "Maximum number of results (1-100)")
	apiKeysListCmd.Flags().IntVar(&apiKeysListOffset, "offset", 0, "Number of results to skip (≥0)")
}

func runAPIKeysList(cmd *cobra.Command, args []string) error {
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)
	if apiKeysListLimit < 1 || apiKeysListLimit > 100 {
		return apiKeysInvalidParameters(f, "--limit must be between 1 and 100")
	}
	if apiKeysListOffset < 0 {
		return apiKeysInvalidParameters(f, "--offset must be >= 0")
	}

	query := url.Values{}
	query.Set("limit", strconv.Itoa(apiKeysListLimit))
	query.Set("offset", strconv.Itoa(apiKeysListOffset))

	resp, err := NewClient().Get(cmd.Context(), "/api-keys", query)
	if err != nil {
		return fmt.Errorf("requesting API keys: %w", err)
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

	var keys []apiKeyMetadata
	if err := json.Unmarshal(apiResp.Data, &keys); err != nil {
		return fmt.Errorf("parsing API keys data: %w", err)
	}

	rows := make([][]string, len(keys))
	for i, key := range keys {
		rows[i] = []string{
			key.ID,
			key.Description,
			key.KeyPreview,
			key.CreatedAt,
			orDash(key.LastSeenAt),
		}
	}
	f.PrintTable([]string{"ID", "DESCRIPTION", "KEY", "CREATED", "LAST USED"}, rows)
	if apiResp.Meta.HasMore {
		f.PrintPaginationHint(apiResp.Meta.Total, apiKeysListOffset, apiKeysListLimit)
	}
	return nil
}
