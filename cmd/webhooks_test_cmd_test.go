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

func setupWebhooksTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_key")
	oldURL := apiURL
	var oldAPIURLChanged bool
	if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
		oldAPIURLChanged = f.Changed
	}
	t.Cleanup(func() {
		apiURL = oldURL
		jsonMode = false
		quiet = false
		if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
			f.Changed = oldAPIURLChanged
		}
	})
	apiURL = srv.URL
	if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
		f.Changed = true
	}
	return srv
}

func webhookTestSuccessResponse() string {
	return `{"data":{"success":true,"status_code":200,"response_body":"OK"},` +
		`"meta":{"timestamp":"2026-02-19T10:00:00Z","trace_id":"trace-test"}}`
}

func webhookTestFailureResponse() string {
	return `{"data":{"success":false,"status_code":500,"response_body":"Internal Server Error"},` +
		`"meta":{"timestamp":"2026-02-19T10:00:00Z","trace_id":"trace-test-fail"}}`
}

func TestWebhooksTestHuman(t *testing.T) {
	setupWebhooksTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/webhooks/wh-123/test", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(webhookTestSuccessResponse()))
	})

	out, _, err := executeCommand("webhooks", "test", "wh-123")
	require.NoError(t, err)
	assert.Contains(t, out, "true")
	assert.Contains(t, out, "200")
	assert.Contains(t, out, "OK")
}

func TestWebhooksTestJSON(t *testing.T) {
	setupWebhooksTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(webhookTestSuccessResponse()))
	})

	out, _, err := executeCommand("webhooks", "test", "wh-123", "--json")
	require.NoError(t, err)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "trace-test", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), "true")
}

func TestWebhooksTestQuiet(t *testing.T) {
	setupWebhooksTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(webhookTestSuccessResponse()))
	})

	out, _, err := executeCommand("webhooks", "test", "wh-123", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestWebhooksTestFailureResult(t *testing.T) {
	setupWebhooksTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(webhookTestFailureResponse()))
	})

	out, _, err := executeCommand("webhooks", "test", "wh-123")
	require.NoError(t, err)
	assert.Contains(t, out, "false")
	assert.Contains(t, out, "500")
	assert.Contains(t, out, "Internal Server Error")
}

func TestWebhooksTest404(t *testing.T) {
	body := `{"error":{"code":"WEBHOOK_NOT_FOUND","message":"Webhook not found"},"meta":{"trace_id":"t1"}}`
	setupWebhooksTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "test", "wh-missing")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 4, apiErr.ExitCode())
	assert.Contains(t, stderr, "Webhook not found")
}

func TestWebhooksTest401(t *testing.T) {
	body := `{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t2"}}`
	setupWebhooksTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "test", "wh-123")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestWebhooksTest500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t3"}}`
	setupWebhooksTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "test", "wh-123")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestWebhooksTestNoArgs(t *testing.T) {
	setupWebhooksTestServer(t, func(w http.ResponseWriter, r *http.Request) {})

	_, _, err := executeCommand("webhooks", "test")
	require.Error(t, err)
}

func TestWebhooksTestHelp(t *testing.T) {
	out, _, err := executeCommand("webhooks", "test", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "test <webhook-id>")
	assert.Contains(t, out, "Test a webhook")
}
