package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findingMutationResponse(dismissed bool) string {
	return `{"data":{"finding_id":"otel-aabbccddeeff","dismissed":` + fmt.Sprint(dismissed) + `,"dismissed_at":null,"dismissed_reason":null},"meta":{"trace_id":"trace-1"}}`
}

func TestRoseFindingsDismissHumanAndRequest(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/rose/repositories/"+roseTestRepositoryID+"/findings/otel-aabbccddeeff", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, map[string]any{"dismissed": true, "dismissed_reason": "accepted risk"}, body)
		w.Write([]byte(findingMutationResponse(true)))
	})
	out, _, err := executeCommand("rose", "findings", "dismiss", roseTestRepositoryID, "otel-aabbccddeeff", "--reason", "accepted risk")
	require.NoError(t, err)
	assert.Equal(t, "Finding otel-aabbccddeeff is dismissed.\n", out)
}

func TestRoseFindingsDismissOmitsUnspecifiedReason(t *testing.T) {
	roseFindingsDismissReason = ""
	roseFindingsDismissCmd.Flags().Lookup("reason").Changed = false
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.NotContains(t, body, "dismissed_reason")
		w.Write([]byte(findingMutationResponse(true)))
	})
	_, _, err := executeCommand("rose", "findings", "dismiss", roseTestRepositoryID, "otel-aabbccddeeff", "--quiet")
	require.NoError(t, err)
}

func TestRoseFindingsDismissJSONPreservesEnvelope(t *testing.T) {
	want := findingMutationResponse(true)
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(want)) })
	out, _, err := executeCommand("rose", "findings", "dismiss", roseTestRepositoryID, "otel-aabbccddeeff", "--json")
	require.NoError(t, err)
	assert.JSONEq(t, want, out)
}

func TestRoseFindingsDismissValidationBeforeRequest(t *testing.T) {
	requests := 0
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) { requests++ })
	for _, args := range [][]string{
		{"rose", "findings", "dismiss", "bad", "otel-aabbccddeeff"},
		{"rose", "findings", "dismiss", roseTestRepositoryID, "bad"},
		{"rose", "findings", "dismiss", roseTestRepositoryID, "otel-aabbccddeeff", "--reason", strings.Repeat("x", 1001)},
	} {
		_, _, err := executeCommand(args...)
		require.Error(t, err)
		assert.Equal(t, 2, client.ExitCodeFromError(err))
	}
	assert.Zero(t, requests)
}
