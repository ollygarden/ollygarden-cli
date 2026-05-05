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

func setupInsightsSummaryServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_key")
	oldURL := apiURL
	t.Cleanup(func() {
		apiURL = oldURL
		jsonMode = false
		quiet = false
		if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
			f.Changed = false
		}
	})
	apiURL = srv.URL
	if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
		f.Changed = true
	}
	return srv
}

func summaryResponse(insightID, content, model, generatedAt string, cached bool) string {
	cachedStr := "false"
	if cached {
		cachedStr = "true"
	}
	return `{"data":{"insight_id":"` + insightID + `","content":"` + content + `","model":"` + model + `","generated_at":"` + generatedAt + `","cached":` + cachedStr + `},"meta":{"timestamp":"2026-04-20T12:00:00Z","trace_id":"abc123"}}`
}

func TestInsightsSummaryHuman(t *testing.T) {
	body := summaryResponse(
		"ins-1",
		"## Why it matters\\n\\nExposing PII can lead to compliance violations.",
		"gemini-2.0-flash",
		"2026-03-10T08:48:16Z",
		true,
	)
	setupInsightsSummaryServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/insights/ins-1/summary", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("insights", "summary", "ins-1")
	require.NoError(t, err)
	assert.Contains(t, out, "ins-1")
	assert.Contains(t, out, "gemini-2.0-flash")
	assert.Contains(t, out, "2026-03-10T08:48:16Z")
	assert.Contains(t, out, "yes")
	assert.Contains(t, out, "PII")
}

func TestInsightsSummaryNotCached(t *testing.T) {
	body := summaryResponse(
		"ins-2",
		"Generated on demand.",
		"gemini-2.0-flash",
		"2026-04-20T12:00:00Z",
		false,
	)
	setupInsightsSummaryServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("insights", "summary", "ins-2")
	require.NoError(t, err)
	assert.Contains(t, out, "no")
	assert.Contains(t, out, "Generated on demand.")
}

func TestInsightsSummaryJSON(t *testing.T) {
	body := summaryResponse("ins-1", "Summary content", "gemini-2.0-flash", "2026-03-10T08:48:16Z", true)
	setupInsightsSummaryServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("insights", "summary", "ins-1", "--json")
	require.NoError(t, err)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "abc123", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), `"gemini-2.0-flash"`)
}

func TestInsightsSummaryQuiet(t *testing.T) {
	body := summaryResponse("ins-1", "Summary content", "gemini-2.0-flash", "2026-03-10T08:48:16Z", true)
	setupInsightsSummaryServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("insights", "summary", "ins-1", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestInsightsSummaryMissingArg(t *testing.T) {
	setupInsightsSummaryServer(t, func(w http.ResponseWriter, r *http.Request) {})

	_, _, err := executeCommand("insights", "summary")
	require.Error(t, err)
}

func TestInsightsSummary404(t *testing.T) {
	body := `{"error":{"code":"INSIGHT_NOT_FOUND","message":"Insight not found"},"meta":{"trace_id":"t1"}}`
	setupInsightsSummaryServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("insights", "summary", "nonexistent")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 4, apiErr.ExitCode())
	assert.Contains(t, stderr, "Insight not found")
}

func TestInsightsSummary401(t *testing.T) {
	body := `{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t1"}}`
	setupInsightsSummaryServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("insights", "summary", "ins-1")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestInsightsSummary503(t *testing.T) {
	body := `{"error":{"code":"SERVICE_UNAVAILABLE","message":"Summary service is not available"},"meta":{"trace_id":"t1"}}`
	setupInsightsSummaryServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("insights", "summary", "ins-1")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Summary service is not available")
}

func TestInsightsSummary500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t2"}}`
	setupInsightsSummaryServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("insights", "summary", "ins-1")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestInsightsSummaryHelp(t *testing.T) {
	out, _, err := executeCommand("insights", "summary", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "summary")
	assert.Contains(t, out, "AI-generated summary")
}
