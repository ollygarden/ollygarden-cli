package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDeactivateServer(t *testing.T, requests *int) {
	oldTerminal, oldReader := stdinIsTerminal, stdinReader
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		(*requests)++
		var body map[string]bool
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, map[string]bool{"is_active": false}, body)
		w.Write([]byte(`{"data":{"id":"22222222-2222-2222-2222-222222222222","is_active":false,"active_repo_count":2,"repo_limit":3},"meta":{}}`))
	})
	t.Cleanup(func() {
		roseRepositoriesDeactivateConfirm = false
		stdinIsTerminal, stdinReader = oldTerminal, oldReader
	})
}

func TestRoseRepositoriesDeactivateNonTTYRequiresConfirm(t *testing.T) {
	requests := 0
	setupDeactivateServer(t, &requests)
	stdinIsTerminal = func() bool { return false }
	_, _, err := executeCommand("rose", "repositories", "deactivate", roseTestRepositoryID)
	require.Error(t, err)
	assert.Equal(t, 2, client.ExitCodeFromError(err))
	assert.Zero(t, requests)
}

func TestRoseRepositoriesDeactivateTTYAnswers(t *testing.T) {
	for _, tc := range []struct {
		name, answer string
		wantRequests int
	}{
		{"yes", "y\n", 1}, {"no", "n\n", 0}, {"default", "\n", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			setupDeactivateServer(t, &requests)
			stdinIsTerminal = func() bool { return true }
			stdinReader = io.NopCloser(strings.NewReader(tc.answer))
			_, stderr, err := executeCommand("rose", "repositories", "deactivate", roseTestRepositoryID, "--quiet")
			require.NoError(t, err)
			assert.Equal(t, tc.wantRequests, requests)
			assert.Contains(t, stderr, "Deactivate repository (id: "+roseTestRepositoryID+")? [y/N]:")
		})
	}
}

func TestRoseRepositoriesDeactivateConfirm(t *testing.T) {
	requests := 0
	setupDeactivateServer(t, &requests)
	stdinIsTerminal = func() bool { return false }
	out, stderr, err := executeCommand("rose", "repositories", "deactivate", roseTestRepositoryID, "--confirm")
	require.NoError(t, err)
	assert.Equal(t, 1, requests)
	assert.Contains(t, out, "is inactive (2/3 active)")
	assert.Empty(t, stderr)
}
