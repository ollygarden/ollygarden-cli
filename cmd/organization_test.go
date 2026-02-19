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

func setupOrgServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
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

func orgResponse(tier, features, score string) string {
	s := `{"data":{"tier":{"name":"` + tier + `","features":[` + features + `]`
	if score != "" {
		s += `},"score":` + score
	} else {
		s += `},"score":null`
	}
	s += `},"meta":{"timestamp":"2026-02-18T12:00:00Z","trace_id":"abc123"}}`
	return s
}

func TestOrganizationHuman(t *testing.T) {
	body := orgResponse("pro", `"webhooks","analytics","api_access"`, `{"value":82,"updated_at":"2026-02-18T12:00:00Z"}`)
	setupOrgServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/organization", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("organization")
	require.NoError(t, err)
	assert.Contains(t, out, "pro")
	assert.Contains(t, out, "webhooks, analytics, api_access")
	assert.Contains(t, out, "82")
	assert.Contains(t, out, "2026-02-18T12:00:00Z")
}

func TestOrganizationJSON(t *testing.T) {
	body := orgResponse("pro", `"webhooks"`, `{"value":82,"updated_at":"2026-02-18T12:00:00Z"}`)
	setupOrgServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("organization", "--json")
	require.NoError(t, err)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "abc123", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), `"pro"`)
}

func TestOrganizationQuiet(t *testing.T) {
	body := orgResponse("pro", `"webhooks"`, `{"value":82,"updated_at":"2026-02-18T12:00:00Z"}`)
	setupOrgServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("organization", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestOrganizationScoreNil(t *testing.T) {
	body := orgResponse("free", `"api_access"`, "")
	setupOrgServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("organization")
	require.NoError(t, err)
	assert.Contains(t, out, "free")
	assert.Contains(t, out, "\u2014") // em dash for nil score
}

func TestOrganizationInsightTypesNil(t *testing.T) {
	body := `{"data":{"tier":{"name":"pro","features":["webhooks"],"allowed_insight_types":null},"score":null},"meta":{}}`
	setupOrgServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("organization")
	require.NoError(t, err)
	assert.Contains(t, out, "all")
}

func TestOrganizationInsightTypesFiltered(t *testing.T) {
	body := `{"data":{"tier":{"name":"starter","features":["api_access"],"allowed_insight_types":["span_naming","missing_attributes"]},"score":null},"meta":{}}`
	setupOrgServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("organization")
	require.NoError(t, err)
	assert.Contains(t, out, "span_naming, missing_attributes")
}

func TestOrganization401(t *testing.T) {
	body := `{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t1"}}`
	setupOrgServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("organization")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestOrganization500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t2"}}`
	setupOrgServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("organization")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestOrganizationHelp(t *testing.T) {
	out, _, err := executeCommand("organization", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "organization")
	assert.Contains(t, out, "Show organization details")
}
