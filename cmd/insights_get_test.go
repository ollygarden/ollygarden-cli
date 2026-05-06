package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupInsightsGetServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_key")
	oldURL := apiURL
	var oldAPIURLChanged bool
	apiURL = srv.URL
	if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
		oldAPIURLChanged = f.Changed
		f.Changed = true
	}
	t.Cleanup(func() {
		apiURL = oldURL
		jsonMode = false
		quiet = false
		if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
			f.Changed = oldAPIURLChanged
		}
	})
	return srv
}

func insightGetResponse(id, status, serviceID, serviceName, serviceEnv string, withType bool) string {
	typeJSON := "null"
	if withType {
		typeJSON = `{"id":"type-1","name":"high_error_rate","display_name":"High Error Rate","description":"Error rate exceeds threshold","impact":"Critical","signal_type":"trace","remediation_instructions":"Check error logs"}`
	}
	return `{"data":{"id":"` + id + `","status":"` + status + `","service_id":"` + serviceID + `","service_name":"` + serviceName + `","service_environment":"` + serviceEnv + `","service_namespace":"backend","service_version":"v1.0.0","insight_type":` + typeJSON + `,"attributes":{"error_rate":0.15},"trace_id":"otel-trace-1","detected_ts":"2026-02-19T10:00:00Z","telemetry_ts":"2026-02-19T09:55:00Z","created_at":"2026-02-19T09:00:00Z","updated_at":"2026-02-19T12:00:00Z"},"meta":{"timestamp":"2026-02-19T12:00:00Z","trace_id":"abc123"},"links":{"self":"/api/v1/insights/` + id + `"}}`
}

func TestInsightsGetHuman(t *testing.T) {
	body := insightGetResponse("ins-1", "active", "svc-1", "payment-service", "production", true)
	setupInsightsGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/insights/ins-1", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("insights", "get", "ins-1")
	require.NoError(t, err)
	assert.Contains(t, out, "ins-1")
	assert.Contains(t, out, "active")
	assert.Contains(t, out, "High Error Rate")
	assert.Contains(t, out, "Critical")
	assert.Contains(t, out, "trace")
	assert.Contains(t, out, "payment-service (svc-1)")
	assert.Contains(t, out, "v1.0.0")
	assert.Contains(t, out, "production")
	assert.Contains(t, out, "backend")
	assert.Contains(t, out, "otel-trace-1")
	assert.Contains(t, out, "2026-02-19T09:55:00Z")
	assert.Contains(t, out, "2026-02-19T10:00:00Z")
	assert.Contains(t, out, "2026-02-19T09:00:00Z")
	assert.Contains(t, out, "2026-02-19T12:00:00Z")
	assert.Contains(t, out, "Error rate exceeds threshold")
	assert.Contains(t, out, "Check error logs")
	assert.Contains(t, out, "error_rate")
}

func TestInsightsGetJSON(t *testing.T) {
	body := insightGetResponse("ins-1", "active", "svc-1", "payment-service", "production", true)
	setupInsightsGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("insights", "get", "ins-1", "--json")
	require.NoError(t, err)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "abc123", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), `"payment-service"`)
}

func TestInsightsGetQuiet(t *testing.T) {
	body := insightGetResponse("ins-1", "active", "svc-1", "payment-service", "production", true)
	setupInsightsGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("insights", "get", "ins-1", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestInsightsGetNilInsightType(t *testing.T) {
	body := insightGetResponse("ins-1", "active", "svc-1", "payment-service", "production", false)
	setupInsightsGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("insights", "get", "ins-1")
	require.NoError(t, err)
	assert.Contains(t, out, "\u2014") // em dash for nil insight type fields
}

func TestInsightsGetEmptyOptionalFields(t *testing.T) {
	body := insightGetResponse("ins-1", "active", "svc-1", "payment-service", "", true)
	setupInsightsGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("insights", "get", "ins-1")
	require.NoError(t, err)
	assert.Contains(t, out, "\u2014") // em dash for empty environment
	assert.Contains(t, out, "payment-service")
}

func TestInsightsGetMissingArg(t *testing.T) {
	setupInsightsGetServer(t, func(w http.ResponseWriter, r *http.Request) {})

	_, _, err := executeCommand("insights", "get")
	require.Error(t, err)
}

func TestInsightsGet404(t *testing.T) {
	body := `{"error":{"code":"INSIGHT_NOT_FOUND","message":"Insight not found"},"meta":{"trace_id":"t1"}}`
	setupInsightsGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("insights", "get", "nonexistent")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 4, apiErr.ExitCode())
	assert.Contains(t, stderr, "Insight not found")
}

func TestInsightsGet401(t *testing.T) {
	body := `{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t1"}}`
	setupInsightsGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("insights", "get", "ins-1")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestInsightsGet500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t2"}}`
	setupInsightsGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("insights", "get", "ins-1")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestInsightsGetHelp(t *testing.T) {
	out, _, err := executeCommand("insights", "get", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "get")
	assert.Contains(t, out, "Show insight details")
}
