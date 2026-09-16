package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

// magnoliaWidgetEnvelope mirrors the /api/v3/magnolia response envelope. Data
// stays raw so each widget command decodes only the shape it renders.
type magnoliaWidgetEnvelope struct {
	OrgID       string          `json:"orgId"`
	RunDate     string          `json:"runDate"`
	GeneratedAt string          `json:"generatedAt"`
	Data        json.RawMessage `json:"data"`
}

// runMagnoliaWidget fetches one /api/v3/magnolia widget and prints it. The
// organization is inferred from authentication server-side, so no orgId is
// sent. In human mode the shared envelope header is printed as key-value
// pairs, then render draws the widget-specific payload.
func runMagnoliaWidget(cmd *cobra.Command, path string, render func(f *output.Formatter, data json.RawMessage) error) error {
	f := newFormatter(cmd)
	resp, err := NewVersionedClient(client.V3).Get(cmd.Context(), path, nil)
	if err != nil {
		return fmt.Errorf("requesting %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		var errorResponse client.ErrorResponse
		apiErr := &client.APIError{StatusCode: resp.StatusCode}
		if json.Unmarshal(body, &errorResponse) == nil {
			apiErr.ErrorResponse = &errorResponse
		}
		f.PrintError(apiErr.Error(), body)
		return apiErr
	}
	if f.IsJSON() {
		f.PrintJSON(body)
		return nil
	}
	if f.IsQuiet() {
		return nil
	}
	var envelope magnoliaWidgetEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("parsing %s response: %w", path, err)
	}
	f.PrintKeyValue([]output.KVPair{
		{Key: "Organization", Value: envelope.OrgID},
		{Key: "Run date", Value: envelope.RunDate},
		{Key: "Generated", Value: envelope.GeneratedAt},
	})
	return render(f, envelope.Data)
}
