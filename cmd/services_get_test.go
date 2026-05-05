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

func setupServicesGetServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_key")
	oldURL := apiURL
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
		if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
			f.Changed = oldAPIURLChanged
		}
	})
	return srv
}

func serviceGetResponse(id, name, version, environment, namespace string, score *int) string {
	scoreJSON := "null"
	if score != nil {
		scoreJSON = `{"id":"score-1","score":` + json.Number(itoa(*score)).String() + `,"calculated_timestamp":"2026-02-18T12:00:00Z","calculation_window_seconds":3600,"evaluated_rule_ids":[],"created_at":"2026-02-18T12:00:00Z"}`
	}
	return `{"data":{"id":"` + id + `","name":"` + name + `","version":"` + version + `","environment":"` + environment + `","namespace":"` + namespace + `","organization_id":"org-1","created_at":"2026-01-15T10:00:00Z","updated_at":"2026-02-18T14:00:00Z","first_seen_at":"2026-01-15T10:30:00Z","last_seen_at":"2026-02-18T14:22:00Z","instrumentation_score":` + scoreJSON + `},"meta":{"timestamp":"2026-02-18T12:00:00Z","trace_id":"abc123"},"links":{"insights":"/api/v1/services/` + id + `/insights"}}`
}

func TestServicesGetHuman(t *testing.T) {
	score := 82
	body := serviceGetResponse("svc-1", "payment-service", "v1.2.3", "production", "backend", &score)
	setupServicesGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/services/svc-1", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "get", "svc-1")
	require.NoError(t, err)
	assert.Contains(t, out, "svc-1")
	assert.Contains(t, out, "payment-service")
	assert.Contains(t, out, "v1.2.3")
	assert.Contains(t, out, "production")
	assert.Contains(t, out, "backend")
	assert.Contains(t, out, "2026-01-15T10:30:00Z")
	assert.Contains(t, out, "2026-02-18T14:22:00Z")
	assert.Contains(t, out, "82")
}

func TestServicesGetJSON(t *testing.T) {
	score := 82
	body := serviceGetResponse("svc-1", "payment-service", "v1.2.3", "production", "backend", &score)
	setupServicesGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "get", "svc-1", "--json")
	require.NoError(t, err)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "abc123", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), `"payment-service"`)
}

func TestServicesGetQuiet(t *testing.T) {
	score := 82
	body := serviceGetResponse("svc-1", "payment-service", "v1.2.3", "production", "backend", &score)
	setupServicesGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "get", "svc-1", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestServicesGetNilScore(t *testing.T) {
	body := serviceGetResponse("svc-1", "payment-service", "v1.2.3", "production", "backend", nil)
	setupServicesGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "get", "svc-1")
	require.NoError(t, err)
	assert.Contains(t, out, "\u2014") // em dash for nil score
}

func TestServicesGetEmptyOptionalFields(t *testing.T) {
	body := serviceGetResponse("svc-1", "payment-service", "", "", "", nil)
	setupServicesGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "get", "svc-1")
	require.NoError(t, err)
	// Version, Environment, Namespace should show em dash
	// Count em dashes: score (nil) + version + environment + namespace = at least 4
	assert.Contains(t, out, "\u2014")
	assert.Contains(t, out, "payment-service")
}

func TestServicesGetMissingArg(t *testing.T) {
	setupServicesGetServer(t, func(w http.ResponseWriter, r *http.Request) {})

	_, _, err := executeCommand("services", "get")
	require.Error(t, err)
}

func TestServicesGet404(t *testing.T) {
	body := `{"error":{"code":"SERVICE_NOT_FOUND","message":"Service not found"},"meta":{"trace_id":"t1"}}`
	setupServicesGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("services", "get", "nonexistent")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 4, apiErr.ExitCode())
	assert.Contains(t, stderr, "Service not found")
}

func TestServicesGet401(t *testing.T) {
	body := `{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t1"}}`
	setupServicesGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("services", "get", "svc-1")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestServicesGet500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t2"}}`
	setupServicesGetServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("services", "get", "svc-1")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestServicesGetHelp(t *testing.T) {
	out, _, err := executeCommand("services", "get", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "get")
	assert.Contains(t, out, "Show service details")
}
