package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnalyticsMetricsHistogramHumanAndV3PathWithoutOrgQuery(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v3/magnolia/metrics-histogram", r.URL.Path)
		assert.NotContains(t, r.URL.Query(), "orgId")
		_, _ = w.Write([]byte(`{"orgId":"org_test","runDate":"2026-09-07","generatedAt":"2026-09-08T01:00:00Z","data":[{"metric_name":"http.duration","service_name":"checkout","service_namespace":"shop","environment":"prod","datapoints":450,"value_diversity_pct":99.9,"metric_type":"histogram"}]}`))
	})
	out, _, err := executeCommand("analytics", "metrics-histogram")
	assert.NoError(t, err)
	assert.Contains(t, out, "org_test")
	assert.Contains(t, out, "http.duration")
	assert.Contains(t, out, "99.9%")
}

func TestAnalyticsMetricsHistogramJSONPreservesEnvelope(t *testing.T) {
	body := `{"orgId":"org_test","runDate":"2026-09-07","generatedAt":"2026-09-08T01:00:00Z","data":[]}`
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) })
	out, _, err := executeCommand("--json", "analytics", "metrics-histogram")
	assert.NoError(t, err)
	assert.JSONEq(t, body, out)
}

func TestAnalyticsMetricsHistogramQuiet(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"orgId":"org_test","data":[]}`))
	})
	out, stderr, err := executeCommand("analytics", "metrics-histogram", "--quiet")
	assert.NoError(t, err)
	assert.Empty(t, out)
	assert.Empty(t, stderr)
}
