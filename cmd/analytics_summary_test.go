package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnalyticsSummaryHumanAndV3PathWithoutOrgQuery(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v3/magnolia/summary", r.URL.Path)
		assert.NotContains(t, r.URL.Query(), "orgId")
		_, _ = w.Write([]byte(`{"orgId":"org_test","runDate":"2026-09-07","generatedAt":"2026-09-08T01:00:00Z","data":{"totals":{"spans":10,"log_records":20,"metric_datapoints":30,"metric_datapoints_by_type":{"gauge":5,"sum":15,"histogram":10}},"service_counts_by_signal":[{"signal":"traces","services":3}]}}`))
	})
	out, _, err := executeCommand("analytics", "summary")
	assert.NoError(t, err)
	assert.Contains(t, out, "org_test")
	assert.Contains(t, out, "2026-09-07")
	assert.Contains(t, out, "traces")
}

func TestAnalyticsSummaryJSONPreservesEnvelope(t *testing.T) {
	body := `{"orgId":"org_test","runDate":"2026-09-07","generatedAt":"2026-09-08T01:00:00Z","data":{"totals":{},"service_counts_by_signal":[]}}`
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) })
	out, _, err := executeCommand("--json", "analytics", "summary")
	assert.NoError(t, err)
	assert.JSONEq(t, body, out)
}

func TestAnalyticsSummaryQuiet(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"orgId":"org_test","data":{}}`))
	})
	out, stderr, err := executeCommand("analytics", "summary", "--quiet")
	assert.NoError(t, err)
	assert.Empty(t, out)
	assert.Empty(t, stderr)
}
