package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnalyticsLogDuplicationHumanAndV3PathWithoutOrgQuery(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v3/magnolia/log-duplication", r.URL.Path)
		assert.NotContains(t, r.URL.Query(), "orgId")
		_, _ = w.Write([]byte(`{"orgId":"org_test","runDate":"2026-09-07","generatedAt":"2026-09-08T01:00:00Z","data":{"total":1000,"unique_log_messages":250,"dedup_potential_pct":75}}`))
	})
	out, _, err := executeCommand("analytics", "log-duplication")
	assert.NoError(t, err)
	assert.Contains(t, out, "org_test")
	assert.Contains(t, out, "1000")
	assert.Contains(t, out, "75.0%")
}

func TestAnalyticsLogDuplicationJSONPreservesEnvelope(t *testing.T) {
	body := `{"orgId":"org_test","runDate":"2026-09-07","generatedAt":"2026-09-08T01:00:00Z","data":{"total":0,"unique_log_messages":0,"dedup_potential_pct":0}}`
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) })
	out, _, err := executeCommand("--json", "analytics", "log-duplication")
	assert.NoError(t, err)
	assert.JSONEq(t, body, out)
}

func TestAnalyticsLogDuplicationQuiet(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"orgId":"org_test","data":{}}`))
	})
	out, stderr, err := executeCommand("analytics", "log-duplication", "--quiet")
	assert.NoError(t, err)
	assert.Empty(t, out)
	assert.Empty(t, stderr)
}
