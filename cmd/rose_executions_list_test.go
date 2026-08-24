package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func roseExecutionsListResponse(hasMore bool) string {
	pagination := `{"limit":2,"offset":0,"total":2,"hasMore":false}`
	if hasMore {
		pagination = `{"limit":2,"offset":0,"total":649,"hasMore":true}`
	}
	return `{"data":{"data":[
		{"id":"exec-1","executionType":"review","status":"completed","triggerSource":"scheduled","ref":"main","commitSha":"4f2e9c1ab34f2e9c1ab34f2e9c1ab34f2e9c1ab3","startedAt":"2026-08-22 02:10:44.1+00","completedAt":"2026-08-22 02:18:03.2+00","repositoryId":"repo-1","installationId":"inst-1","repoOwner":"acme","repoName":"checkout","currentPhase":null},
		{"id":"exec-2","executionType":"fix","status":"running","triggerSource":"manual","ref":"main","commitSha":"aaa","startedAt":"2026-08-22 09:01:12+00","completedAt":null,"repositoryId":"repo-1","installationId":"inst-1","repoOwner":"acme","repoName":"checkout","currentPhase":{"id":"ph-1","key":"implementing_fix","label":"Implement fix","state":"running"}}
	],"pagination":` + pagination + `},"meta":{"timestamp":"2026-08-22T02:20:00Z","trace_id":"trace-execs"}}`
}

func TestRoseExecutionsListHuman(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/rose/executions", r.URL.Path)
		q := r.URL.Query()
		assert.Equal(t, "50", q.Get("limit"))
		assert.Equal(t, "0", q.Get("offset"))
		assert.Empty(t, q.Get("status"))
		w.Write([]byte(roseExecutionsListResponse(false)))
	})

	out, stderr, err := executeCommand("rose", "executions", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "exec-1")
	assert.Contains(t, out, "review")
	assert.Contains(t, out, "completed")
	assert.Contains(t, out, "acme/checkout")
	assert.Contains(t, out, "2026-08-22T02:10:44Z")
	assert.Contains(t, out, "running")
	assert.Contains(t, out, "—") // running execution has no completedAt
	assert.NotContains(t, stderr, "more results")
}

func TestRoseExecutionsListFilters(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "5", q.Get("limit"))
		assert.Equal(t, "10", q.Get("offset"))
		assert.Equal(t, "failed", q.Get("status"))
		assert.Equal(t, "repo-1", q.Get("repositoryId"))
		assert.Equal(t, "review,fix", q.Get("executionType"))
		w.Write([]byte(roseExecutionsListResponse(false)))
	})

	_, _, err := executeCommand("rose", "executions", "list",
		"--limit", "5", "--offset", "10", "--status", "failed",
		"--repository-id", "repo-1", "--type", "review,fix")
	require.NoError(t, err)
}

func TestRoseExecutionsListPaginationHint(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(roseExecutionsListResponse(true)))
	})

	_, stderr, err := executeCommand("rose", "executions", "list", "--limit", "2")
	require.NoError(t, err)
	assert.Contains(t, stderr, "# 647 more results. Use --offset 2 to see next page.")
}

func TestRoseExecutionsListJSON(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(roseExecutionsListResponse(false)))
	})

	out, _, err := executeCommand("rose", "executions", "list", "--json")
	require.NoError(t, err)
	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "trace-execs", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), `"executionType":"review"`)
}

func TestRoseExecutionsListQuiet(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(roseExecutionsListResponse(false)))
	})

	out, _, err := executeCommand("rose", "executions", "list", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestRoseExecutionsListBadFlags(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request expected for invalid flags")
	})

	for _, args := range [][]string{
		{"rose", "executions", "list", "--limit", "0"},
		{"rose", "executions", "list", "--limit", "101"},
		{"rose", "executions", "list", "--offset", "-1"},
	} {
		_, _, err := executeCommand(args...)
		require.Error(t, err, "args: %v", args)
	}
}

func TestRoseExecutionsList429(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"code":"RATE_LIMIT_EXCEEDED","message":"Rate limit exceeded"},"meta":{"trace_id":"t1"}}`))
	})

	_, stderr, err := executeCommand("rose", "executions", "list")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 5, apiErr.ExitCode())
	assert.Contains(t, stderr, "Rate limit exceeded")
}
