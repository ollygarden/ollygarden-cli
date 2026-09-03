package cmd

import (
	"net/http"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMagnoliaFindingsHumanAndRawJSON(t *testing.T) {
	body := `{"run":{"organization_id":"org_test","run_id":"run-1"},"summary":"ok","findings":[{}],"groups":[{"title":"Reduce duplicates","severity":"high","pillar":"waste","finding_ids":["f1"]}]}`
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/magnolia/findings", r.URL.Path)
		w.Write([]byte(body))
	})
	out, _, err := executeCommand("magnolia", "findings", "--org-id", "org_test")
	assert.NoError(t, err)
	assert.Contains(t, out, "Reduce duplicates")

	out, _, err = executeCommand("--json", "magnolia", "findings", "--org-id", "org_test")
	assert.NoError(t, err)
	assert.JSONEq(t, body, out)
}

func TestMagnoliaFindingsQuiet(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"run":{},"findings":[],"groups":[]}`))
	})
	out, stderr, err := executeCommand("magnolia", "findings", "--org-id", "org_test", "--quiet")
	assert.NoError(t, err)
	assert.Empty(t, out)
	assert.Empty(t, stderr)
}

func TestMagnoliaFindingsAPIError(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"MAGNOLIA_FINDINGS_NOT_FOUND","message":"No Magnolia findings found"},"meta":{"trace_id":"trace-404"}}`))
	})
	_, stderr, err := executeCommand("magnolia", "findings", "--org-id", "org_test")
	require.Error(t, err)
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 4, apiErr.ExitCode())
	assert.Contains(t, stderr, "No Magnolia findings found")
	assert.Contains(t, stderr, "trace-404")
}
