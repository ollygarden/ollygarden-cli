package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const roseFindingsSummaryResponse = `{"data":{"total":29,"by_severity":[{"severity":"critical","count":3},{"severity":"high","count":8}],"by_category":[{"category":"Sensitive Data","count":3},{"category":"Governance","count":12}]},"meta":{"timestamp":"2026-08-22T02:20:00Z","trace_id":"trace-sum"}}`

func TestRoseFindingsSummaryHuman(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/rose/findings/summary", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		w.Write([]byte(roseFindingsSummaryResponse))
	})

	out, _, err := executeCommand("rose", "findings", "summary")
	require.NoError(t, err)
	assert.Contains(t, out, "Total active findings:  29")
	assert.Contains(t, out, "SEVERITY")
	assert.Contains(t, out, "critical")
	assert.Contains(t, out, "CATEGORY")
	assert.Contains(t, out, "Sensitive Data")
	assert.Contains(t, out, "12")
}

func TestRoseFindingsSummaryJSON(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(roseFindingsSummaryResponse))
	})

	out, _, err := executeCommand("rose", "findings", "summary", "--json")
	require.NoError(t, err)
	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "trace-sum", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), `"total":29`)
}

func TestRoseFindingsSummaryQuiet(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(roseFindingsSummaryResponse))
	})

	out, _, err := executeCommand("rose", "findings", "summary", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestRoseFindingsSummary401(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t1"}}`))
	})

	_, stderr, err := executeCommand("rose", "findings", "summary")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestRoseFindingsSummary503(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":{"code":"SERVICE_UNAVAILABLE","message":"Rose service is not configured"},"meta":{"trace_id":"t1"}}`))
	})

	_, stderr, err := executeCommand("rose", "findings", "summary")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Rose service is not configured")
}
