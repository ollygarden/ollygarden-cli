package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoseExecutionsReviewRequestAndOutputModes(t *testing.T) {
	response := `{"data":{"status":"scheduled","executionId":"11111111-1111-1111-1111-111111111111","message":"Review scheduled"},"meta":{"trace_id":"trace"}}`
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/rose/codebase/review/execute", r.URL.Path)
		var body roseReviewExecutionRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, roseTestRepositoryID, body.RepositoryID)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(response))
	})
	out, _, err := executeCommand("rose", "executions", "review", roseTestRepositoryID)
	require.NoError(t, err)
	assert.Contains(t, out, "Status:       scheduled")
	assert.Contains(t, out, "Execution ID: "+roseTestExecutionID)
	out, _, err = executeCommand("rose", "executions", "review", roseTestRepositoryID, "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
	out, _, err = executeCommand("rose", "executions", "review", roseTestRepositoryID, "--json")
	require.NoError(t, err)
	assert.JSONEq(t, response, out)
}

func TestRoseExecutionsReviewSuccessfulSkippedOutcome(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"status":"cooldown","message":"Try later"},"meta":{}}`))
	})
	out, _, err := executeCommand("rose", "executions", "review", roseTestRepositoryID)
	require.NoError(t, err)
	assert.Contains(t, out, "Status:       cooldown")
	assert.NotContains(t, out, "Execution ID:")
}
