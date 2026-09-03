package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

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
