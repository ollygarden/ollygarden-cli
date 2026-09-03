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

const repositoryActivationResponse = `{"data":{"id":"22222222-2222-2222-2222-222222222222","is_active":true,"active_repo_count":3,"repo_limit":3},"meta":{"trace_id":"trace-2"}}`

func TestRoseRepositoriesActivateRequestAndModes(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		var body map[string]bool
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, map[string]bool{"is_active": true}, body)
		w.Write([]byte(repositoryActivationResponse))
	})
	out, _, err := executeCommand("rose", "repositories", "activate", roseTestRepositoryID)
	require.NoError(t, err)
	assert.Contains(t, out, "is active (3/3 active)")
	out, _, err = executeCommand("rose", "repositories", "activate", roseTestRepositoryID, "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
	out, _, err = executeCommand("rose", "repositories", "activate", roseTestRepositoryID, "--json")
	require.NoError(t, err)
	assert.JSONEq(t, repositoryActivationResponse, out)
}

func TestRoseRepositoryMutationErrors(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusNotFound, http.StatusConflict, http.StatusRequestEntityTooLarge, http.StatusUnprocessableEntity, http.StatusBadGateway} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				fmt.Fprintf(w, `{"error":{"code":"UPSTREAM_ERROR","message":"status %d"},"meta":{"trace_id":"error-trace"}}`, status)
			})
			_, stderr, err := executeCommand("rose", "repositories", "activate", roseTestRepositoryID)
			require.Error(t, err)
			var apiErr *client.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, status, apiErr.StatusCode)
			assert.Contains(t, stderr, fmt.Sprintf("status %d", status))
		})
	}
}
