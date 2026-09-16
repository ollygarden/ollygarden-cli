package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnalyticsLogPatternsHumanAndV3PathWithoutOrgQuery(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v3/magnolia/log-patterns", r.URL.Path)
		assert.NotContains(t, r.URL.Query(), "orgId")
		_, _ = w.Write([]byte(`{"orgId":"org_test","runDate":"2026-09-07","generatedAt":"2026-09-08T01:00:00Z","data":[{"ServiceName":"checkout","SeverityText":"INFO","service_namespace":"shop","environment":"prod","body_preview":"payment processed for order N","cnt":9001}]}`))
	})
	out, _, err := executeCommand("analytics", "log-patterns")
	assert.NoError(t, err)
	assert.Contains(t, out, "org_test")
	assert.Contains(t, out, "checkout")
	assert.Contains(t, out, "payment processed")
}

func TestAnalyticsLogPatternsJSONPreservesEnvelope(t *testing.T) {
	body := `{"orgId":"org_test","runDate":"2026-09-07","generatedAt":"2026-09-08T01:00:00Z","data":[]}`
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) })
	out, _, err := executeCommand("--json", "analytics", "log-patterns")
	assert.NoError(t, err)
	assert.JSONEq(t, body, out)
}

func TestAnalyticsLogPatternsQuiet(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"orgId":"org_test","data":[]}`))
	})
	out, stderr, err := executeCommand("analytics", "log-patterns", "--quiet")
	assert.NoError(t, err)
	assert.Empty(t, out)
	assert.Empty(t, stderr)
}
