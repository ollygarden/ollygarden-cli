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

func setupServicesInsightsServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_key")
	oldURL := apiURL
	oldStatus := servicesInsightsStatus
	oldLimit := servicesInsightsLimit
	oldOffset := servicesInsightsOffset
	apiURL = srv.URL
	t.Cleanup(func() {
		apiURL = oldURL
		jsonMode = false
		quiet = false
		servicesInsightsStatus = oldStatus
		servicesInsightsLimit = oldLimit
		servicesInsightsOffset = oldOffset
	})
	return srv
}

func insightJSON(id, status, displayName, impact, signalType, detectedTS string) string {
	return `{"id":"` + id + `","status":"` + status + `","insight_type":{"display_name":"` + displayName + `","impact":"` + impact + `","signal_type":"` + signalType + `"},"detected_ts":"` + detectedTS + `"}`
}

func insightsListResponse(insights string, total int, hasMore bool) string {
	return `{"data":[` + insights + `],"meta":{"timestamp":"2026-02-19T12:00:00Z","trace_id":"tr1","total":` +
		itoa(total) + `,"has_more":` + btoa(hasMore) + `}}`
}

func TestServicesInsightsHuman(t *testing.T) {
	i1 := insightJSON("ins-1", "active", "High Error Rate", "high", "error_rate", "2026-02-19T10:00:00Z")
	i2 := insightJSON("ins-2", "active", "Slow Response", "medium", "latency", "2026-02-19T09:00:00Z")
	body := insightsListResponse(i1+","+i2, 2, false)

	setupServicesInsightsServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/services/svc-1/insights", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "insights", "svc-1")
	require.NoError(t, err)
	assert.Contains(t, out, "ins-1")
	assert.Contains(t, out, "ins-2")
	assert.Contains(t, out, "High Error Rate")
	assert.Contains(t, out, "Slow Response")
	assert.Contains(t, out, "high")
	assert.Contains(t, out, "medium")
	assert.Contains(t, out, "error_rate")
	assert.Contains(t, out, "latency")
}

func TestServicesInsightsJSON(t *testing.T) {
	i1 := insightJSON("ins-1", "active", "High Error Rate", "high", "error_rate", "2026-02-19T10:00:00Z")
	body := insightsListResponse(i1, 1, false)

	setupServicesInsightsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "insights", "svc-1", "--json")
	require.NoError(t, err)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "tr1", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), "High Error Rate")
}

func TestServicesInsightsQuiet(t *testing.T) {
	i1 := insightJSON("ins-1", "active", "High Error Rate", "high", "error_rate", "2026-02-19T10:00:00Z")
	body := insightsListResponse(i1, 1, false)

	setupServicesInsightsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "insights", "svc-1", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestServicesInsightsStatusFlag(t *testing.T) {
	i1 := insightJSON("ins-1", "active", "High Error Rate", "high", "error_rate", "2026-02-19T10:00:00Z")
	body := insightsListResponse(i1, 1, false)

	setupServicesInsightsServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "active,muted", r.URL.Query().Get("status"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, _, err := executeCommand("services", "insights", "svc-1", "--status", "active,muted")
	require.NoError(t, err)
}

func TestServicesInsightsPagination(t *testing.T) {
	i1 := insightJSON("ins-1", "active", "High Error Rate", "high", "error_rate", "2026-02-19T10:00:00Z")
	body := insightsListResponse(i1, 100, true)

	setupServicesInsightsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("services", "insights", "svc-1")
	require.NoError(t, err)
	assert.Contains(t, stderr, "more results")
	assert.Contains(t, stderr, "--offset")
}

func TestServicesInsightsFlags(t *testing.T) {
	i1 := insightJSON("ins-1", "active", "High Error Rate", "high", "error_rate", "2026-02-19T10:00:00Z")
	body := insightsListResponse(i1, 1, false)

	setupServicesInsightsServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "25", r.URL.Query().Get("limit"))
		assert.Equal(t, "10", r.URL.Query().Get("offset"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, _, err := executeCommand("services", "insights", "svc-1", "--limit", "25", "--offset", "10")
	require.NoError(t, err)
}

func TestServicesInsightsInvalidLimit(t *testing.T) {
	setupServicesInsightsServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with invalid limit")
	})

	_, _, err := executeCommand("services", "insights", "svc-1", "--limit", "0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--limit")

	_, _, err = executeCommand("services", "insights", "svc-1", "--limit", "101")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--limit")
}

func TestServicesInsightsInvalidOffset(t *testing.T) {
	setupServicesInsightsServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with invalid offset")
	})

	_, _, err := executeCommand("services", "insights", "svc-1", "--offset", "-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--offset")
}

func TestServicesInsightsMissingArg(t *testing.T) {
	setupServicesInsightsServer(t, func(w http.ResponseWriter, r *http.Request) {})

	_, _, err := executeCommand("services", "insights")
	require.Error(t, err)
}

func TestServicesInsights404(t *testing.T) {
	body := `{"error":{"code":"SERVICE_NOT_FOUND","message":"Service not found"},"meta":{"trace_id":"t1"}}`
	setupServicesInsightsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("services", "insights", "nonexistent")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 4, apiErr.ExitCode())
	assert.Contains(t, stderr, "Service not found")
}

func TestServicesInsights401(t *testing.T) {
	body := `{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t1"}}`
	setupServicesInsightsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("services", "insights", "svc-1")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestServicesInsights500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t2"}}`
	setupServicesInsightsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("services", "insights", "svc-1")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestServicesInsightsHelp(t *testing.T) {
	out, _, err := executeCommand("services", "insights", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "insights")
	assert.Contains(t, out, "List insights for a service")
	assert.Contains(t, out, "--status")
	assert.Contains(t, out, "--limit")
	assert.Contains(t, out, "--offset")
}
