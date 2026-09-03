package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupServicesGroupedServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	oldLimit := servicesGroupedLimit
	oldOffset := servicesGroupedOffset
	oldSort := servicesGroupedSort
	oldQuery := servicesGroupedQuery
	oldView := servicesGroupedView
	oldEnvironment := servicesGroupedEnvironment
	oldMinScore := servicesGroupedMinScore
	oldMaxScore := servicesGroupedMaxScore
	oldHasInsightType := servicesGroupedHasInsightType
	oldOrder := servicesGroupedOrder
	oldSnapshot := servicesGroupedSnapshot
	for _, name := range []string{"sort", "query", "view", "environment", "min-score", "max-score", "has-insight-type", "order", "snapshot", "cursor", "all", "max-pages"} {
		servicesGroupedCmd.Flags().Lookup(name).Changed = false
	}
	setupAPIServer(t, handler)
	t.Cleanup(func() {
		servicesGroupedLimit = oldLimit
		servicesGroupedOffset = oldOffset
		servicesGroupedSort = oldSort
		servicesGroupedQuery = oldQuery
		servicesGroupedView = oldView
		servicesGroupedEnvironment = oldEnvironment
		servicesGroupedMinScore = oldMinScore
		servicesGroupedMaxScore = oldMaxScore
		servicesGroupedHasInsightType = oldHasInsightType
		servicesGroupedOrder = oldOrder
		servicesGroupedSnapshot = oldSnapshot
		for _, name := range []string{"sort", "query", "view", "environment", "min-score", "max-score", "has-insight-type", "order", "snapshot", "cursor", "all", "max-pages"} {
			servicesGroupedCmd.Flags().Lookup(name).Changed = false
		}
	})
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

func TestServicesGroupedServiceViewStartsSnapshot(t *testing.T) {
	setupServicesGroupedServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "service", r.URL.Query().Get("view"))
		assert.Equal(t, "true", r.URL.Query().Get("snapshot"))
		assert.Equal(t, "score", r.URL.Query().Get("sort"))
		w.Write([]byte(`{"data":[],"meta":{"next_cursor":"opaque","snapshot_expires_at":"2026-09-03T12:15:00Z"}}`))
	})
	out, _, err := executeCommand("services", "grouped", "--view", "service", "--json")
	require.NoError(t, err)
	assert.Contains(t, out, `"next_cursor":"opaque"`)
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

func TestServicesGroupedServiceIdentityFilters(t *testing.T) {
	body := groupedListResponse(`{"id":"aaa-111","name":"api-gateway","namespace":"default","environments":["production","staging"],"insights_count":5,"instrumentation_score":{"score":85},"last_seen_at":"2026-09-03T10:00:00Z"}`, 1, false)
	setupServicesGroupedServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "gateway & api", q.Get("q"))
		assert.Equal(t, "service", q.Get("view"))
		assert.Equal(t, "production", q.Get("environment"))
		assert.Equal(t, "40", q.Get("min_score"))
		assert.Equal(t, "90", q.Get("max_score"))
		assert.Equal(t, "missing-service-name", q.Get("has_insight_type"))
		assert.Equal(t, "score", q.Get("sort"))
		assert.Equal(t, "desc", q.Get("order"))
		assert.Equal(t, "true", q.Get("snapshot"))
		w.Write([]byte(body))
	})
	out, _, err := executeCommand("services", "grouped", "--query", "gateway & api", "--view", "service", "--environment", "production", "--min-score", "40", "--max-score", "90", "--has-insight-type", "missing-service-name", "--sort", "score", "--order", "desc")
	require.NoError(t, err)
	assert.Contains(t, out, "ENVIRONMENTS")
	assert.Contains(t, out, "production, staging")
	assert.Contains(t, out, "2026-09-03T10:00:00Z")
}

func TestServicesGroupedInvalidIdentityFlagsBeforeRequest(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		message string
	}{
		{"view", []string{"--view", "legacy"}, "--view"},
		{"minimum low", []string{"--view", "service", "--min-score", "-2"}, "--min-score"},
		{"maximum high", []string{"--view", "service", "--max-score", "101"}, "--max-score"},
		{"score order", []string{"--view", "service", "--min-score", "80", "--max-score", "20"}, "must not exceed"},
		{"order enum", []string{"--view", "service", "--order", "sideways"}, "--order"},
		{"empty query", []string{"--query", "   "}, "--query"},
		{"long environment", []string{"--view", "service", "--environment", strings.Repeat("x", 129)}, "--environment"},
		{"empty insight type", []string{"--view", "service", "--has-insight-type", ""}, "--has-insight-type"},
		{"identity filter without view", []string{"--environment", "production"}, "--view service"},
		{"identity sort without view", []string{"--sort", "score"}, "--view service"},
		{"legacy sort with view", []string{"--view", "service", "--sort", "name-asc"}, "requires --sort"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupServicesGroupedServer(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })
			args := append([]string{"services", "grouped"}, tt.args...)
			_, _, err := executeCommand(args...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.message)
		})
	}
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
	assert.Contains(t, out, "--query")
	assert.Contains(t, out, "--view")
	assert.Contains(t, out, "--min-score")
	assert.Contains(t, out, "--has-insight-type")
}
