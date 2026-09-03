package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoseExecutionsFindingsHumanAndFilters(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/rose/codebase/findings", r.URL.Path)
		q := r.URL.Query()
		assert.Equal(t, roseTestExecutionID, q.Get("executionId"))
		assert.Equal(t, "resolved", q.Get("status"))
		assert.Equal(t, "all", q.Get("dismissed"))
		assert.Equal(t, "2", q.Get("page"))
		assert.Equal(t, "10", q.Get("page_size"))
		_, _ = w.Write([]byte(roseFindingsListResponse(false)))
	})
	out, _, err := executeCommand("rose", "executions", "findings", roseTestExecutionID, "--status", "resolved", "--dismissed", "all", "--page", "2", "--limit", "10")
	require.NoError(t, err)
	assert.Contains(t, out, "otel-6576685e6e1f")
	assert.Contains(t, out, "acme/checkout")
}

func TestRoseExecutionsFindingsDefaultsAndPageHint(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "all", r.URL.Query().Get("status"))
		assert.Equal(t, "false", r.URL.Query().Get("dismissed"))
		_, _ = w.Write([]byte(roseFindingsListResponse(true)))
	})
	_, stderr, err := executeCommand("rose", "executions", "findings", roseTestExecutionID, "--limit", "2")
	require.NoError(t, err)
	assert.Contains(t, stderr, "# 27 more results. Use --page 2")
}

func TestRoseExecutionsFindingsJSON(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(roseFindingsListResponse(false))) })
	out, _, err := executeCommand("rose", "executions", "findings", roseTestExecutionID, "--json")
	require.NoError(t, err)
	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "trace-list", envelope.Meta.TraceID)
}

func TestRoseExecutionsFindingsQuiet(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(roseFindingsListResponse(false))) })
	out, _, err := executeCommand("rose", "executions", "findings", roseTestExecutionID, "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestRoseExecutionsFindingsEmpty(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"data":[],"pagination":{"limit":50,"offset":0,"total":0,"hasMore":false}},"meta":{}}`))
	})
	out, _, err := executeCommand("rose", "executions", "findings", roseTestExecutionID)
	require.NoError(t, err)
	assert.Contains(t, out, "FINDING ID")
}

func TestRoseExecutionsFindingsValidation(t *testing.T) {
	for i, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"bad"}, "execution-id must be a UUID"},
		{[]string{roseTestExecutionID, "--status", "pending"}, "--status must be one of"},
		{[]string{roseTestExecutionID, "--dismissed", "yes"}, "--dismissed must be one of"},
		{[]string{roseTestExecutionID, "--page", "0"}, "--page must be >= 1"},
		{[]string{roseTestExecutionID, "--limit", "101"}, "--limit must be between 1 and 100"},
	} {
		t.Run(string(rune('a'+i)), func(t *testing.T) {
			setupRoseServer(t, func(http.ResponseWriter, *http.Request) { t.Fatal("unexpected request") })
			args := append([]string{"rose", "executions", "findings"}, tc.args...)
			_, stderr, err := executeCommand(args...)
			require.Error(t, err)
			assert.Contains(t, stderr, tc.want)
		})
	}
}

func TestRoseExecutionsFindingsErrorsAndHelp(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusBadGateway, http.StatusServiceUnavailable} {
		setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":{"code":"UPSTREAM_ERROR","message":"Rose unavailable"},"meta":{}}`))
		})
		_, _, err := executeCommand("rose", "executions", "findings", roseTestExecutionID)
		require.Error(t, err)
	}
	out, _, err := executeCommand("rose", "executions", "findings", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "--dismissed")
	assert.Contains(t, out, "default \"all\"")
}
