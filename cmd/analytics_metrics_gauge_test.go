package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnalyticsMetricsGaugeHumanAndV3PathWithoutOrgQuery(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v3/magnolia/metrics-gauge", r.URL.Path)
		assert.NotContains(t, r.URL.Query(), "orgId")
		_, _ = w.Write([]byte(`{"orgId":"org_test","runDate":"2026-09-07","generatedAt":"2026-09-08T01:00:00Z","data":[{"metric_name":"process.memory","service_name":"checkout","service_namespace":"shop","environment":"prod","datapoints":120,"value_diversity_pct":33.333,"metric_type":"gauge"}]}`))
	})
	out, _, err := executeCommand("analytics", "metrics-gauge")
	assert.NoError(t, err)
	assert.Contains(t, out, "org_test")
	assert.Contains(t, out, "process.memory")
	assert.Contains(t, out, "33.3%")
}

func TestAnalyticsMetricsGaugeJSONPreservesEnvelope(t *testing.T) {
	body := `{"orgId":"org_test","runDate":"2026-09-07","generatedAt":"2026-09-08T01:00:00Z","data":[]}`
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) })
	out, _, err := executeCommand("--json", "analytics", "metrics-gauge")
	assert.NoError(t, err)
	assert.JSONEq(t, body, out)
}

func TestAnalyticsMetricsGaugeQuiet(t *testing.T) {
	setupAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"orgId":"org_test","data":[]}`))
	})
	out, stderr, err := executeCommand("analytics", "metrics-gauge", "--quiet")
	assert.NoError(t, err)
	assert.Empty(t, out)
	assert.Empty(t, stderr)
}
