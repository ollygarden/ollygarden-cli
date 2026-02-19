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

func setupServicesVersionsServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_key")
	oldURL := apiURL
	oldLimit := servicesVersionsLimit
	apiURL = srv.URL
	t.Cleanup(func() {
		apiURL = oldURL
		jsonMode = false
		quiet = false
		servicesVersionsLimit = oldLimit
	})
	return srv
}

func versionJSON(id, version, env, lastSeen string, score *int) string {
	s := `{"id":"` + id + `","name":"my-service","version":"` + version + `","environment":"` + env + `","last_seen_at":"` + lastSeen + `"`
	if score != nil {
		s += `,"instrumentation_score":{"score":` + itoa(*score) + `}`
	} else {
		s += `,"instrumentation_score":null`
	}
	s += `}`
	return s
}

func versionsListResponse(versions string, total int, hasMore bool) string {
	return `{"data":[` + versions + `],"meta":{"timestamp":"2026-02-19T12:00:00Z","trace_id":"tr1","total":` +
		itoa(total) + `,"has_more":` + btoa(hasMore) + `}}`
}

func TestServicesVersionsHuman(t *testing.T) {
	v1 := versionJSON("v-1", "v1.0.0", "production", "2026-02-19T10:00:00Z", intPtr(90))
	v2 := versionJSON("v-2", "v2.0.0", "staging", "2026-02-19T09:00:00Z", intPtr(75))
	body := versionsListResponse(v1+","+v2, 2, false)

	setupServicesVersionsServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/services/svc-1/versions", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "versions", "svc-1")
	require.NoError(t, err)
	assert.Contains(t, out, "v1.0.0")
	assert.Contains(t, out, "v2.0.0")
	assert.Contains(t, out, "production")
	assert.Contains(t, out, "staging")
	assert.Contains(t, out, "90")
	assert.Contains(t, out, "75")
}

func TestServicesVersionsJSON(t *testing.T) {
	v1 := versionJSON("v-1", "v1.0.0", "production", "2026-02-19T10:00:00Z", intPtr(90))
	body := versionsListResponse(v1, 1, false)

	setupServicesVersionsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "versions", "svc-1", "--json")
	require.NoError(t, err)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "tr1", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), "v1.0.0")
}

func TestServicesVersionsQuiet(t *testing.T) {
	v1 := versionJSON("v-1", "v1.0.0", "production", "2026-02-19T10:00:00Z", intPtr(90))
	body := versionsListResponse(v1, 1, false)

	setupServicesVersionsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "versions", "svc-1", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestServicesVersionsNilScore(t *testing.T) {
	v1 := versionJSON("v-1", "v1.0.0", "production", "2026-02-19T10:00:00Z", nil)
	body := versionsListResponse(v1, 1, false)

	setupServicesVersionsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "versions", "svc-1")
	require.NoError(t, err)
	assert.Contains(t, out, "\u2014") // em dash for nil score
}

func TestServicesVersionsLimit(t *testing.T) {
	v1 := versionJSON("v-1", "v1.0.0", "production", "2026-02-19T10:00:00Z", intPtr(90))
	body := versionsListResponse(v1, 1, false)

	setupServicesVersionsServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, _, err := executeCommand("services", "versions", "svc-1", "--limit", "10")
	require.NoError(t, err)
}

func TestServicesVersionsInvalidLimit(t *testing.T) {
	setupServicesVersionsServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with invalid limit")
	})

	_, _, err := executeCommand("services", "versions", "svc-1", "--limit", "0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--limit")

	_, _, err = executeCommand("services", "versions", "svc-1", "--limit", "51")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--limit")
}

func TestServicesVersionsMissingArg(t *testing.T) {
	setupServicesVersionsServer(t, func(w http.ResponseWriter, r *http.Request) {})

	_, _, err := executeCommand("services", "versions")
	require.Error(t, err)
}

func TestServicesVersions404(t *testing.T) {
	body := `{"error":{"code":"SERVICE_NOT_FOUND","message":"Service not found"},"meta":{"trace_id":"t1"}}`
	setupServicesVersionsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("services", "versions", "nonexistent")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 4, apiErr.ExitCode())
	assert.Contains(t, stderr, "Service not found")
}

func TestServicesVersions401(t *testing.T) {
	body := `{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t1"}}`
	setupServicesVersionsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("services", "versions", "svc-1")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestServicesVersions500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t2"}}`
	setupServicesVersionsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("services", "versions", "svc-1")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestServicesVersionsHelp(t *testing.T) {
	out, _, err := executeCommand("services", "versions", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "versions")
	assert.Contains(t, out, "List related service versions")
	assert.Contains(t, out, "--limit")
}
