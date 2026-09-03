package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoseExecutionsFixRequestAndAlreadyPending(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body roseFixExecutionRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, []string{"otel-3f9a1c2b7d4e"}, body.FindingIDs)
		require.NotNil(t, body.IssueNumber)
		assert.Equal(t, 42, *body.IssueNumber)
		_, _ = w.Write([]byte(`{"data":{"status":"already_pending","message":"Fix pending"},"meta":{}}`))
	})
	out, _, err := executeCommand("rose", "executions", "fix", roseTestRepositoryID, "--finding-id", "otel-3f9a1c2b7d4e", "--issue-number", "42")
	require.NoError(t, err)
	assert.Contains(t, out, "already_pending")
}

func TestRoseExecutionsFixValidationAndError(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"UPSTREAM_ERROR","message":"repository not found"},"meta":{"trace_id":"t"}}`))
	})
	_, stderr, err := executeCommand("rose", "executions", "fix", roseTestRepositoryID)
	require.Error(t, err)
	assert.Contains(t, stderr, "--finding-id")
	_, stderr, err = executeCommand("rose", "executions", "fix", roseTestRepositoryID, "--finding-id", "otel-3f9a1c2b7d4e")
	require.Error(t, err)
	assert.Contains(t, stderr, "repository not found")
}
