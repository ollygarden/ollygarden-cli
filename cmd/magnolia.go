package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

var magnoliaCmd = &cobra.Command{
	Use:    "magnolia",
	Short:  "Read Magnolia telemetry analysis artifacts",
	Args:   cobra.NoArgs,
	Hidden: true,
}

func init() {
	rootCmd.AddCommand(magnoliaCmd)
}

func magnoliaGet(cmd *cobra.Command, f *output.Formatter, path, orgID string) ([]byte, error) {
	query := url.Values{}
	if orgID != "" {
		query.Set("orgId", orgID)
	}
	resp, err := NewVersionedClient(client.V2).Get(cmd.Context(), path, query)
	if err != nil {
		return nil, fmt.Errorf("requesting %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode < http.StatusBadRequest {
		return body, nil
	}
	var errorResponse client.ErrorResponse
	apiErr := &client.APIError{StatusCode: resp.StatusCode}
	if json.Unmarshal(body, &errorResponse) == nil {
		apiErr.ErrorResponse = &errorResponse
	}
	f.PrintError(apiErr.Error(), body)
	return nil, apiErr
}
