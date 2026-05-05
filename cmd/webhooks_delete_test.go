package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupWebhooksDeleteServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_key")

	// Reset cobra flag state that persists across Execute() calls
	webhooksDeleteCmd.Flags().Set("help", "false")

	oldURL := apiURL
	oldIsTerminal := stdinIsTerminal
	oldReader := stdinReader
	t.Cleanup(func() {
		apiURL = oldURL
		jsonMode = false
		quiet = false
		webhooksDeleteConfirm = false
		stdinIsTerminal = oldIsTerminal
		stdinReader = oldReader
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

func webhookDeleteGetResponse() string {
	return `{"data":{"id":"wh-del","name":"my-webhook","url":"https://example.com/hook","is_enabled":true,` +
		`"min_severity":"Low","event_types":[],"environments":[],` +
		`"organization_id":"org-1","created_at":"2026-02-19T09:00:00Z","updated_at":"2026-02-19T12:00:00Z"},` +
		`"meta":{"timestamp":"2026-02-19T12:00:00Z","trace_id":"abc123"},"links":{"self":"/api/v1/webhooks/wh-del"}}`
}

func TestWebhooksDeleteHuman(t *testing.T) {
	var gotDelete bool
	setupWebhooksDeleteServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			assert.Equal(t, "/api/v1/webhooks/wh-del", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(webhookDeleteGetResponse()))
		case "DELETE":
			assert.Equal(t, "/api/v1/webhooks/wh-del", r.URL.Path)
			gotDelete = true
			w.WriteHeader(http.StatusNoContent)
		}
	})

	_, stderr, err := executeCommand("webhooks", "delete", "wh-del", "--confirm")
	require.NoError(t, err)
	assert.True(t, gotDelete)
	assert.Contains(t, stderr, `Deleted webhook "my-webhook" (id: wh-del).`)
}

func TestWebhooksDeleteJSON(t *testing.T) {
	setupWebhooksDeleteServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(webhookDeleteGetResponse()))
		case "DELETE":
			w.WriteHeader(http.StatusNoContent)
		}
	})

	out, stderr, err := executeCommand("webhooks", "delete", "wh-del", "--confirm", "--json")
	require.NoError(t, err)
	assert.Empty(t, out)
	assert.NotContains(t, stderr, "Deleted")
}

func TestWebhooksDeleteQuiet(t *testing.T) {
	setupWebhooksDeleteServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(webhookDeleteGetResponse()))
		case "DELETE":
			w.WriteHeader(http.StatusNoContent)
		}
	})

	out, stderr, err := executeCommand("webhooks", "delete", "wh-del", "--confirm", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
	assert.Empty(t, stderr)
}

func TestWebhooksDelete404(t *testing.T) {
	body := `{"error":{"code":"WEBHOOK_NOT_FOUND","message":"Webhook not found"},"meta":{"trace_id":"t1"}}`
	setupWebhooksDeleteServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "delete", "nonexistent", "--confirm")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 4, apiErr.ExitCode())
	assert.Contains(t, stderr, "Webhook not found")
}

func TestWebhooksDelete401(t *testing.T) {
	body := `{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t1"}}`
	setupWebhooksDeleteServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "delete", "wh-del", "--confirm")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestWebhooksDelete500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t2"}}`
	setupWebhooksDeleteServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(webhookDeleteGetResponse()))
		case "DELETE":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(body))
		}
	})

	_, stderr, err := executeCommand("webhooks", "delete", "wh-del", "--confirm")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestWebhooksDeleteNoArgs(t *testing.T) {
	setupWebhooksDeleteServer(t, func(w http.ResponseWriter, r *http.Request) {})

	_, _, err := executeCommand("webhooks", "delete")
	require.Error(t, err)
}

func TestWebhooksDeleteNonTTYNoConfirm(t *testing.T) {
	setupWebhooksDeleteServer(t, func(w http.ResponseWriter, r *http.Request) {})
	stdinIsTerminal = func() bool { return false }

	_, _, err := executeCommand("webhooks", "delete", "wh-del")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--confirm required for non-interactive webhook deletion")
}

func TestWebhooksDeleteTTYConfirmPromptYes(t *testing.T) {
	var gotDelete bool
	setupWebhooksDeleteServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(webhookDeleteGetResponse()))
		case "DELETE":
			gotDelete = true
			w.WriteHeader(http.StatusNoContent)
		}
	})

	stdinIsTerminal = func() bool { return true }
	stdinReader = io.NopCloser(strings.NewReader("y\n"))

	_, stderr, err := executeCommand("webhooks", "delete", "wh-del")
	require.NoError(t, err)
	assert.True(t, gotDelete)
	assert.Contains(t, stderr, `Delete webhook "my-webhook" (id: wh-del)? [y/N]:`)
	assert.Contains(t, stderr, `Deleted webhook "my-webhook" (id: wh-del).`)
}

func TestWebhooksDeleteTTYConfirmPromptNo(t *testing.T) {
	var gotDelete bool
	setupWebhooksDeleteServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(webhookDeleteGetResponse()))
		case "DELETE":
			gotDelete = true
			w.WriteHeader(http.StatusNoContent)
		}
	})

	stdinIsTerminal = func() bool { return true }
	stdinReader = io.NopCloser(strings.NewReader("n\n"))

	_, stderr, err := executeCommand("webhooks", "delete", "wh-del")
	require.NoError(t, err)
	assert.False(t, gotDelete)
	assert.Contains(t, stderr, `Delete webhook "my-webhook" (id: wh-del)? [y/N]:`)
	assert.Contains(t, stderr, "Aborted.")
}

// Help test at end to avoid cobra's --help flag persisting across tests
func TestWebhooksDeleteHelp(t *testing.T) {
	out, _, err := executeCommand("webhooks", "delete", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "delete")
	assert.Contains(t, out, "Delete a webhook")
	assert.Contains(t, out, "--confirm")
}
