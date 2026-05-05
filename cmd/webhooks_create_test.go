package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupWebhooksCreateServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_key")
	oldURL := apiURL
	t.Cleanup(func() {
		apiURL = oldURL
		jsonMode = false
		quiet = false
		// Reset flag values
		webhooksCreateName = ""
		webhooksCreateURL = ""
		webhooksCreateEventTypes = nil
		webhooksCreateEnvironments = nil
		webhooksCreateMinSeverity = "Low"
		webhooksCreateEnabled = false
		if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
			f.Changed = false
		}
	})
	apiURL = srv.URL
	if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
		f.Changed = true
	}
	return srv
}

func webhookCreateResponse() string {
	return `{"data":{"id":"wh-new","name":"my-hook","url":"https://example.com/hook","is_enabled":true,` +
		`"min_severity":"Normal","event_types":["insight.created"],"environments":["production"],` +
		`"organization_id":"org-1","created_at":"2026-02-19T10:00:00Z","updated_at":"2026-02-19T10:00:00Z"},` +
		`"meta":{"timestamp":"2026-02-19T10:00:00Z","trace_id":"trace-create"},"links":{"self":"/api/v1/webhooks/wh-new"}}`
}

func TestWebhooksCreateHuman(t *testing.T) {
	setupWebhooksCreateServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/webhooks", r.URL.Path)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(webhookCreateResponse()))
	})

	out, _, err := executeCommand("webhooks", "create", "--name", "my-hook", "--url", "https://example.com/hook",
		"--event-type", "insight.created", "--environment", "production", "--min-severity", "Normal", "--enabled")
	require.NoError(t, err)
	assert.Contains(t, out, "wh-new")
	assert.Contains(t, out, "my-hook")
	assert.Contains(t, out, "https://example.com/hook")
	assert.Contains(t, out, "true")
	assert.Contains(t, out, "Normal")
	assert.Contains(t, out, "insight.created")
	assert.Contains(t, out, "production")
}

func TestWebhooksCreateJSON(t *testing.T) {
	setupWebhooksCreateServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(webhookCreateResponse()))
	})

	out, _, err := executeCommand("webhooks", "create", "--name", "my-hook", "--url", "https://example.com/hook", "--json")
	require.NoError(t, err)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "trace-create", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), "my-hook")
}

func TestWebhooksCreateQuiet(t *testing.T) {
	setupWebhooksCreateServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(webhookCreateResponse()))
	})

	out, _, err := executeCommand("webhooks", "create", "--name", "my-hook", "--url", "https://example.com/hook", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestWebhooksCreateMissingName(t *testing.T) {
	setupWebhooksCreateServer(t, func(w http.ResponseWriter, r *http.Request) {})

	_, _, err := executeCommand("webhooks", "create", "--url", "https://example.com/hook")
	require.Error(t, err)
}

func TestWebhooksCreateMissingURL(t *testing.T) {
	setupWebhooksCreateServer(t, func(w http.ResponseWriter, r *http.Request) {})

	_, _, err := executeCommand("webhooks", "create", "--name", "my-hook")
	require.Error(t, err)
}

func TestWebhooksCreateInvalidMinSeverity(t *testing.T) {
	setupWebhooksCreateServer(t, func(w http.ResponseWriter, r *http.Request) {})

	_, _, err := executeCommand("webhooks", "create", "--name", "my-hook", "--url", "https://example.com/hook", "--min-severity", "invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--min-severity")
}

func TestWebhooksCreateNameTooLong(t *testing.T) {
	setupWebhooksCreateServer(t, func(w http.ResponseWriter, r *http.Request) {})

	longName := strings.Repeat("x", 256)
	_, _, err := executeCommand("webhooks", "create", "--name", longName, "--url", "https://example.com/hook")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "255")
}

func TestWebhooksCreateRepeatableFlags(t *testing.T) {
	var requestBody webhookCreateRequest

	setupWebhooksCreateServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &requestBody)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(webhookCreateResponse()))
	})

	_, _, err := executeCommand("webhooks", "create", "--name", "my-hook", "--url", "https://example.com/hook",
		"--event-type", "insight.created", "--event-type", "insight.resolved",
		"--environment", "production", "--environment", "staging")
	require.NoError(t, err)
	assert.Equal(t, []string{"insight.created", "insight.resolved"}, requestBody.EventTypes)
	assert.Equal(t, []string{"production", "staging"}, requestBody.Environments)
}

func TestWebhooksCreate401(t *testing.T) {
	body := `{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t1"}}`
	setupWebhooksCreateServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "create", "--name", "my-hook", "--url", "https://example.com/hook")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestWebhooksCreate500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t2"}}`
	setupWebhooksCreateServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "create", "--name", "my-hook", "--url", "https://example.com/hook")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestWebhooksCreateHelp(t *testing.T) {
	out, _, err := executeCommand("webhooks", "create", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "create")
	assert.Contains(t, out, "Create a webhook")
	assert.Contains(t, out, "--name")
	assert.Contains(t, out, "--url")
	assert.Contains(t, out, "--event-type")
	assert.Contains(t, out, "--environment")
	assert.Contains(t, out, "--min-severity")
	assert.Contains(t, out, "--enabled")
}
