package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const roseExecutionDetailResponse = `{"data":{
	"execution":{"id":"exec-1","executionType":"fix","status":"failed","triggerSource":"manual","ref":"main","commitSha":"a4f49a9052528b9615692e38a46821cb2f82b811","startedAt":"2026-08-22 02:10:44.1+00","completedAt":"2026-08-22 02:18:03.2+00","repositoryId":"repo-1","installationId":"inst-1","repoOwner":"acme","repoName":"checkout"},
	"currentPhase":{"id":"ph-2","key":"implementing_fix","label":"Implement fix","state":"failed"},
	"phases":[
		{"id":"ph-1","key":"checkout_branch","label":"Checkout branch","order":1,"state":"completed","startedAt":"2026-08-22 02:10:44.1+00","completedAt":"2026-08-22 02:10:51+00"},
		{"id":"ph-2","key":"implementing_fix","label":"Implement fix","order":2,"state":"failed","startedAt":"2026-08-22 02:10:51+00","completedAt":"2026-08-22 02:18:03.2+00"},
		{"id":"ph-3","key":"open_pr","label":"Open pull request","order":3,"state":"pending","startedAt":null,"completedAt":null}
	],
	"events":[{"createdAt":"2026-08-22 02:18:03.2+00","eventName":"execution.timed_out","message":"timed out","phaseKey":"implementing_fix","transition":"timeout"}],
	"running":false,"lastSeenAt":"2026-08-22 02:17:18+00","activity":null},"meta":{"timestamp":"2026-08-22T02:20:00Z","trace_id":"trace-exec"}}`

func TestRoseExecutionsGetHuman(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/rose/executions/exec-1", r.URL.Path)
		w.Write([]byte(roseExecutionDetailResponse))
	})

	out, _, err := executeCommand("rose", "executions", "get", "exec-1")
	require.NoError(t, err)
	assert.Contains(t, out, "exec-1")
	assert.Contains(t, out, "fix")
	assert.Contains(t, out, "failed")
	assert.Contains(t, out, "manual")
	assert.Contains(t, out, "acme/checkout (repo-1)")
	assert.Contains(t, out, "main")
	assert.Contains(t, out, "a4f49a9") // short SHA
	assert.NotContains(t, out, "a4f49a9052528b96")
	assert.Contains(t, out, "2026-08-22T02:10:44Z")
	assert.Contains(t, out, "no") // not running
	assert.Contains(t, out, "Implement fix (failed)")
	assert.Contains(t, out, "Checkout branch")
	assert.Contains(t, out, "Open pull request")
	assert.Contains(t, out, "pending")
	assert.Contains(t, out, "—") // pending phase timestamps
}

func TestRoseExecutionsGetJSON(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(roseExecutionDetailResponse))
	})

	out, _, err := executeCommand("rose", "executions", "get", "exec-1", "--json")
	require.NoError(t, err)
	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "trace-exec", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), `"events"`) // full payload passes through
}

func TestRoseExecutionsGetQuiet(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(roseExecutionDetailResponse))
	})

	out, _, err := executeCommand("rose", "executions", "get", "exec-1", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestRoseExecutionsGet404(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"code":"UPSTREAM_ERROR","message":"execution not found"},"meta":{"trace_id":"t1"}}`))
	})

	_, stderr, err := executeCommand("rose", "executions", "get", "missing")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 4, apiErr.ExitCode())
	assert.Contains(t, stderr, "execution not found")
}

func TestRoseExecutionsGetMissingArg(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {})

	_, _, err := executeCommand("rose", "executions", "get")
	require.Error(t, err)
}
