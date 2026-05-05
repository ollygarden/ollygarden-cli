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

func setupServicesGroupedServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_key")
	oldURL := apiURL
	oldLimit := servicesGroupedLimit
	oldOffset := servicesGroupedOffset
	oldSort := servicesGroupedSort
	apiURL = srv.URL
	if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
		f.Changed = true
	}
	t.Cleanup(func() {
		apiURL = oldURL
		jsonMode = false
		quiet = false
		servicesGroupedLimit = oldLimit
		servicesGroupedOffset = oldOffset
		servicesGroupedSort = oldSort
		if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
			f.Changed = false
		}
	})
	return srv
}

func groupedServiceJSON(name, env string, versionCount, insightsCount int, score *int) string {
	s := `{"name":"` + name + `","environment":"` + env + `","namespace":"default","latest_id":"aaa-111","version_count":` + itoa(versionCount) + `,"insights_count":` + itoa(insightsCount)
	if score != nil {
		s += `,"instrumentation_score":{"score":` + itoa(*score) + `}`
	} else {
		s += `,"instrumentation_score":null`
	}
	s += `}`
	return s
}

func groupedListResponse(services string, total int, hasMore bool) string {
	return `{"data":[` + services + `],"meta":{"timestamp":"2026-02-19T12:00:00Z","trace_id":"tr1","total":` +
		json.Number(itoa(total)).String() + `,"has_more":` + btoa(hasMore) + `}}`
}

func TestServicesGroupedHuman(t *testing.T) {
	s1 := groupedServiceJSON("api-gateway", "production", 3, 5, intPtr(85))
	s2 := groupedServiceJSON("auth-service", "staging", 1, 0, intPtr(42))
	body := groupedListResponse(s1+","+s2, 2, false)

	setupServicesGroupedServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/services/grouped", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "grouped")
	require.NoError(t, err)
	assert.Contains(t, out, "api-gateway")
	assert.Contains(t, out, "auth-service")
	assert.Contains(t, out, "production")
	assert.Contains(t, out, "staging")
	assert.Contains(t, out, "3")
	assert.Contains(t, out, "5")
	assert.Contains(t, out, "85")
	assert.Contains(t, out, "42")
}

func TestServicesGroupedJSON(t *testing.T) {
	s1 := groupedServiceJSON("api-gateway", "production", 3, 5, intPtr(85))
	body := groupedListResponse(s1, 1, false)

	setupServicesGroupedServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "grouped", "--json")
	require.NoError(t, err)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "tr1", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), "api-gateway")
}

func TestServicesGroupedQuiet(t *testing.T) {
	s1 := groupedServiceJSON("api-gateway", "production", 3, 5, intPtr(85))
	body := groupedListResponse(s1, 1, false)

	setupServicesGroupedServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "grouped", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestServicesGroupedPagination(t *testing.T) {
	s1 := groupedServiceJSON("api-gateway", "production", 3, 5, intPtr(85))
	body := groupedListResponse(s1, 75, true)

	setupServicesGroupedServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("services", "grouped")
	require.NoError(t, err)
	assert.Contains(t, stderr, "more results")
	assert.Contains(t, stderr, "--offset 50")
}

func TestServicesGroupedNilScore(t *testing.T) {
	s1 := groupedServiceJSON("api-gateway", "production", 3, 5, nil)
	body := groupedListResponse(s1, 1, false)

	setupServicesGroupedServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	out, _, err := executeCommand("services", "grouped")
	require.NoError(t, err)
	assert.Contains(t, out, "\u2014") // em dash for nil score
}

func TestServicesGroupedFlags(t *testing.T) {
	s1 := groupedServiceJSON("api-gateway", "production", 3, 5, intPtr(85))
	body := groupedListResponse(s1, 1, false)

	setupServicesGroupedServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		assert.Equal(t, "20", r.URL.Query().Get("offset"))
		assert.Equal(t, "name-asc", r.URL.Query().Get("sort"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, _, err := executeCommand("services", "grouped", "--limit", "10", "--offset", "20", "--sort", "name-asc")
	require.NoError(t, err)
}

func TestServicesGroupedSortFlag(t *testing.T) {
	s1 := groupedServiceJSON("api-gateway", "production", 3, 5, intPtr(85))
	body := groupedListResponse(s1, 1, false)

	setupServicesGroupedServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "created-desc", r.URL.Query().Get("sort"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})

	_, _, err := executeCommand("services", "grouped", "--sort", "created-desc")
	require.NoError(t, err)
}

func TestServicesGroupedInvalidLimit(t *testing.T) {
	setupServicesGroupedServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with invalid limit")
	})

	_, _, err := executeCommand("services", "grouped", "--limit", "0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--limit")
}

func TestServicesGroupedInvalidOffset(t *testing.T) {
	setupServicesGroupedServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with invalid offset")
	})

	_, _, err := executeCommand("services", "grouped", "--offset", "-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--offset")
}

func TestServicesGroupedInvalidSort(t *testing.T) {
	setupServicesGroupedServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with invalid sort")
	})

	_, _, err := executeCommand("services", "grouped", "--sort", "bad-value")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--sort")
}

func TestServicesGrouped401(t *testing.T) {
	body := `{"error":{"code":"INVALID_API_KEY","message":"Invalid API key"},"meta":{"trace_id":"t1"}}`
	setupServicesGroupedServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("services", "grouped")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 3, apiErr.ExitCode())
	assert.Contains(t, stderr, "Invalid API key")
}

func TestServicesGrouped500(t *testing.T) {
	body := `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"},"meta":{"trace_id":"t2"}}`
	setupServicesGroupedServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(body))
	})

	_, stderr, err := executeCommand("services", "grouped")
	require.Error(t, err)
	apiErr, ok := err.(*client.APIError)
	require.True(t, ok)
	assert.Equal(t, 6, apiErr.ExitCode())
	assert.Contains(t, stderr, "Internal server error")
}

func TestServicesGroupedHelp(t *testing.T) {
	out, _, err := executeCommand("services", "grouped", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "List services grouped by name")
	assert.Contains(t, out, "--limit")
	assert.Contains(t, out, "--offset")
	assert.Contains(t, out, "--sort")
}
