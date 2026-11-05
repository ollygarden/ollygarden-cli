package cmd

import (
	"net/http"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupInsightsCountsServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	oldGroupBy := insightsCountsGroupBy
	oldServiceID := insightsCountsServiceID
	oldStatus := insightsCountsStatus
	oldInsightType := insightsCountsInsightType
	oldSignalType := insightsCountsSignalType
	oldImpact := insightsCountsImpact
	oldDateFrom := insightsCountsDateFrom
	oldDateTo := insightsCountsDateTo
	setupAPIServer(t, handler)
	t.Cleanup(func() {
		insightsCountsGroupBy = oldGroupBy
		insightsCountsServiceID = oldServiceID
		insightsCountsStatus = oldStatus
		insightsCountsInsightType = oldInsightType
		insightsCountsSignalType = oldSignalType
		insightsCountsImpact = oldImpact
		insightsCountsDateFrom = oldDateFrom
		insightsCountsDateTo = oldDateTo
	})
}

func countsResponse(groups string, capped bool) string {
	return `{"data":[` + groups + `],"meta":{"trace_id":"tr1","capped":` + btoa(capped) + `}}`
}

func TestInsightsCountsGroupByValues(t *testing.T) {
	for _, tt := range []struct {
		groupBy string
		header  string
	}{
		{"insight_type", "TYPE"},
		{"service", "SERVICE"},
		{"impact", "IMPACT"},
	} {
		t.Run(tt.groupBy, func(t *testing.T) {
			body := countsResponse(`{"key":"a","count":3},{"key":"b","count":1}`, false)
			setupInsightsCountsServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/insights/summary", r.URL.Path)
				assert.Equal(t, tt.groupBy, r.URL.Query().Get("group_by"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(body))
			})

			out, _, err := executeCommand("insights", "counts", "--group-by", tt.groupBy)
			require.NoError(t, err)
			assert.Contains(t, out, tt.header)
			assert.Contains(t, out, "COUNT")
			assert.Contains(t, out, "a")
			assert.Contains(t, out, "3")
			assert.Contains(t, out, "b")
			assert.Contains(t, out, "1")
		})
	}
}

func TestInsightsCountsFiltersForwarded(t *testing.T) {
	body := countsResponse(`{"key":"trace","count":2}`, false)
	setupInsightsCountsServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "impact", q.Get("group_by"))
		assert.Equal(t, "svc-123", q.Get("service_id"))
		assert.Equal(t, "active,muted", q.Get("status"))
		assert.Equal(t, "duplicate-log-records", q.Get("insight_type"))
		assert.Equal(t, "resource", q.Get("signal_type"))
		assert.Equal(t, "Critical,Important", q.Get("impact"))
		assert.Equal(t, "2026-01-01T00:00:00Z", q.Get("date_from"))
		assert.Equal(t, "2026-02-01T00:00:00Z", q.Get("date_to"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, _, err := executeCommand("insights", "counts",
		"--group-by", "impact",
		"--service-id", "svc-123",
		"--status", "active,muted",
		"--insight-type", "duplicate-log-records",
		"--signal-type", "resource",
		"--impact", "Critical,Important",
		"--date-from", "2026-01-01T00:00:00Z",
		"--date-to", "2026-02-01T00:00:00Z",
	)
	require.NoError(t, err)
}

func TestInsightsCountsJSONPassthrough(t *testing.T) {
	body := countsResponse(`{"key":"trace","count":5}`, true)
	setupInsightsCountsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("insights", "counts", "--group-by", "impact", "--json")
	require.NoError(t, err)
	assert.JSONEq(t, body, out)
}

func TestInsightsCountsCappedSurfacedInHumanMode(t *testing.T) {
	body := countsResponse(`{"key":"trace","count":100}`, true)
	setupInsightsCountsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, stderr, err := executeCommand("insights", "counts", "--group-by", "impact")
	require.NoError(t, err)
	assert.Contains(t, out, "trace")
	assert.Contains(t, stderr, "truncated")
}

func TestInsightsCountsNotCappedNoWarning(t *testing.T) {
	body := countsResponse(`{"key":"trace","count":100}`, false)
	setupInsightsCountsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("insights", "counts", "--group-by", "impact")
	require.NoError(t, err)
	assert.Empty(t, stderr)
}

func TestInsightsCountsQuiet(t *testing.T) {
	body := countsResponse(`{"key":"trace","count":5}`, false)
	setupInsightsCountsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("insights", "counts", "--group-by", "impact", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestInsightsCountsInvalidGroupBy(t *testing.T) {
	for _, tt := range []string{"", "bogus", "insight_type,service"} {
		t.Run(tt, func(t *testing.T) {
			setupInsightsCountsServer(t, func(w http.ResponseWriter, r *http.Request) { t.Fatal("unexpected request") })
			args := []string{"insights", "counts"}
			if tt != "" {
				args = append(args, "--group-by", tt)
			}
			_, _, err := executeCommand(args...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "--group-by")
		})
	}
}

func TestInsightsCountsInvalidFiltersBeforeRequest(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		message string
	}{
		{"status", []string{"--status", "active,bad"}, "--status"},
		{"signal type", []string{"--signal-type", "span"}, "--signal-type"},
		{"impact", []string{"--impact", "critical"}, "--impact"},
		{"date from", []string{"--date-from", "yesterday"}, "--date-from"},
		{"date order", []string{"--date-from", "2026-02-02T00:00:00Z", "--date-to", "2026-02-01T00:00:00Z"}, "must not be after"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupInsightsCountsServer(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })
			args := append([]string{"insights", "counts", "--group-by", "impact"}, tt.args...)
			_, _, err := executeCommand(args...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.message)
		})
	}
}

func TestInsightsCounts500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t2"}}`
	setupInsightsCountsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("insights", "counts", "--group-by", "impact")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestInsightsCountsHelp(t *testing.T) {
	out, _, err := executeCommand("insights", "counts", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "Count insights")
	assert.Contains(t, out, "--group-by")
	assert.Contains(t, out, "--service-id")
	assert.Contains(t, out, "--status")
	assert.Contains(t, out, "--insight-type")
	assert.Contains(t, out, "--signal-type")
	assert.Contains(t, out, "--impact")
	assert.Contains(t, out, "--date-from")
	assert.Contains(t, out, "--date-to")
	assert.NotContains(t, out, "--limit")
	assert.NotContains(t, out, "--offset")
}
