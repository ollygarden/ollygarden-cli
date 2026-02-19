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

func setupWebhooksListServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_key")
	oldURL := apiURL
	oldLimit := webhooksListLimit
	oldOffset := webhooksListOffset
	apiURL = srv.URL
	t.Cleanup(func() {
		apiURL = oldURL
		jsonMode = false
		quiet = false
		webhooksListLimit = oldLimit
		webhooksListOffset = oldOffset
	})
	return srv
}

func webhooksListResponse(webhooks string, total int, hasMore bool) string {
	return `{"data":[` + webhooks + `],"meta":{"timestamp":"2026-02-19T12:00:00Z","trace_id":"tr1","total":` +
		itoa(total) + `,"has_more":` + btoa(hasMore) + `}}`
}

func webhookJSON(id, name, url, severity string, enabled bool) string {
	e := "false"
	if enabled {
		e = "true"
	}
	return `{"id":"` + id + `","name":"` + name + `","url":"` + url + `","is_enabled":` + e + `,"min_severity":"` + severity + `"}`
}

func TestWebhooksListHuman(t *testing.T) {
	w1 := webhookJSON("wh-111", "deploy-alerts", "https://example.com/hook1", "Low", true)
	w2 := webhookJSON("wh-222", "error-alerts", "https://example.com/hook2", "Critical", false)
	body := webhooksListResponse(w1+","+w2, 2, false)

	setupWebhooksListServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/webhooks", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("webhooks", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "deploy-alerts")
	assert.Contains(t, out, "error-alerts")
	assert.Contains(t, out, "https://example.com/hook1")
	assert.Contains(t, out, "true")
	assert.Contains(t, out, "false")
	assert.Contains(t, out, "Low")
	assert.Contains(t, out, "Critical")
}

func TestWebhooksListJSON(t *testing.T) {
	w1 := webhookJSON("wh-111", "deploy-alerts", "https://example.com/hook1", "Low", true)
	body := webhooksListResponse(w1, 1, false)

	setupWebhooksListServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("webhooks", "list", "--json")
	require.NoError(t, err)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "tr1", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), "deploy-alerts")
}

func TestWebhooksListQuiet(t *testing.T) {
	w1 := webhookJSON("wh-111", "deploy-alerts", "https://example.com/hook1", "Low", true)
	body := webhooksListResponse(w1, 1, false)

	setupWebhooksListServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("webhooks", "list", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestWebhooksListPagination(t *testing.T) {
	w1 := webhookJSON("wh-111", "deploy-alerts", "https://example.com/hook1", "Low", true)
	body := webhooksListResponse(w1, 75, true)

	setupWebhooksListServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "list")
	require.NoError(t, err)
	assert.Contains(t, stderr, "more results")
	assert.Contains(t, stderr, "--offset 50")
}

func TestWebhooksListFlags(t *testing.T) {
	w1 := webhookJSON("wh-111", "deploy-alerts", "https://example.com/hook1", "Low", true)
	body := webhooksListResponse(w1, 1, false)

	setupWebhooksListServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		assert.Equal(t, "20", r.URL.Query().Get("offset"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, _, err := executeCommand("webhooks", "list", "--limit", "10", "--offset", "20")
	require.NoError(t, err)
}

func TestWebhooksListInvalidLimit(t *testing.T) {
	setupWebhooksListServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with invalid limit")
	})

	_, _, err := executeCommand("webhooks", "list", "--limit", "0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--limit")
}

func TestWebhooksListInvalidOffset(t *testing.T) {
	setupWebhooksListServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with invalid offset")
	})

	_, _, err := executeCommand("webhooks", "list", "--offset", "-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--offset")
}

func TestWebhooksList401(t *testing.T) {
	body := `{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t1"}}`
	setupWebhooksListServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "list")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestWebhooksList500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t2"}}`
	setupWebhooksListServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "list")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestWebhooksListHelp(t *testing.T) {
	out, _, err := executeCommand("webhooks", "list", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "List webhooks")
	assert.Contains(t, out, "--limit")
	assert.Contains(t, out, "--offset")
}
