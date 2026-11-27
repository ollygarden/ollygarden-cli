package cmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupWebhooksTestServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	setupAPIServer(t, handler)
}

func webhookTestResponseBody(data string) string {
	return `{"data":` + data + `,"meta":{"timestamp":"2026-02-19T10:00:00Z","trace_id":"trace-test"}}`
}

func webhookTestSuccessResponse() string {
	return webhookTestResponseBody(`{"success":true,"status_code":200}`)
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
	assert.Equal(t, "Success:      true\nStatus Code:  200\n", out)
}

func TestWebhooksTestJSON(t *testing.T) {
	data := `{"success":false,"status_code":0,"error":"timeout"}`
	setupWebhooksTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(webhookTestResponseBody(data)))
	})

	out, _, err := executeCommand("webhooks", "test", "wh-123", "--json")
	require.NoError(t, err)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "trace-test", envelope.Meta.TraceID)
	assert.JSONEq(t, data, string(envelope.Data))
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

func TestWebhooksTestResults(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "non-2xx status",
			data: `{"success":false,"status_code":500}`,
			want: "Success:      false\nStatus Code:  500\n",
		},
		{
			name: "redirect reported as-is",
			data: `{"success":false,"status_code":302}`,
			want: "Success:      false\nStatus Code:  302\n",
		},
		{
			name: "blocked destination",
			data: `{"success":false,"status_code":0,"error":"blocked_destination"}`,
			want: "Success:      false\nStatus Code:  0\nError:        blocked_destination\n" +
				"Hint:         The URL must resolve to a public address and must not point at OllyGarden hosts.\n",
		},
		{
			name: "timeout",
			data: `{"success":false,"status_code":0,"error":"timeout"}`,
			want: "Success:      false\nStatus Code:  0\nError:        timeout\n" +
				"Hint:         The endpoint did not respond in time. Check that it is reachable.\n",
		},
		{
			name: "tls error",
			data: `{"success":false,"status_code":0,"error":"tls_error"}`,
			want: "Success:      false\nStatus Code:  0\nError:        tls_error\n" +
				"Hint:         The TLS handshake failed. Check the endpoint's certificate.\n",
		},
		{
			name: "connection failed",
			data: `{"success":false,"status_code":0,"error":"connection_failed"}`,
			want: "Success:      false\nStatus Code:  0\nError:        connection_failed\n" +
				"Hint:         Could not connect. Check the endpoint's host and port.\n",
		},
		{
			name: "unknown category without hint",
			data: `{"success":false,"status_code":0,"error":"dns_failure"}`,
			want: "Success:      false\nStatus Code:  0\nError:        dns_failure\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupWebhooksTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(webhookTestResponseBody(tt.data)))
			})

			out, _, err := executeCommand("webhooks", "test", "wh-123")
			require.NoError(t, err)
			assert.Equal(t, tt.want, out)
		})
	}
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
