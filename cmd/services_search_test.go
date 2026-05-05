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

func setupServicesSearchServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_key")
	oldURL := apiURL
	oldQuery := servicesSearchQuery
	oldLimit := servicesSearchLimit
	oldOffset := servicesSearchOffset
	oldEnv := servicesSearchEnvironment
	oldNs := servicesSearchNamespace
	apiURL = srv.URL
	if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
		f.Changed = true
	}
	t.Cleanup(func() {
		apiURL = oldURL
		jsonMode = false
		quiet = false
		servicesSearchQuery = oldQuery
		servicesSearchLimit = oldLimit
		servicesSearchOffset = oldOffset
		servicesSearchEnvironment = oldEnv
		servicesSearchNamespace = oldNs
		if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
			f.Changed = false
		}
	})
	return srv
}

func servicesSearchResponse(services string, total int, hasMore bool) string {
	return `{"data":[` + services + `],"meta":{"timestamp":"2026-02-19T12:00:00Z","trace_id":"tr1","total":` +
		json.Number(itoa(total)).String() + `,"has_more":` + btoa(hasMore) + `}}`
}

func TestServicesSearchHumanPositional(t *testing.T) {
	s1 := svcJSON("aaa-111", "api-gateway", "production", "2026-02-19T10:00:00Z", intPtr(85))
	s2 := svcJSON("bbb-222", "api-proxy", "staging", "2026-02-19T09:00:00Z", intPtr(42))
	body := servicesSearchResponse(s1+","+s2, 2, false)

	setupServicesSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/services/search", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "api", r.URL.Query().Get("q"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "search", "api")
	require.NoError(t, err)
	assert.Contains(t, out, "api-gateway")
	assert.Contains(t, out, "api-proxy")
	assert.Contains(t, out, "production")
	assert.Contains(t, out, "staging")
	assert.Contains(t, out, "85")
	assert.Contains(t, out, "42")
}

func TestServicesSearchHumanFlag(t *testing.T) {
	s1 := svcJSON("aaa-111", "api-gateway", "production", "2026-02-19T10:00:00Z", intPtr(85))
	body := servicesSearchResponse(s1, 1, false)

	setupServicesSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "gateway", r.URL.Query().Get("q"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "search", "--query", "gateway")
	require.NoError(t, err)
	assert.Contains(t, out, "api-gateway")
}

func TestServicesSearchJSON(t *testing.T) {
	s1 := svcJSON("aaa-111", "api-gateway", "production", "2026-02-19T10:00:00Z", intPtr(85))
	body := servicesSearchResponse(s1, 1, false)

	setupServicesSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "search", "api", "--json")
	require.NoError(t, err)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "tr1", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), "api-gateway")
}

func TestServicesSearchQuiet(t *testing.T) {
	s1 := svcJSON("aaa-111", "api-gateway", "production", "2026-02-19T10:00:00Z", intPtr(85))
	body := servicesSearchResponse(s1, 1, false)

	setupServicesSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "search", "api", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestServicesSearchPagination(t *testing.T) {
	s1 := svcJSON("aaa-111", "api-gateway", "production", "2026-02-19T10:00:00Z", intPtr(85))
	body := servicesSearchResponse(s1, 50, true)

	setupServicesSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("services", "search", "api")
	require.NoError(t, err)
	assert.Contains(t, stderr, "more results")
	assert.Contains(t, stderr, "--offset 20")
}

func TestServicesSearchNilScore(t *testing.T) {
	s1 := svcJSON("aaa-111", "api-gateway", "production", "2026-02-19T10:00:00Z", nil)
	body := servicesSearchResponse(s1, 1, false)

	setupServicesSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "search", "api")
	require.NoError(t, err)
	assert.Contains(t, out, "\u2014") // em dash for nil score
}

func TestServicesSearchFlags(t *testing.T) {
	s1 := svcJSON("aaa-111", "api-gateway", "production", "2026-02-19T10:00:00Z", intPtr(85))
	body := servicesSearchResponse(s1, 1, false)

	setupServicesSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		assert.Equal(t, "5", r.URL.Query().Get("offset"))
		assert.Equal(t, "production", r.URL.Query().Get("environment"))
		assert.Equal(t, "default", r.URL.Query().Get("namespace"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, _, err := executeCommand("services", "search", "api", "--limit", "10", "--offset", "5", "--environment", "production", "--namespace", "default")
	require.NoError(t, err)
}

func TestServicesSearchQueryParam(t *testing.T) {
	s1 := svcJSON("aaa-111", "api-gateway", "production", "2026-02-19T10:00:00Z", intPtr(85))
	body := servicesSearchResponse(s1, 1, false)

	setupServicesSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "my-service", r.URL.Query().Get("q"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, _, err := executeCommand("services", "search", "my-service")
	require.NoError(t, err)
}

func TestServicesSearchMissingQuery(t *testing.T) {
	setupServicesSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server without query")
	})

	_, _, err := executeCommand("services", "search")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query is required")
}

func TestServicesSearchInvalidLimit(t *testing.T) {
	setupServicesSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with invalid limit")
	})

	_, _, err := executeCommand("services", "search", "api", "--limit", "0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--limit")
}

func TestServicesSearchInvalidOffset(t *testing.T) {
	setupServicesSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with invalid offset")
	})

	_, _, err := executeCommand("services", "search", "api", "--offset", "-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--offset")
}

func TestServicesSearch401(t *testing.T) {
	body := `{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t1"}}`
	setupServicesSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("services", "search", "api")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestServicesSearch500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t2"}}`
	setupServicesSearchServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("services", "search", "api")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestServicesSearchHelp(t *testing.T) {
	out, _, err := executeCommand("services", "search", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "Search services")
	assert.Contains(t, out, "--query")
	assert.Contains(t, out, "--limit")
	assert.Contains(t, out, "--offset")
	assert.Contains(t, out, "--environment")
	assert.Contains(t, out, "--namespace")
}
