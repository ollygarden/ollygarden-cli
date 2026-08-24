package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoseRepositoriesGetHuman(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/rose/repositories/repo-1", r.URL.Path)
		w.Write([]byte(roseRepositoryDetailResponse))
	})

	out, _, err := executeCommand("rose", "repositories", "get", "repo-1")
	require.NoError(t, err)
	assert.Contains(t, out, "repo-1")
	assert.Contains(t, out, "acme/checkout")
	assert.Contains(t, out, "https://github.com/acme/checkout")
	assert.Contains(t, out, "github")
	assert.Contains(t, out, "2026-08-22T02:10:44Z (sha 4f2e9c1)")
	assert.Contains(t, out, "#87")
	assert.Contains(t, out, "2 (critical 1, high 1, medium 0, low 0, suggestion 0)")
	assert.Contains(t, out, "Instrumentation:")
	assert.Contains(t, out, "traces, metrics")
	assert.Contains(t, out, "go.opentelemetry.io/otel")
	assert.Contains(t, out, "HTTP spans and runtime metrics.")
	assert.Contains(t, out, "Findings:")
	assert.Contains(t, out, "otel-3f9a1c2b7d4e")
	assert.Contains(t, out, "PII (email) logged in span attributes")
	assert.Contains(t, out, "#142 (open)")
}

func TestRoseRepositoriesGetNoInstrumentation(t *testing.T) {
	body := `{"data":{"repository":{"id":"repo-9","repo_full_name":"acme/new","repo_url":"https://github.com/acme/new","is_active":false,"vcs_provider":"github","repository_access_status":"active","last_scanned_at":null,"last_scanned_commit_sha":null,"dashboard_issue_number":null,"active_findings_count":0,"finding_counts":{"critical":0,"high":0,"medium":0,"low":0,"suggestion":0}},"instrumentation_metadata":null,"findings":[]},"meta":{"trace_id":"t1"}}`
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("rose", "repositories", "get", "repo-9")
	require.NoError(t, err)
	assert.Contains(t, out, "acme/new")
	assert.Contains(t, out, "—") // never scanned, no dashboard issue
	assert.NotContains(t, out, "Instrumentation:")
	assert.NotContains(t, out, "Findings:")
}

func TestRoseRepositoriesGetJSON(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(roseRepositoryDetailResponse))
	})

	out, _, err := executeCommand("rose", "repositories", "get", "repo-1", "--json")
	require.NoError(t, err)
	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "trace-repo", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), `"instrumentation_metadata"`)
}

func TestRoseRepositoriesGetQuiet(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(roseRepositoryDetailResponse))
	})

	out, _, err := executeCommand("rose", "repositories", "get", "repo-1", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestRoseRepositoriesGet404(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"code":"UPSTREAM_ERROR","message":"repository not found"},"meta":{"trace_id":"t1"}}`))
	})

	_, stderr, err := executeCommand("rose", "repositories", "get", "missing")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 4, apiErr.ExitCode())
	assert.Contains(t, stderr, "repository not found")
}

func TestRoseRepositoriesGetMissingArg(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {})

	_, _, err := executeCommand("rose", "repositories", "get")
	require.Error(t, err)
}
