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

func setupInsightsListServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_key")
	oldURL := apiURL
	oldLimit := insightsListLimit
	oldOffset := insightsListOffset
	oldServiceID := insightsListServiceID
	oldStatus := insightsListStatus
	oldSignalType := insightsListSignalType
	oldImpact := insightsListImpact
	oldDateFrom := insightsListDateFrom
	oldDateTo := insightsListDateTo
	oldSort := insightsListSort
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
		insightsListLimit = oldLimit
		insightsListOffset = oldOffset
		insightsListServiceID = oldServiceID
		insightsListStatus = oldStatus
		insightsListSignalType = oldSignalType
		insightsListImpact = oldImpact
		insightsListDateFrom = oldDateFrom
		insightsListDateTo = oldDateTo
		insightsListSort = oldSort
		if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
			f.Changed = oldAPIURLChanged
		}
	})
	return srv
}

func insightsListItemJSON(id, status, serviceName, displayName, impact, signalType, detectedTS string) string {
	return `{"id":"` + id + `","status":"` + status + `","service_name":"` + serviceName +
		`","insight_type":{"display_name":"` + displayName + `","impact":"` + impact +
		`","signal_type":"` + signalType + `"},"detected_ts":"` + detectedTS + `"}`
}

func orgInsightsListResponse(insights string, total int, hasMore bool) string {
	return `{"data":[` + insights + `],"meta":{"timestamp":"2026-02-19T12:00:00Z","trace_id":"tr1","total":` +
		itoa(total) + `,"has_more":` + btoa(hasMore) + `}}`
}

func TestInsightsListHuman(t *testing.T) {
	i1 := insightsListItemJSON("ins-1", "active", "my-service", "High Error Rate", "Critical", "trace", "2026-02-19T10:00:00Z")
	i2 := insightsListItemJSON("ins-2", "active", "other-svc", "Slow Response", "Normal", "metric", "2026-02-19T09:00:00Z")
	body := orgInsightsListResponse(i1+","+i2, 2, false)

	setupInsightsListServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/insights", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "-detected_ts", r.URL.Query().Get("sort"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("insights", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "ins-1")
	assert.Contains(t, out, "ins-2")
	assert.Contains(t, out, "High Error Rate")
	assert.Contains(t, out, "Slow Response")
	assert.Contains(t, out, "Critical")
	assert.Contains(t, out, "Normal")
	assert.Contains(t, out, "trace")
	assert.Contains(t, out, "metric")
	assert.Contains(t, out, "my-service")
	assert.Contains(t, out, "other-svc")
}

func TestInsightsListJSON(t *testing.T) {
	i1 := insightsListItemJSON("ins-1", "active", "my-service", "High Error Rate", "Critical", "trace", "2026-02-19T10:00:00Z")
	body := orgInsightsListResponse(i1, 1, false)

	setupInsightsListServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("insights", "list", "--json")
	require.NoError(t, err)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "tr1", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), "High Error Rate")
}

func TestInsightsListQuiet(t *testing.T) {
	i1 := insightsListItemJSON("ins-1", "active", "my-service", "High Error Rate", "Critical", "trace", "2026-02-19T10:00:00Z")
	body := orgInsightsListResponse(i1, 1, false)

	setupInsightsListServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("insights", "list", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestInsightsListFilterFlags(t *testing.T) {
	i1 := insightsListItemJSON("ins-1", "active", "my-service", "High Error Rate", "Critical", "trace", "2026-02-19T10:00:00Z")
	body := orgInsightsListResponse(i1, 1, false)

	setupInsightsListServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "svc-123", q.Get("service_id"))
		assert.Equal(t, "active,muted", q.Get("status"))
		assert.Equal(t, "trace", q.Get("signal_type"))
		assert.Equal(t, "Critical,Important", q.Get("impact"))
		assert.Equal(t, "2026-01-01T00:00:00Z", q.Get("date_from"))
		assert.Equal(t, "2026-02-01T00:00:00Z", q.Get("date_to"))
		assert.Equal(t, "+impact", q.Get("sort"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, _, err := executeCommand("insights", "list",
		"--service-id", "svc-123",
		"--status", "active,muted",
		"--signal-type", "trace",
		"--impact", "Critical,Important",
		"--date-from", "2026-01-01T00:00:00Z",
		"--date-to", "2026-02-01T00:00:00Z",
		"--sort", "+impact",
	)
	require.NoError(t, err)
}

func TestInsightsListPagination(t *testing.T) {
	i1 := insightsListItemJSON("ins-1", "active", "my-service", "High Error Rate", "Critical", "trace", "2026-02-19T10:00:00Z")
	body := orgInsightsListResponse(i1, 100, true)

	setupInsightsListServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("insights", "list")
	require.NoError(t, err)
	assert.Contains(t, stderr, "more results")
	assert.Contains(t, stderr, "--offset")
}

func TestInsightsListInvalidLimit(t *testing.T) {
	setupInsightsListServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with invalid limit")
	})

	_, _, err := executeCommand("insights", "list", "--limit", "0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--limit")

	_, _, err = executeCommand("insights", "list", "--limit", "101")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--limit")
}

func TestInsightsListInvalidOffset(t *testing.T) {
	setupInsightsListServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with invalid offset")
	})

	_, _, err := executeCommand("insights", "list", "--offset", "-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--offset")
}

func TestInsightsList401(t *testing.T) {
	body := `{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t1"}}`
	setupInsightsListServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("insights", "list")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestInsightsList500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t2"}}`
	setupInsightsListServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("insights", "list")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestInsightsListHelp(t *testing.T) {
	out, _, err := executeCommand("insights", "list", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "List insights")
	assert.Contains(t, out, "--limit")
	assert.Contains(t, out, "--offset")
	assert.Contains(t, out, "--service-id")
	assert.Contains(t, out, "--status")
	assert.Contains(t, out, "--signal-type")
	assert.Contains(t, out, "--impact")
	assert.Contains(t, out, "--date-from")
	assert.Contains(t, out, "--date-to")
	assert.Contains(t, out, "--sort")
}
