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

func setupAnalyticsServicesServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_key")
	oldURL := apiURL
	oldLimit := analyticsServicesLimit
	apiURL = srv.URL
	t.Cleanup(func() {
		apiURL = oldURL
		jsonMode = false
		quiet = false
		analyticsServicesLimit = oldLimit
	})
	return srv
}

func analyticsServicesResponse(services string) string {
	return `{"data":{"period_start":"2024-01-01T00:00:00Z","period_end":"2024-01-08T00:00:00Z","services":[` + services + `]},"meta":{"timestamp":"2026-02-19T12:00:00Z","trace_id":"tr1"}}`
}

func analyticsServiceJSON(name, env string, totalBytes int64, totalPct float64, version string) string {
	versionJSON := "null"
	if version != "" {
		versionJSON = `{"id":"ver-1","version":"` + version + `"}`
	}
	return `{"name":"` + name + `","namespace":"production","environment":"` + env +
		`","total_bytes":` + itoa(int(totalBytes)) + `,"total_percent":` +
		json.Number(func() string { b, _ := json.Marshal(totalPct); return string(b) }()).String() +
		`,"metrics_bytes":500000000,"metrics_percent":40.5,"metrics_count":1000000` +
		`,"traces_bytes":600000000,"traces_percent":48.6,"traces_count":50000` +
		`,"logs_bytes":134567890,"logs_percent":10.9,"logs_count":200000` +
		`,"latest_version":` + versionJSON + `}`
}

func TestAnalyticsServicesHuman(t *testing.T) {
	s1 := analyticsServiceJSON("api-gateway", "prod", 1234567890, 45.2, "2.0.0")
	s2 := analyticsServiceJSON("auth-service", "staging", 567000000, 22.1, "1.3.0")
	body := analyticsServicesResponse(s1 + "," + s2)

	setupAnalyticsServicesServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/analytics/services", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, stderr, err := executeCommand("analytics", "services")
	require.NoError(t, err)
	assert.Contains(t, out, "api-gateway")
	assert.Contains(t, out, "auth-service")
	assert.Contains(t, out, "prod")
	assert.Contains(t, out, "staging")
	assert.Contains(t, out, "1.2 GB")
	assert.Contains(t, out, "567.0 MB")
	assert.Contains(t, out, "45.2%")
	assert.Contains(t, out, "22.1%")
	assert.Contains(t, out, "2.0.0")
	assert.Contains(t, out, "1.3.0")
	assert.Contains(t, stderr, "Period: 2024-01-01T00:00:00Z to 2024-01-08T00:00:00Z")
}

func TestAnalyticsServicesJSON(t *testing.T) {
	s1 := analyticsServiceJSON("api-gateway", "prod", 1234567890, 45.2, "2.0.0")
	body := analyticsServicesResponse(s1)

	setupAnalyticsServicesServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("analytics", "services", "--json")
	require.NoError(t, err)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "tr1", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), "api-gateway")
}

func TestAnalyticsServicesQuiet(t *testing.T) {
	s1 := analyticsServiceJSON("api-gateway", "prod", 1234567890, 45.2, "2.0.0")
	body := analyticsServicesResponse(s1)

	setupAnalyticsServicesServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("analytics", "services", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestAnalyticsServicesFlags(t *testing.T) {
	s1 := analyticsServiceJSON("api-gateway", "prod", 1234567890, 45.2, "2.0.0")
	body := analyticsServicesResponse(s1)

	setupAnalyticsServicesServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "10", q.Get("limit"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, _, err := executeCommand("analytics", "services", "--limit", "10")
	require.NoError(t, err)
}

func TestAnalyticsServicesInvalidLimit(t *testing.T) {
	setupAnalyticsServicesServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with invalid limit")
	})

	_, _, err := executeCommand("analytics", "services", "--limit", "0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--limit")

	_, _, err = executeCommand("analytics", "services", "--limit", "101")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--limit")
}

func TestAnalyticsServicesNilVersion(t *testing.T) {
	s1 := analyticsServiceJSON("api-gateway", "prod", 1234567890, 45.2, "")
	body := analyticsServicesResponse(s1)

	setupAnalyticsServicesServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("analytics", "services")
	require.NoError(t, err)
	assert.Contains(t, out, "\u2014") // em dash for nil latest_version
}

func TestAnalyticsServices401(t *testing.T) {
	body := `{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t1"}}`
	setupAnalyticsServicesServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("analytics", "services")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestAnalyticsServices500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t2"}}`
	setupAnalyticsServicesServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("analytics", "services")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestAnalyticsServicesHelp(t *testing.T) {
	out, _, err := executeCommand("analytics", "services", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "List service analytics")
	assert.Contains(t, out, "--limit")
}
