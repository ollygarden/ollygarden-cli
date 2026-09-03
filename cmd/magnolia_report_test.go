package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMagnoliaReportHumanAndV2Path(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/magnolia/report", r.URL.Path)
		assert.Equal(t, "org_test", r.URL.Query().Get("orgId"))
		w.Write([]byte(`{"orgId":"org_test","generatedAt":"now","window":{"from":"a","to":"b"},"data":{"summary":{"totals":{"spans":3,"logs":4,"datapoints":5}}}}`))
	})
	out, _, err := executeCommand("magnolia", "report", "--org-id", "org_test")
	assert.NoError(t, err)
	assert.Contains(t, out, "Organization")
	assert.Contains(t, out, "org_test")
}

func TestMagnoliaReportJSONPreservesCustomEnvelope(t *testing.T) {
	body := `{"orgId":"org_test","window":{},"data":{"summary":{}}}`
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) })
	out, _, err := executeCommand("--json", "magnolia", "report", "--org-id", "org_test")
	assert.NoError(t, err)
	assert.JSONEq(t, body, out)
}
