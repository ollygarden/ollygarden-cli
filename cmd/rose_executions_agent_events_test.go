package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const roseAgentEventsResponse = `{"data":{"sessions":[{"id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","runtimeSessionId":"review-main","name":"Review","createdAt":"2026-09-03T10:00:00Z","usage":{"input":10,"output":5,"cacheRead":0,"cacheWrite":0,"reasoning":0,"totalTokens":15,"costTotal":0.01,"turnCount":1}}],"events":[{"seq":4,"createdAt":"2026-09-03T10:00:01Z","agentSessionId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","payload":{"type":"tool_execution_start","toolName":"bash","toolCallId":"call-4"}},{"seq":5,"createdAt":"2026-09-03T10:00:02Z","agentSessionId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","payload":{"type":"message","role":"assistant","preview":"Found two issues","bytes":16}}],"latestSeq":5},"meta":{"trace_id":"trace-events"}}`

func TestRoseExecutionsAgentEventsHumanAndPolling(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/rose/executions/"+roseTestExecutionID+"/agent-events", r.URL.Path)
		assert.Equal(t, "3", r.URL.Query().Get("afterSeq"))
		assert.Equal(t, "25", r.URL.Query().Get("limit"))
		_, _ = w.Write([]byte(roseAgentEventsResponse))
	})

	out, stderr, err := executeCommand("rose", "executions", "agent-events", roseTestExecutionID, "--after-seq", "3", "--limit", "25")
	require.NoError(t, err)
	assert.Contains(t, out, "SEQ")
	assert.Contains(t, out, "tool_execution_start")
	assert.Contains(t, out, "bash")
	assert.Contains(t, out, "assistant: Found two issues")
	assert.Empty(t, stderr)
}

func TestRoseExecutionsAgentEventsEmpty(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"sessions":[],"events":[],"latestSeq":null},"meta":{}}`))
	})
	out, _, err := executeCommand("rose", "executions", "agent-events", roseTestExecutionID)
	require.NoError(t, err)
	assert.Equal(t, "SEQ  CREATED  SESSION  TYPE  DETAIL\n", out)
}

func TestRoseExecutionsAgentEventsJSON(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(roseAgentEventsResponse)) })
	out, _, err := executeCommand("rose", "executions", "agent-events", roseTestExecutionID, "--json")
	require.NoError(t, err)
	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Contains(t, string(envelope.Data), `"latestSeq":5`)
}

func TestRoseExecutionsAgentEventsQuiet(t *testing.T) {
	setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(roseAgentEventsResponse)) })
	out, _, err := executeCommand("rose", "executions", "agent-events", roseTestExecutionID, "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestRoseExecutionsAgentEventsValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"id", []string{"bad"}, "execution-id must be a UUID"},
		{"cursor", []string{roseTestExecutionID, "--after-seq", "-2"}, "--after-seq must be >= -1"},
		{"limit low", []string{roseTestExecutionID, "--limit", "0"}, "--limit must be between 1 and 500"},
		{"limit high", []string{roseTestExecutionID, "--limit", "501"}, "--limit must be between 1 and 500"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupRoseServer(t, func(http.ResponseWriter, *http.Request) { t.Fatal("unexpected request") })
			args := append([]string{"rose", "executions", "agent-events"}, tc.args...)
			_, stderr, err := executeCommand(args...)
			require.Error(t, err)
			assert.Contains(t, stderr, tc.want)
		})
	}
}

func TestRoseExecutionsAgentEventsErrorsAndHelp(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusBadGateway, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			setupRoseServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"error":{"code":"UPSTREAM_ERROR","message":"Rose unavailable"},"meta":{"trace_id":"t1"}}`))
			})
			_, _, err := executeCommand("rose", "executions", "agent-events", roseTestExecutionID)
			require.Error(t, err)
			apiErr := err.(*client.APIError)
			if status == http.StatusNotFound {
				assert.Equal(t, 4, apiErr.ExitCode())
			} else {
				assert.Equal(t, 6, apiErr.ExitCode())
			}
		})
	}
	out, _, err := executeCommand("rose", "executions", "agent-events", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "--after-seq")
	assert.Contains(t, out, "1-500")
}
