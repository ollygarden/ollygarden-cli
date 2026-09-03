package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const logVolumeResponse = `{"data":{"period":"24h","total_count":1250,"severities":[{"severity_text":"INFO","record_count":1000,"percent":80},{"severity_text":"ERROR","record_count":250,"percent":20}]},"meta":{"timestamp":"2026-09-03T12:00:00Z","trace_id":"trace-1"},"links":{"self":"/api/v1/analytics/log-volume"}}`

func setupAnalyticsLogVolumeServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	oldPeriod := analyticsLogVolumePeriod
	setupAPIServer(t, handler)
	t.Cleanup(func() { analyticsLogVolumePeriod = oldPeriod })
}

func TestAnalyticsLogVolumeHuman(t *testing.T) {
	setupAnalyticsLogVolumeServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/analytics/log-volume", r.URL.Path)
		assert.Equal(t, "24h", r.URL.Query().Get("period"))
		_, _ = w.Write([]byte(logVolumeResponse))
	})

	out, stderr, err := executeCommand("analytics", "log-volume")
	require.NoError(t, err)
	assert.Contains(t, out, "SEVERITY")
	assert.Contains(t, out, "INFO")
	assert.Contains(t, out, "1000")
	assert.Contains(t, out, "80.00%")
	assert.Contains(t, stderr, "Period: 24h")
	assert.Contains(t, stderr, "Total records: 1250")
}

func TestAnalyticsLogVolumeJSONPreservesEnvelope(t *testing.T) {
	setupAnalyticsLogVolumeServer(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(logVolumeResponse)) })
	out, stderr, err := executeCommand("analytics", "log-volume", "--json")
	require.NoError(t, err)
	assert.Empty(t, stderr)

	var envelope map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Contains(t, envelope, "data")
	assert.Contains(t, envelope, "meta")
	assert.Contains(t, envelope, "links")
}

func TestAnalyticsLogVolumeQuiet(t *testing.T) {
	setupAnalyticsLogVolumeServer(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(logVolumeResponse)) })
	out, stderr, err := executeCommand("analytics", "log-volume", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
	assert.Empty(t, stderr)
}

func TestAnalyticsLogVolumeValidPeriods(t *testing.T) {
	for _, period := range []string{"1h", "6h", "12h", "24h", "7d", "30d"} {
		t.Run(period, func(t *testing.T) {
			setupAnalyticsLogVolumeServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, period, r.URL.Query().Get("period"))
				_, _ = w.Write([]byte(logVolumeResponse))
			})
			_, _, err := executeCommand("analytics", "log-volume", "--period", period, "--quiet")
			require.NoError(t, err)
		})
	}
}

func TestAnalyticsLogVolumeInvalidPeriodBeforeIO(t *testing.T) {
	setupAnalyticsLogVolumeServer(t, func(http.ResponseWriter, *http.Request) { t.Fatal("unexpected request") })
	_, _, err := executeCommand("analytics", "log-volume", "--period", "2h")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--period must be one of")
}

func TestAnalyticsLogVolumePrometheusErrors(t *testing.T) {
	for _, status := range []int{http.StatusBadGateway, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			setupAnalyticsLogVolumeServer(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"error":{"code":"UPSTREAM_ERROR","message":"Log volume unavailable"},"meta":{"trace_id":"trace-error"}}`))
			})
			_, stderr, err := executeCommand("analytics", "log-volume")
			require.Error(t, err)
			var apiErr *client.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, 6, apiErr.ExitCode())
			assert.Contains(t, stderr, "Log volume unavailable")
		})
	}
}
