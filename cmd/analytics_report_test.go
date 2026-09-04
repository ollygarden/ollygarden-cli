package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnalyticsReportHumanAndV2PathWithoutOrgQuery(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/magnolia/report", r.URL.Path)
		assert.NotContains(t, r.URL.Query(), "orgId")
		w.Write([]byte(`{"orgId":"org_test","generatedAt":"now","window":{"from":"a","to":"b"},"data":{"summary":{"totals":{"spans":3,"logs":4,"datapoints":5}}}}`))
	})
	out, _, err := executeCommand("analytics", "report")
	assert.NoError(t, err)
	assert.Contains(t, out, "Organization")
	assert.Contains(t, out, "org_test")
}

func TestAnalyticsReportJSONPreservesCustomEnvelope(t *testing.T) {
	body := `{"orgId":"org_test","window":{},"data":{"summary":{}}}`
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) })
	out, _, err := executeCommand("--json", "analytics", "report")
	assert.NoError(t, err)
	assert.JSONEq(t, body, out)
}

func TestAnalyticsReportQuiet(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"orgId":"org_test","window":{},"data":{"summary":{}}}`))
	})
	out, stderr, err := executeCommand("analytics", "report", "--quiet")
	assert.NoError(t, err)
	assert.Empty(t, out)
	assert.Empty(t, stderr)
}

func TestLegacyMagnoliaReportIsHiddenAndSendsOrgID(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "org_legacy", r.URL.Query().Get("orgId"))
		_, _ = w.Write([]byte(`{"orgId":"org_legacy","window":{},"data":{"summary":{}}}`))
	})
	_, _, err := executeCommand("magnolia", "report", "--org-id=")
	assert.EqualError(t, err, "--org-id is required")

	_, _, err = executeCommand("magnolia", "report", "--org-id", "org_legacy", "--quiet")
	assert.NoError(t, err)

	out, _, err := executeCommand("magnolia", "--help")
	assert.NoError(t, err)
	assert.NotContains(t, out, "report")
}
