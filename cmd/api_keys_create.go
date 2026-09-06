package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

var apiKeysCreateDescription string

type apiKeyCreateRequest struct {
	Description string `json:"description"`
}

type createdAPIKey struct {
	apiKeyMetadata
	Key string `json:"key"`
}

var apiKeysCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an API key",
	Args:  cobra.NoArgs,
	RunE:  runAPIKeysCreate,
}

func init() {
	apiKeysCmd.AddCommand(apiKeysCreateCmd)
	apiKeysCreateCmd.Flags().StringVar(&apiKeysCreateDescription, "description", "", "API key description (required, 1-100 characters)")
}

func runAPIKeysCreate(cmd *cobra.Command, args []string) error {
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)
	description := strings.TrimSpace(apiKeysCreateDescription)
	if !utf8.ValidString(description) {
		return apiKeysInvalidParameters(f, "--description must be valid UTF-8")
	}
	if count := utf8.RuneCountInString(description); count < 1 || count > 100 {
		return apiKeysInvalidParameters(f, "--description must be between 1 and 100 characters")
	}

	resp, err := NewClient().Post(cmd.Context(), "/api-keys", apiKeyCreateRequest{Description: description})
	if err != nil {
		return fmt.Errorf("creating API key: %w", err)
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

	var key createdAPIKey
	if err := json.Unmarshal(apiResp.Data, &key); err != nil {
		return fmt.Errorf("parsing created API key data: %w", err)
	}
	if key.Key == "" {
		return fmt.Errorf("created API key response did not include the one-time key")
	}

	if f.IsJSON() {
		raw, _ := json.Marshal(apiResp)
		f.PrintJSON(raw)
		return nil
	}
	if f.IsQuiet() {
		return nil
	}

	f.PrintKeyValue([]output.KVPair{
		{Key: "ID", Value: key.ID},
		{Key: "Description", Value: key.Description},
		{Key: "API Key", Value: key.Key},
		{Key: "Created", Value: key.CreatedAt},
	})
	fmt.Fprintln(cmd.ErrOrStderr(), "Save this API key now. It cannot be retrieved again.")
	return nil
}

func apiKeysInvalidParameters(f *output.Formatter, message string) error {
	errResp := &client.ErrorResponse{
		Error: client.ErrorDetail{Code: "INVALID_PARAMETERS", Message: message},
	}
	apiErr := &client.APIError{StatusCode: http.StatusBadRequest, ErrorResponse: errResp}
	raw, _ := json.Marshal(errResp)
	f.PrintError(apiErr.Error(), raw)
	return apiErr
}
