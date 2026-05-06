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

func setupWebhooksDeliveriesListServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_key")
	oldURL := apiURL
	oldLimit := webhooksDeliveriesListLimit
	oldOffset := webhooksDeliveriesListOffset
	var oldAPIURLChanged bool
	apiURL = srv.URL
	if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
		oldAPIURLChanged = f.Changed
		f.Changed = true
	}
	t.Cleanup(func() {
		apiURL = oldURL
		jsonMode = false
		quiet = false
		webhooksDeliveriesListLimit = oldLimit
		webhooksDeliveriesListOffset = oldOffset
		if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
			f.Changed = oldAPIURLChanged
		}
	})
	return srv
}

func deliveryJSON(id, status string, httpCode, attempts int, createdAt string) string {
	return `{"id":"` + id + `","status":"` + status + `","http_status_code":` + itoa(httpCode) +
		`,"attempt_number":` + itoa(attempts) + `,"created_at":"` + createdAt + `"}`
}

func deliveriesListResponse(deliveries string, total int, hasMore bool) string {
	return `{"data":[` + deliveries + `],"meta":{"timestamp":"2026-02-19T12:00:00Z","trace_id":"tr1","total":` +
		itoa(total) + `,"has_more":` + btoa(hasMore) + `}}`
}

func TestWebhooksDeliveriesListHuman(t *testing.T) {
	d1 := deliveryJSON("del-111", "success", 200, 1, "2026-02-19T10:00:00Z")
	d2 := deliveryJSON("del-222", "failed", 500, 3, "2026-02-19T11:00:00Z")
	body := deliveriesListResponse(d1+","+d2, 2, false)

	setupWebhooksDeliveriesListServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/webhooks/wh-111/deliveries", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("webhooks", "deliveries", "list", "wh-111")
	require.NoError(t, err)
	assert.Contains(t, out, "del-111")
	assert.Contains(t, out, "del-222")
	assert.Contains(t, out, "success")
	assert.Contains(t, out, "failed")
	assert.Contains(t, out, "200")
	assert.Contains(t, out, "500")
}

func TestWebhooksDeliveriesListHTTPZero(t *testing.T) {
	d1 := deliveryJSON("del-333", "pending", 0, 0, "2026-02-19T10:00:00Z")
	body := deliveriesListResponse(d1, 1, false)

	setupWebhooksDeliveriesListServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("webhooks", "deliveries", "list", "wh-111")
	require.NoError(t, err)
	assert.Contains(t, out, "—")
	assert.Contains(t, out, "pending")
}

func TestWebhooksDeliveriesListJSON(t *testing.T) {
	d1 := deliveryJSON("del-111", "success", 200, 1, "2026-02-19T10:00:00Z")
	body := deliveriesListResponse(d1, 1, false)

	setupWebhooksDeliveriesListServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("webhooks", "deliveries", "list", "wh-111", "--json")
	require.NoError(t, err)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "tr1", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), "del-111")
}

func TestWebhooksDeliveriesListQuiet(t *testing.T) {
	d1 := deliveryJSON("del-111", "success", 200, 1, "2026-02-19T10:00:00Z")
	body := deliveriesListResponse(d1, 1, false)

	setupWebhooksDeliveriesListServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("webhooks", "deliveries", "list", "wh-111", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestWebhooksDeliveriesListPagination(t *testing.T) {
	d1 := deliveryJSON("del-111", "success", 200, 1, "2026-02-19T10:00:00Z")
	body := deliveriesListResponse(d1, 75, true)

	setupWebhooksDeliveriesListServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "deliveries", "list", "wh-111")
	require.NoError(t, err)
	assert.Contains(t, stderr, "more results")
	assert.Contains(t, stderr, "--offset 50")
}

func TestWebhooksDeliveriesListFlags(t *testing.T) {
	d1 := deliveryJSON("del-111", "success", 200, 1, "2026-02-19T10:00:00Z")
	body := deliveriesListResponse(d1, 1, false)

	setupWebhooksDeliveriesListServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		assert.Equal(t, "20", r.URL.Query().Get("offset"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, _, err := executeCommand("webhooks", "deliveries", "list", "wh-111", "--limit", "10", "--offset", "20")
	require.NoError(t, err)
}

func TestWebhooksDeliveriesListInvalidLimit(t *testing.T) {
	setupWebhooksDeliveriesListServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with invalid limit")
	})

	_, _, err := executeCommand("webhooks", "deliveries", "list", "wh-111", "--limit", "0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--limit")
}

func TestWebhooksDeliveriesListInvalidOffset(t *testing.T) {
	setupWebhooksDeliveriesListServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with invalid offset")
	})

	_, _, err := executeCommand("webhooks", "deliveries", "list", "wh-111", "--offset", "-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--offset")
}

func TestWebhooksDeliveriesListMissingArg(t *testing.T) {
	setupWebhooksDeliveriesListServer(t, func(w http.ResponseWriter, r *http.Request) {})

	_, _, err := executeCommand("webhooks", "deliveries", "list")
	require.Error(t, err)
}

func TestWebhooksDeliveriesList401(t *testing.T) {
	body := `{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t1"}}`
	setupWebhooksDeliveriesListServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "deliveries", "list", "wh-111")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestWebhooksDeliveriesList404(t *testing.T) {
	body := `{"error":{"code":"WEBHOOK_NOT_FOUND","message":"Webhook not found"},"meta":{"trace_id":"t1"}}`
	setupWebhooksDeliveriesListServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "deliveries", "list", "nonexistent")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 4, apiErr.ExitCode())
	assert.Contains(t, stderr, "Webhook not found")
}

func TestWebhooksDeliveriesList500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t2"}}`
	setupWebhooksDeliveriesListServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("webhooks", "deliveries", "list", "wh-111")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestWebhooksDeliveriesListHelp(t *testing.T) {
	out, _, err := executeCommand("webhooks", "deliveries", "list", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "List deliveries for a webhook")
	assert.Contains(t, out, "--limit")
	assert.Contains(t, out, "--offset")
}
