package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAPIKeysListServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	resetHelpFlag(t, apiKeysListCmd)
	oldLimit := apiKeysListLimit
	oldOffset := apiKeysListOffset
	setupAPIServer(t, handler)
	t.Cleanup(func() {
		apiKeysListLimit = oldLimit
		apiKeysListOffset = oldOffset
	})
}

func apiKeysListResponse(total int, hasMore bool) string {
	return `{"data":[{"id":"key-1","key_preview":"og_sk_***abc","description":"Production",` +
		`"created_at":"2026-09-06T10:00:00Z","last_seen_at":"2026-09-06T11:00:00Z","is_active":true},` +
		`{"id":"key-2","key_preview":"og_sk_***def","description":"CI",` +
		`"created_at":"2026-09-05T10:00:00Z","last_seen_at":null,"is_active":true}],` +
		`"meta":{"timestamp":"2026-09-06T12:00:00Z","trace_id":"trace-list","total":` +
		itoa(total) + `,"has_more":` + btoa(hasMore) + `}}`
}

func TestAPIKeysListHuman(t *testing.T) {
	setupAPIKeysListServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/api-keys", r.URL.Path)
		assert.Equal(t, "50", r.URL.Query().Get("limit"))
		assert.Equal(t, "0", r.URL.Query().Get("offset"))
		_, _ = w.Write([]byte(apiKeysListResponse(2, false)))
	})

	out, stderr, err := executeCommand("api-keys", "list")
	require.NoError(t, err)
	assert.Empty(t, stderr)
	assert.Contains(t, out, "ID")
	assert.Contains(t, out, "DESCRIPTION")
	assert.Contains(t, out, "KEY")
	assert.Contains(t, out, "LAST USED")
	assert.Contains(t, out, "key-1")
	assert.Contains(t, out, "Production")
	assert.Contains(t, out, "og_sk_***abc")
	assert.Contains(t, out, "2026-09-06T11:00:00Z")
	assert.Contains(t, out, emDash)
}

func TestAPIKeysListJSON(t *testing.T) {
	setupAPIKeysListServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(apiKeysListResponse(2, false)))
	})

	out, stderr, err := executeCommand("api-keys", "list", "--json")
	require.NoError(t, err)
	assert.Empty(t, stderr)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "trace-list", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), "og_sk_***abc")
	assert.NotContains(t, string(envelope.Data), `"key":`)
}

func TestAPIKeysListQuiet(t *testing.T) {
	setupAPIKeysListServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(apiKeysListResponse(2, false)))
	})

	out, stderr, err := executeCommand("api-keys", "list", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
	assert.Empty(t, stderr)
}

func TestAPIKeysListPagination(t *testing.T) {
	setupAPIKeysListServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "2", r.URL.Query().Get("limit"))
		assert.Equal(t, "4", r.URL.Query().Get("offset"))
		_, _ = w.Write([]byte(apiKeysListResponse(10, true)))
	})

	_, stderr, err := executeCommand("api-keys", "list", "--limit", "2", "--offset", "4")
	require.NoError(t, err)
	assert.Contains(t, stderr, "# 4 more results. Use --offset 6")
}

func TestAPIKeysListRejectsInvalidPaginationBeforeRequest(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{name: "zero limit", args: []string{"--limit", "0"}, want: "--limit"},
		{name: "high limit", args: []string{"--limit", "101"}, want: "--limit"},
		{name: "negative offset", args: []string{"--offset", "-1"}, want: "--offset"},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupAPIKeysListServer(t, func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("request must not be sent for invalid pagination")
			})

			_, stderr, err := executeCommand(append([]string{"api-keys", "list"}, test.args...)...)
			require.Error(t, err)
			apiErr, ok := err.(*client.APIError)
			require.True(t, ok)
			assert.Equal(t, 2, apiErr.ExitCode())
			assert.Contains(t, stderr, test.want)
		})
	}
}

func TestAPIKeysListAPIErrors(t *testing.T) {
	for _, test := range []struct {
		status int
		code   string
		exit   int
	}{
		{status: http.StatusBadRequest, code: "INVALID_PARAMETERS", exit: 2},
		{status: http.StatusUnauthorized, code: "INVALID_API_KEY", exit: 3},
		{status: http.StatusTooManyRequests, code: "RATE_LIMIT_EXCEEDED", exit: 5},
		{status: http.StatusInternalServerError, code: "INTERNAL_ERROR", exit: 6},
	} {
		t.Run(test.code, func(t *testing.T) {
			setupAPIKeysListServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(`{"error":{"code":"` + test.code + `","message":"request failed"},"meta":{"trace_id":"trace-error"}}`))
			})

			_, stderr, err := executeCommand("api-keys", "list")
			require.Error(t, err)
			apiErr, ok := err.(*client.APIError)
			require.True(t, ok)
			assert.Equal(t, test.exit, apiErr.ExitCode())
			assert.Contains(t, stderr, "request failed")
			assert.NotContains(t, stderr, "og_sk_")
		})
	}
}

func TestAPIKeysListHelp(t *testing.T) {
	out, _, err := executeCommand("api-keys", "list", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "List API keys")
	assert.Contains(t, out, "--limit")
	assert.Contains(t, out, "--offset")
}
