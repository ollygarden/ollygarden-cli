package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoseFindingsGetHuman(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/rose/repositories/repo-1", r.URL.Path)
		w.Write([]byte(roseRepositoryDetailResponse))
	})

	out, _, err := executeCommand("rose", "findings", "get", "repo-1", "otel-3f9a1c2b7d4e")
	require.NoError(t, err)
	assert.Contains(t, out, "otel-3f9a1c2b7d4e")
	assert.Contains(t, out, "critical")
	assert.Contains(t, out, "PII (email) logged in span attributes")
	assert.Contains(t, out, "acme/checkout (repo-1)")
	assert.Contains(t, out, "exec-1")
	assert.Contains(t, out, "active")
	assert.Contains(t, out, "pending")
	assert.Contains(t, out, "#142 (open)")
	assert.Contains(t, out, "2026-08-20T14:03:11Z") // normalized timestamp
	assert.Contains(t, out, "Summary:")
	assert.Contains(t, out, "user.email exported on every request.")
	assert.Contains(t, out, "Why:")
	assert.Contains(t, out, "Fix:")
	assert.Contains(t, out, "internal/http/middleware.go:88")
	assert.Contains(t, out, "sets span attribute")
}

func TestRoseFindingsGetResolvedNullFields(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(roseRepositoryDetailResponse))
	})

	out, _, err := executeCommand("rose", "findings", "get", "repo-1", "otel-8b2e4d1f0a6c")
	require.NoError(t, err)
	assert.Contains(t, out, "resolved") // checked=true
	assert.Contains(t, out, "Missing status code")
	assert.Contains(t, out, "—") // em dash for null category/PR/timestamps
	assert.NotContains(t, out, "Summary:")
	assert.NotContains(t, out, "Locations:")
}

func TestRoseFindingsGetJSON(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(roseRepositoryDetailResponse))
	})

	out, _, err := executeCommand("rose", "findings", "get", "repo-1", "otel-3f9a1c2b7d4e", "--json")
	require.NoError(t, err)
	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "trace-repo", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), `"finding_id":"otel-3f9a1c2b7d4e"`)
	assert.NotContains(t, string(envelope.Data), "otel-8b2e4d1f0a6c") // only the requested finding
}

func TestRoseFindingsGetNotFoundInRepo(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(roseRepositoryDetailResponse))
	})

	_, stderr, err := executeCommand("rose", "findings", "get", "repo-1", "otel-000000000000")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 4, apiErr.ExitCode())
	assert.Equal(t, "FINDING_NOT_FOUND", apiErr.ErrorResponse.Error.Code)
	assert.Contains(t, stderr, "not found in repository")
}

func TestRoseFindingsGetRepo404(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"code":"UPSTREAM_ERROR","message":"repository not found"},"meta":{"trace_id":"t1"}}`))
	})

	_, stderr, err := executeCommand("rose", "findings", "get", "missing-repo", "otel-3f9a1c2b7d4e")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 4, apiErr.ExitCode())
	assert.Contains(t, stderr, "repository not found")
}

func TestRoseFindingsGetMissingArgs(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {})

	_, _, err := executeCommand("rose", "findings", "get", "repo-1")
	require.Error(t, err)
}
