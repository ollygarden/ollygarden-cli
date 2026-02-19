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

func setupWebhooksGetServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_key")
	oldURL := apiURL
	apiURL = srv.URL
	t.Cleanup(func() {
		apiURL = oldURL
		jsonMode = false
		quiet = false
	})
	return srv
}

func webhookGetResponse(id, name, url, severity string, enabled bool, eventTypes, environments string) string {
	e := "false"
	if enabled {
		e = "true"
	}
	return `{"data":{"id":"` + id + `","name":"` + name + `","url":"` + url + `","is_enabled":` + e +
		`,"min_severity":"` + severity + `","event_types":` + eventTypes + `,"environments":` + environments +
		`,"organization_id":"org-1","created_at":"2026-02-19T09:00:00Z","updated_at":"2026-02-19T12:00:00Z"}` +
		`,"meta":{"timestamp":"2026-02-19T12:00:00Z","trace_id":"abc123"},"links":{"self":"/api/v1/webhooks/` + id + `"}}`
}

func TestWebhooksGetHuman(t *testing.T) {
	body := webhookGetResponse("wh-111", "deploy-alerts", "https://example.com/hook", "Low", true,
		`["insight.created","insight.resolved"]`, `["production","staging"]`)

	setupWebhooksGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/webhooks/wh-111", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("webhooks", "get", "wh-111")
	require.NoError(t, err)
	assert.Contains(t, out, "wh-111")
	assert.Contains(t, out, "deploy-alerts")
	assert.Contains(t, out, "https://example.com/hook")
	assert.Contains(t, out, "true")
	assert.Contains(t, out, "Low")
	assert.Contains(t, out, "insight.created, insight.resolved")
	assert.Contains(t, out, "production, staging")
	assert.Contains(t, out, "2026-02-19T09:00:00Z")
	assert.Contains(t, out, "2026-02-19T12:00:00Z")
}

func TestWebhooksGetJSON(t *testing.T) {
	body := webhookGetResponse("wh-111", "deploy-alerts", "https://example.com/hook", "Low", true,
		`["insight.created"]`, `["production"]`)

	setupWebhooksGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("webhooks", "get", "wh-111", "--json")
	require.NoError(t, err)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "abc123", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), "deploy-alerts")
}

func TestWebhooksGetQuiet(t *testing.T) {
	body := webhookGetResponse("wh-111", "deploy-alerts", "https://example.com/hook", "Low", true,
		`[]`, `[]`)

	setupWebhooksGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("webhooks", "get", "wh-111", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestWebhooksGetEmptyArrays(t *testing.T) {
	body := webhookGetResponse("wh-111", "deploy-alerts", "https://example.com/hook", "Low", true,
		`[]`, `[]`)

	setupWebhooksGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("webhooks", "get", "wh-111")
	require.NoError(t, err)
	assert.Contains(t, out, "all")
}

func TestWebhooksGetMissingArg(t *testing.T) {
	setupWebhooksGetServer(t, func(w http.ResponseWriter, r *http.Request) {})

	_, _, err := executeCommand("webhooks", "get")
	require.Error(t, err)
}

func TestWebhooksGet404(t *testing.T) {
	body := `{"error":{"code":"WEBHOOK_NOT_FOUND","message":"Webhook not found"},"meta":{"trace_id":"t1"}}`
	setupWebhooksGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "get", "nonexistent")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 4, apiErr.ExitCode())
	assert.Contains(t, stderr, "Webhook not found")
}

func TestWebhooksGet401(t *testing.T) {
	body := `{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t1"}}`
	setupWebhooksGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "get", "wh-111")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestWebhooksGet500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t2"}}`
	setupWebhooksGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "get", "wh-111")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestWebhooksGetHelp(t *testing.T) {
	out, _, err := executeCommand("webhooks", "get", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "get")
	assert.Contains(t, out, "Show webhook details")
}
