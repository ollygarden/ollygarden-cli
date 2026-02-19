package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupWebhooksUpdateServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_key")
	oldURL := apiURL
	t.Cleanup(func() {
		apiURL = oldURL
		jsonMode = false
		quiet = false
		webhooksUpdateName = ""
		webhooksUpdateURL = ""
		webhooksUpdateEventTypes = nil
		webhooksUpdateEnvironments = nil
		webhooksUpdateMinSeverity = ""
		webhooksUpdateEnabled = false
		// Reset cobra Changed state to prevent leaking between tests
		webhooksUpdateCmd.Flags().VisitAll(func(f *pflag.Flag) {
			f.Changed = false
		})
	})
	apiURL = srv.URL
	return srv
}

func webhookUpdateResponse() string {
	return `{"data":{"id":"wh-123","name":"updated-hook","url":"https://example.com/updated","is_enabled":true,` +
		`"min_severity":"Important","event_types":["insight.created"],"environments":["production"],` +
		`"organization_id":"org-1","created_at":"2026-02-19T10:00:00Z","updated_at":"2026-02-19T11:00:00Z"},` +
		`"meta":{"timestamp":"2026-02-19T11:00:00Z","trace_id":"trace-update"},"links":{"self":"/api/v1/webhooks/wh-123"}}`
}

func TestWebhooksUpdateHuman(t *testing.T) {
	setupWebhooksUpdateServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/api/v1/webhooks/wh-123", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(webhookUpdateResponse()))
	})

	out, _, err := executeCommand("webhooks", "update", "wh-123", "--name", "updated-hook", "--enabled")
	require.NoError(t, err)
	assert.Contains(t, out, "wh-123")
	assert.Contains(t, out, "updated-hook")
	assert.Contains(t, out, "https://example.com/updated")
	assert.Contains(t, out, "true")
	assert.Contains(t, out, "Important")
	assert.Contains(t, out, "insight.created")
	assert.Contains(t, out, "production")
}

func TestWebhooksUpdateJSON(t *testing.T) {
	setupWebhooksUpdateServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(webhookUpdateResponse()))
	})

	out, _, err := executeCommand("webhooks", "update", "wh-123", "--name", "updated-hook", "--json")
	require.NoError(t, err)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "trace-update", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), "updated-hook")
}

func TestWebhooksUpdateQuiet(t *testing.T) {
	setupWebhooksUpdateServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(webhookUpdateResponse()))
	})

	out, _, err := executeCommand("webhooks", "update", "wh-123", "--name", "updated-hook", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestWebhooksUpdatePartialBody(t *testing.T) {
	var requestBody map[string]any

	setupWebhooksUpdateServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &requestBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(webhookUpdateResponse()))
	})

	_, _, err := executeCommand("webhooks", "update", "wh-123", "--name", "only-name")
	require.NoError(t, err)

	// Only "name" should be in the request body
	assert.Contains(t, requestBody, "name")
	assert.Equal(t, "only-name", requestBody["name"])
	assert.NotContains(t, requestBody, "url")
	assert.NotContains(t, requestBody, "is_enabled")
	assert.NotContains(t, requestBody, "min_severity")
	assert.NotContains(t, requestBody, "event_types")
	assert.NotContains(t, requestBody, "environments")
}

func TestWebhooksUpdateNoFlags(t *testing.T) {
	setupWebhooksUpdateServer(t, func(w http.ResponseWriter, r *http.Request) {})

	_, _, err := executeCommand("webhooks", "update", "wh-123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one flag")
}

func TestWebhooksUpdateMissingArg(t *testing.T) {
	setupWebhooksUpdateServer(t, func(w http.ResponseWriter, r *http.Request) {})

	_, _, err := executeCommand("webhooks", "update")
	require.Error(t, err)
}

func TestWebhooksUpdateInvalidMinSeverity(t *testing.T) {
	setupWebhooksUpdateServer(t, func(w http.ResponseWriter, r *http.Request) {})

	_, _, err := executeCommand("webhooks", "update", "wh-123", "--min-severity", "invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--min-severity")
}

func TestWebhooksUpdateNameTooLong(t *testing.T) {
	setupWebhooksUpdateServer(t, func(w http.ResponseWriter, r *http.Request) {})

	longName := strings.Repeat("x", 256)
	_, _, err := executeCommand("webhooks", "update", "wh-123", "--name", longName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "255")
}

func TestWebhooksUpdateRepeatableFlags(t *testing.T) {
	var requestBody map[string]any

	setupWebhooksUpdateServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &requestBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(webhookUpdateResponse()))
	})

	_, _, err := executeCommand("webhooks", "update", "wh-123",
		"--event-type", "insight.created", "--event-type", "insight.resolved",
		"--environment", "production", "--environment", "staging")
	require.NoError(t, err)

	eventTypes := requestBody["event_types"].([]any)
	assert.Equal(t, []any{"insight.created", "insight.resolved"}, eventTypes)

	environments := requestBody["environments"].([]any)
	assert.Equal(t, []any{"production", "staging"}, environments)
}

func TestWebhooksUpdate401(t *testing.T) {
	body := `{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t1"}}`
	setupWebhooksUpdateServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "update", "wh-123", "--name", "test")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestWebhooksUpdate404(t *testing.T) {
	body := `{"error":{"code":"WEBHOOK_NOT_FOUND","message":"Webhook not found"},"meta":{"trace_id":"t2"}}`
	setupWebhooksUpdateServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "update", "wh-123", "--name", "test")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 4, apiErr.ExitCode())
	assert.Contains(t, stderr, "Webhook not found")
}

func TestWebhooksUpdate500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t3"}}`
	setupWebhooksUpdateServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "update", "wh-123", "--name", "test")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestWebhooksUpdateHelp(t *testing.T) {
	out, _, err := executeCommand("webhooks", "update", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "update")
	assert.Contains(t, out, "Update a webhook")
	assert.Contains(t, out, "--name")
	assert.Contains(t, out, "--url")
	assert.Contains(t, out, "--event-type")
	assert.Contains(t, out, "--environment")
	assert.Contains(t, out, "--min-severity")
	assert.Contains(t, out, "--enabled")
}
