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

func setupServicesServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_key")
	oldURL := apiURL
	oldLimit := servicesListLimit
	oldOffset := servicesListOffset
	apiURL = srv.URL
	if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
		f.Changed = true
	}
	t.Cleanup(func() {
		apiURL = oldURL
		jsonMode = false
		quiet = false
		servicesListLimit = oldLimit
		servicesListOffset = oldOffset
		if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
			f.Changed = false
		}
	})
	return srv
}

func servicesListResponse(services string, total int, hasMore bool) string {
	return `{"data":[` + services + `],"meta":{"timestamp":"2026-02-19T12:00:00Z","trace_id":"tr1","total":` +
		json.Number(itoa(total)).String() + `,"has_more":` + btoa(hasMore) + `}}`
}

func itoa(i int) string {
	return json.Number(func() string {
		b, _ := json.Marshal(i)
		return string(b)
	}()).String()
}

func btoa(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func svcJSON(id, name, env, lastSeen string, score *int) string {
	s := `{"id":"` + id + `","name":"` + name + `","environment":"` + env + `","last_seen_at":"` + lastSeen + `"`
	if score != nil {
		s += `,"instrumentation_score":{"score":` + itoa(*score) + `}`
	} else {
		s += `,"instrumentation_score":null`
	}
	s += `}`
	return s
}

func intPtr(i int) *int { return &i }

func TestServicesListHuman(t *testing.T) {
	s1 := svcJSON("aaa-111", "api-gateway", "production", "2026-02-19T10:00:00Z", intPtr(85))
	s2 := svcJSON("bbb-222", "auth-service", "staging", "2026-02-19T09:00:00Z", intPtr(42))
	body := servicesListResponse(s1+","+s2, 2, false)

	setupServicesServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/services", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "api-gateway")
	assert.Contains(t, out, "auth-service")
	assert.Contains(t, out, "production")
	assert.Contains(t, out, "staging")
	assert.Contains(t, out, "85")
	assert.Contains(t, out, "42")
}

func TestServicesListJSON(t *testing.T) {
	s1 := svcJSON("aaa-111", "api-gateway", "production", "2026-02-19T10:00:00Z", intPtr(85))
	body := servicesListResponse(s1, 1, false)

	setupServicesServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "list", "--json")
	require.NoError(t, err)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "tr1", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), "api-gateway")
}

func TestServicesListQuiet(t *testing.T) {
	s1 := svcJSON("aaa-111", "api-gateway", "production", "2026-02-19T10:00:00Z", intPtr(85))
	body := servicesListResponse(s1, 1, false)

	setupServicesServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "list", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestServicesListPagination(t *testing.T) {
	s1 := svcJSON("aaa-111", "api-gateway", "production", "2026-02-19T10:00:00Z", intPtr(85))
	body := servicesListResponse(s1, 75, true)

	setupServicesServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("services", "list")
	require.NoError(t, err)
	assert.Contains(t, stderr, "more results")
	assert.Contains(t, stderr, "--offset 50")
}

func TestServicesListNilScore(t *testing.T) {
	s1 := svcJSON("aaa-111", "api-gateway", "production", "2026-02-19T10:00:00Z", nil)
	body := servicesListResponse(s1, 1, false)

	setupServicesServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "\u2014") // em dash for nil score
}

func TestServicesListFlags(t *testing.T) {
	s1 := svcJSON("aaa-111", "api-gateway", "production", "2026-02-19T10:00:00Z", intPtr(85))
	body := servicesListResponse(s1, 1, false)

	setupServicesServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		assert.Equal(t, "20", r.URL.Query().Get("offset"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, _, err := executeCommand("services", "list", "--limit", "10", "--offset", "20")
	require.NoError(t, err)
}

func TestServicesListInvalidLimit(t *testing.T) {
	setupServicesServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with invalid limit")
	})

	_, _, err := executeCommand("services", "list", "--limit", "0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--limit")
}

func TestServicesListInvalidOffset(t *testing.T) {
	setupServicesServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with invalid offset")
	})

	_, _, err := executeCommand("services", "list", "--offset", "-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--offset")
}

func TestServicesList401(t *testing.T) {
	body := `{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t1"}}`
	setupServicesServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("services", "list")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestServicesList500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t2"}}`
	setupServicesServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("services", "list")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestServicesListHelp(t *testing.T) {
	out, _, err := executeCommand("services", "list", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "List services")
	assert.Contains(t, out, "--limit")
	assert.Contains(t, out, "--offset")
}
