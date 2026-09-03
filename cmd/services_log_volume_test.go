package cmd

import (
	"net/http"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupServicesLogVolumeServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	oldPeriod := servicesLogVolumePeriod
	setupAPIServer(t, handler)
	t.Cleanup(func() { servicesLogVolumePeriod = oldPeriod })
}

func TestServicesLogVolume(t *testing.T) {
	setupServicesLogVolumeServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/services/service-123/analytics/log-volume", r.URL.Path)
		assert.Equal(t, "7d", r.URL.Query().Get("period"))
		_, _ = w.Write([]byte(logVolumeResponse))
	})
	out, _, err := executeCommand("services", "log-volume", "service-123", "--period", "7d")
	require.NoError(t, err)
	assert.Contains(t, out, "ERROR")
}

func TestServicesLogVolumeInvalidPeriodBeforeIO(t *testing.T) {
	setupServicesLogVolumeServer(t, func(http.ResponseWriter, *http.Request) { t.Fatal("unexpected request") })
	_, _, err := executeCommand("services", "log-volume", "service-123", "--period", "yesterday")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--period must be one of")
}

func TestServicesLogVolumeNotFound(t *testing.T) {
	setupServicesLogVolumeServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"SERVICE_NOT_FOUND","message":"Service not found"},"meta":{"trace_id":"trace-404"}}`))
	})
	_, stderr, err := executeCommand("services", "log-volume", "missing")
	require.Error(t, err)
	var apiErr *client.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 4, apiErr.ExitCode())
	assert.Contains(t, stderr, "Service not found")
}

func TestServicesLogVolumeRequiresID(t *testing.T) {
	_, _, err := executeCommand("services", "log-volume")
	require.Error(t, err)
}
