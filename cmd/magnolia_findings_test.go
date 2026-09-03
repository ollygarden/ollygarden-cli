package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
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
