package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoseExecutionsInstrumentRequestAndNoOpOutcome(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body roseInstrumentationExecutionRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "general", body.Type)
		_, _ = w.Write([]byte(`{"data":{"status":"not_implemented","message":"General instrumentation is not yet available."},"meta":{}}`))
	})
	out, _, err := executeCommand("rose", "executions", "instrument", roseTestRepositoryID, "--type", "general")
	require.NoError(t, err)
	assert.Contains(t, out, "not_implemented")
}

func TestRoseExecutionsInstrumentValidation(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) { t.Fatal("unexpected request") })
	_, stderr, err := executeCommand("rose", "executions", "instrument", roseTestRepositoryID, "--type", "full")
	require.Error(t, err)
	assert.Contains(t, stderr, "minimum, general")
}
