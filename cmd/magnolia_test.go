package cmd

import (
	"net/http"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/exitcode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The error path of runMagnoliaWidget is shared by all analytics widget
// commands, so it is exercised once here rather than per command.

func TestMagnoliaWidgetReportNotReadyMapsToNotFound(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"REPORT_NOT_READY","message":"report not ready"},"meta":{}}`))
	})
	_, stderr, err := executeCommand("analytics", "summary")
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, exitcode.NotFound, apiErr.ExitCode())
	assert.Contains(t, stderr, "report not ready")
}

func TestMagnoliaWidgetServerErrorSurfaces(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"code":"UPSTREAM_ERROR","message":"ClickHouse query failed"},"meta":{}}`))
	})
	_, stderr, err := executeCommand("analytics", "traces")
	assert.Error(t, err)
	assert.Contains(t, stderr, "ClickHouse query failed")
}

func TestMagnoliaWidgetLegacyMagnoliaGroupRemoved(t *testing.T) {
	_, _, err := executeCommand("magnolia", "report")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown command "magnolia"`)
}
