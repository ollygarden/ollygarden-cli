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

func setupWebhooksDeliveriesGetServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
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

func deliveryGetResponse(id, status string, httpCode, attempts int, errorMsg, completedAt, insightID, webhookID, idempotencyKey, createdAt string) string {
	errField := "null"
	if errorMsg != "" {
		errField = `"` + errorMsg + `"`
	}
	compField := "null"
	if completedAt != "" {
		compField = `"` + completedAt + `"`
	}
	return `{"data":{"id":"` + id + `","status":"` + status + `","http_status_code":` + itoa(httpCode) +
		`,"attempt_number":` + itoa(attempts) + `,"error_message":` + errField +
		`,"idempotency_key":"` + idempotencyKey + `","insight_id":"` + insightID +
		`","webhook_config_id":"` + webhookID + `","organization_id":"org-1"` +
		`,"created_at":"` + createdAt + `","completed_at":` + compField +
		`},"meta":{"timestamp":"2026-02-19T12:00:00Z","trace_id":"tr1"}}`
}

func TestWebhooksDeliveriesGetHuman(t *testing.T) {
	body := deliveryGetResponse("del-111", "success", 200, 1,
		"", "2026-02-19T11:00:00Z", "ins-111", "wh-111", "key-111", "2026-02-19T10:00:00Z")

	setupWebhooksDeliveriesGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/webhooks/wh-111/deliveries/del-111", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("webhooks", "deliveries", "get", "wh-111", "del-111")
	require.NoError(t, err)
	assert.Contains(t, out, "del-111")
	assert.Contains(t, out, "success")
	assert.Contains(t, out, "200")
	assert.Contains(t, out, "ins-111")
	assert.Contains(t, out, "wh-111")
	assert.Contains(t, out, "key-111")
	assert.Contains(t, out, "2026-02-19T10:00:00Z")
	assert.Contains(t, out, "2026-02-19T11:00:00Z")
}

func TestWebhooksDeliveriesGetJSON(t *testing.T) {
	body := deliveryGetResponse("del-111", "success", 200, 1,
		"", "2026-02-19T11:00:00Z", "ins-111", "wh-111", "key-111", "2026-02-19T10:00:00Z")

	setupWebhooksDeliveriesGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("webhooks", "deliveries", "get", "wh-111", "del-111", "--json")
	require.NoError(t, err)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "tr1", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), "del-111")
}

func TestWebhooksDeliveriesGetQuiet(t *testing.T) {
	body := deliveryGetResponse("del-111", "success", 200, 1,
		"", "2026-02-19T11:00:00Z", "ins-111", "wh-111", "key-111", "2026-02-19T10:00:00Z")

	setupWebhooksDeliveriesGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("webhooks", "deliveries", "get", "wh-111", "del-111", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestWebhooksDeliveriesGetNullFields(t *testing.T) {
	body := deliveryGetResponse("del-222", "pending", 0, 0,
		"", "", "ins-222", "wh-222", "key-222", "2026-02-19T10:00:00Z")

	setupWebhooksDeliveriesGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("webhooks", "deliveries", "get", "wh-222", "del-222")
	require.NoError(t, err)
	// HTTP status 0 and null completed_at/error_message should show "—"
	assert.Contains(t, out, "—")
	assert.Contains(t, out, "pending")
}

func TestWebhooksDeliveriesGetWithError(t *testing.T) {
	body := deliveryGetResponse("del-333", "failed", 500, 3,
		"connection refused", "2026-02-19T11:00:00Z", "ins-333", "wh-333", "key-333", "2026-02-19T10:00:00Z")

	setupWebhooksDeliveriesGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("webhooks", "deliveries", "get", "wh-333", "del-333")
	require.NoError(t, err)
	assert.Contains(t, out, "connection refused")
	assert.Contains(t, out, "failed")
	assert.Contains(t, out, "500")
}

func TestWebhooksDeliveriesGetMissingArgs(t *testing.T) {
	setupWebhooksDeliveriesGetServer(t, func(w http.ResponseWriter, r *http.Request) {})

	_, _, err := executeCommand("webhooks", "deliveries", "get")
	require.Error(t, err)

	_, _, err = executeCommand("webhooks", "deliveries", "get", "wh-111")
	require.Error(t, err)
}

func TestWebhooksDeliveriesGet401(t *testing.T) {
	body := `{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t1"}}`
	setupWebhooksDeliveriesGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "deliveries", "get", "wh-111", "del-111")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestWebhooksDeliveriesGet404WebhookNotFound(t *testing.T) {
	body := `{"error":{"code":"WEBHOOK_NOT_FOUND","message":"Webhook not found"},"meta":{"trace_id":"t1"}}`
	setupWebhooksDeliveriesGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "deliveries", "get", "nonexistent", "del-111")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 4, apiErr.ExitCode())
	assert.Contains(t, stderr, "Webhook not found")
}

func TestWebhooksDeliveriesGet404DeliveryNotFound(t *testing.T) {
	body := `{"error":{"code":"DELIVERY_NOT_FOUND","message":"Delivery not found"},"meta":{"trace_id":"t1"}}`
	setupWebhooksDeliveriesGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "deliveries", "get", "wh-111", "nonexistent")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 4, apiErr.ExitCode())
	assert.Contains(t, stderr, "Delivery not found")
}

func TestWebhooksDeliveriesGet500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t2"}}`
	setupWebhooksDeliveriesGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "deliveries", "get", "wh-111", "del-111")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestWebhooksDeliveriesGetHelp(t *testing.T) {
	out, _, err := executeCommand("webhooks", "deliveries", "get", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "Show delivery details")
	assert.Contains(t, out, "get")
}
