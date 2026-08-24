package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const roseRepositoriesListResponse = `{"data":{"data":[
	{"id":"inst-1","vcs_provider":"github","provider_installation_status":"active","active_repo_count":2,"repos":[
		{"id":"repo-1","repo_full_name":"acme/checkout","repo_url":"https://github.com/acme/checkout","is_active":true,"last_scanned_at":"2026-08-22 02:10:44.1+00","active_findings_count":14,"finding_counts":{"critical":2,"high":5,"medium":6,"low":1,"suggestion":0}},
		{"id":"repo-2","repo_full_name":"acme/docs","repo_url":"https://github.com/acme/docs","is_active":false,"last_scanned_at":null,"active_findings_count":0,"finding_counts":{"critical":0,"high":0,"medium":0,"low":0,"suggestion":0}}
	]}
],"pagination":{"limit":1,"offset":0,"total":1,"hasMore":false}},"meta":{"timestamp":"2026-08-22T02:20:00Z","trace_id":"trace-repos"}}`

func TestRoseRepositoriesListHuman(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/rose/repositories", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		w.Write([]byte(roseRepositoriesListResponse))
	})

	out, _, err := executeCommand("rose", "repositories", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "repo-1")
	assert.Contains(t, out, "acme/checkout")
	assert.Contains(t, out, "yes")
	assert.Contains(t, out, "2026-08-22T02:10:44Z")
	assert.Contains(t, out, "14")
	assert.Contains(t, out, "2/5/6/1")
	assert.Contains(t, out, "acme/docs")
	assert.Contains(t, out, "no")
	assert.Contains(t, out, "—") // never scanned
}

func TestRoseRepositoriesListJSON(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(roseRepositoriesListResponse))
	})

	out, _, err := executeCommand("rose", "repositories", "list", "--json")
	require.NoError(t, err)
	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "trace-repos", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), `"vcs_provider":"github"`) // installation kept in JSON
}

func TestRoseRepositoriesListQuiet(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(roseRepositoriesListResponse))
	})

	out, _, err := executeCommand("rose", "repositories", "list", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestRoseRepositoriesList403(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":{"code":"FORBIDDEN","message":"forbidden"},"meta":{"trace_id":"t1"}}`))
	})

	_, stderr, err := executeCommand("rose", "repositories", "list")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 1, apiErr.ExitCode())
	assert.Contains(t, stderr, "forbidden")
}
