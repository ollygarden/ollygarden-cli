package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnalyticsTracesHumanAndV3PathWithoutOrgQuery(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v3/magnolia/traces", r.URL.Path)
		assert.NotContains(t, r.URL.Query(), "orgId")
		_, _ = w.Write([]byte(`{"orgId":"org_test","runDate":"2026-09-07","generatedAt":"2026-09-08T01:00:00Z","data":[{"trace_id":"abc123","root_span_name":"GET /","service_name":"gateway","span_count":512}]}`))
	})
	out, _, err := executeCommand("analytics", "traces")
	assert.NoError(t, err)
	assert.Contains(t, out, "org_test")
	assert.Contains(t, out, "abc123")
	assert.Contains(t, out, "512")
}

func TestAnalyticsTracesJSONPreservesEnvelope(t *testing.T) {
	body := `{"orgId":"org_test","runDate":"2026-09-07","generatedAt":"2026-09-08T01:00:00Z","data":[]}`
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) })
	out, _, err := executeCommand("--json", "analytics", "traces")
	assert.NoError(t, err)
	assert.JSONEq(t, body, out)
}

func TestAnalyticsTracesQuiet(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"orgId":"org_test","data":[]}`))
	})
	out, stderr, err := executeCommand("analytics", "traces", "--quiet")
	assert.NoError(t, err)
	assert.Empty(t, out)
	assert.Empty(t, stderr)
}
