package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func roseFindingsListResponse(hasMore bool) string {
	pagination := `{"limit":2,"offset":0,"total":2,"hasMore":false}`
	if hasMore {
		pagination = `{"limit":2,"offset":0,"total":29,"hasMore":true}`
	}
	return `{"data":{"data":[
		{"finding_id":"otel-6576685e6e1f","repository_id":"repo-1","repo_full_name":"acme/checkout","repo_url":"https://github.com/acme/checkout","severity":"critical","category":"Sensitive Data","title":"Config logged","display_title":"Secrets exposure: config object logged","pr_number":112,"pr_status":"open","checked":null},
		{"finding_id":"otel-80d3f9847f36","repository_id":"repo-2","repo_full_name":"acme/ingest","repo_url":"https://github.com/acme/ingest","severity":"high","category":"Volume","title":"Debug logs at INFO","display_title":null,"pr_number":null,"pr_status":null,"checked":null}
	],"pagination":` + pagination + `},"meta":{"timestamp":"2026-08-22T02:20:00Z","trace_id":"trace-list"}}`
}

func TestRoseFindingsListHuman(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/rose/findings", r.URL.Path)
		q := r.URL.Query()
		assert.Equal(t, "1", q.Get("page"))
		assert.Equal(t, "50", q.Get("page_size"))
		assert.Equal(t, "false", q.Get("dismissed"))
		assert.Empty(t, q.Get("severity"))
		w.Write([]byte(roseFindingsListResponse(false)))
	})

	out, stderr, err := executeCommand("rose", "findings", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "otel-6576685e6e1f")
	assert.Contains(t, out, "Secrets exposure: config object logged") // display_title preferred
	assert.Contains(t, out, "Debug logs at INFO")                     // title fallback
	assert.Contains(t, out, "#112 (open)")
	assert.Contains(t, out, "acme/ingest")
	assert.Contains(t, out, "—") // em dash for missing PR
	assert.NotContains(t, stderr, "more results")
}

func TestRoseFindingsListFilters(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "critical,high", q.Get("severity"))
		assert.Equal(t, "Sensitive Data", q.Get("category"))
		assert.Equal(t, "all", q.Get("dismissed"))
		assert.Equal(t, "2", q.Get("page"))
		assert.Equal(t, "10", q.Get("page_size"))
		w.Write([]byte(roseFindingsListResponse(false)))
	})

	_, _, err := executeCommand("rose", "findings", "list",
		"--severity", "critical,high", "--category", "Sensitive Data",
		"--dismissed", "all", "--page", "2", "--limit", "10")
	require.NoError(t, err)
}

func TestRoseFindingsListPageHint(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(roseFindingsListResponse(true)))
	})

	_, stderr, err := executeCommand("rose", "findings", "list", "--limit", "2")
	require.NoError(t, err)
	assert.Contains(t, stderr, "# 27 more results. Use --page 2 to see next page.")
}

func TestRoseFindingsListJSON(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(roseFindingsListResponse(false)))
	})

	out, _, err := executeCommand("rose", "findings", "list", "--json")
	require.NoError(t, err)
	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "trace-list", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), `"pagination"`)
}

func TestRoseFindingsListQuiet(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(roseFindingsListResponse(false)))
	})

	out, _, err := executeCommand("rose", "findings", "list", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestRoseFindingsListBadFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		message string
	}{
		{"limit below range", []string{"--limit", "0"}, "--limit must be between 1 and 100"},
		{"limit above range", []string{"--limit", "101"}, "--limit must be between 1 and 100"},
		{"page below range", []string{"--page", "0"}, "--page must be >= 1"},
		{"invalid severity", []string{"--severity", "bogus"}, "--severity must contain only: critical, high, medium, low, suggestion"},
		{"mixed invalid severity", []string{"--severity", "critical,bogus"}, "--severity must contain only: critical, high, medium, low, suggestion"},
		{"invalid dismissed", []string{"--dismissed", "yes"}, "--dismissed must be one of: false, true, all"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
				t.Error("no request expected for invalid flags")
			})

			args := append([]string{"rose", "findings", "list"}, tt.args...)
			out, stderr, err := executeCommand(args...)
			require.Error(t, err)
			apiErr, ok := err.(*client.APIError)
			require.True(t, ok)
			assert.Equal(t, 2, apiErr.ExitCode())
			assert.Empty(t, out)
			assert.Equal(t, "Error: "+tt.message+"\n", stderr)
		})
	}
}

func TestRoseFindingsListBadFlagsJSON(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request expected for invalid flags")
	})

	out, stderr, err := executeCommand("rose", "findings", "list", "--limit", "0", "--json")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 2, apiErr.ExitCode())
	assert.Empty(t, out)
	var envelope client.ErrorResponse
	require.NoError(t, json.Unmarshal([]byte(stderr), &envelope))
	assert.Equal(t, "INVALID_PARAMETERS", envelope.Error.Code)
	assert.Equal(t, "--limit must be between 1 and 100", envelope.Error.Message)
}

func TestRoseFindingsListHelp(t *testing.T) {
	out, _, err := executeCommand("rose", "findings", "list", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "--dismissed")
	assert.Contains(t, out, "false, true, all")
}
