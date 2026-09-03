package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoseFindingsRestoreRequestAndModes(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, map[string]any{"dismissed": false}, body)
		w.Write([]byte(findingMutationResponse(false)))
	})
	out, _, err := executeCommand("rose", "findings", "restore", roseTestRepositoryID, "otel-aabbccddeeff")
	require.NoError(t, err)
	assert.Equal(t, "Finding otel-aabbccddeeff is active.\n", out)

	out, _, err = executeCommand("rose", "findings", "restore", roseTestRepositoryID, "otel-aabbccddeeff", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)

	out, _, err = executeCommand("rose", "findings", "restore", roseTestRepositoryID, "otel-aabbccddeeff", "--json")
	require.NoError(t, err)
	assert.JSONEq(t, findingMutationResponse(false), out)
}

func TestRoseFindingsRestoreValidationBeforeRequest(t *testing.T) {
	requests := 0
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) { requests++ })
	for _, args := range [][]string{
		{"rose", "findings", "restore", "bad", "otel-aabbccddeeff"},
		{"rose", "findings", "restore", roseTestRepositoryID, "bad"},
	} {
		_, _, err := executeCommand(args...)
		require.Error(t, err)
		assert.Equal(t, 2, client.ExitCodeFromError(err))
	}
	assert.Zero(t, requests)
}

func TestRoseFindingsRestoreAPIErrors(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   int
		code     string
		message  string
		exitCode int
	}{
		{"not found", http.StatusNotFound, "FINDING_NOT_FOUND", "Finding not found", 4},
		{"upstream failure", http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				fmt.Fprintf(w, `{"error":{"code":%q,"message":%q},"meta":{"trace_id":"trace-error"}}`, tc.code, tc.message)
			})
			_, stderr, err := executeCommand("rose", "findings", "restore", roseTestRepositoryID, "otel-aabbccddeeff")
			require.Error(t, err)
			assert.Equal(t, tc.exitCode, client.ExitCodeFromError(err))
			assert.Contains(t, stderr, tc.message)
			assert.Contains(t, stderr, "trace-error")
		})
	}
}
