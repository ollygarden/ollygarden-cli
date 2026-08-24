package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	roseTestExecutionID  = "11111111-1111-1111-1111-111111111111"
	roseTestRepositoryID = "22222222-2222-2222-2222-222222222222"
)

// setupRoseServer points the CLI at a stub API server and restores all
// rose command flag globals after the test, mirroring the per-command
// setup helpers used elsewhere in this package.
func setupRoseServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_key")
	oldURL := apiURL
	oldFindingsSeverity := roseFindingsListSeverity
	oldFindingsCategory := roseFindingsListCategory
	oldFindingsStatus := roseFindingsListStatus
	oldFindingsExecutionID := roseFindingsListExecutionID
	oldFindingsPage := roseFindingsListPage
	oldFindingsLimit := roseFindingsListLimit
	oldExecutionsLimit := roseExecutionsListLimit
	oldExecutionsOffset := roseExecutionsListOffset
	oldExecutionsStatus := roseExecutionsListStatus
	oldExecutionsRepositoryID := roseExecutionsListRepositoryID
	oldExecutionsType := roseExecutionsListType
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
		roseFindingsListSeverity = oldFindingsSeverity
		roseFindingsListCategory = oldFindingsCategory
		roseFindingsListStatus = oldFindingsStatus
		roseFindingsListExecutionID = oldFindingsExecutionID
		roseFindingsListPage = oldFindingsPage
		roseFindingsListLimit = oldFindingsLimit
		roseExecutionsListLimit = oldExecutionsLimit
		roseExecutionsListOffset = oldExecutionsOffset
		roseExecutionsListStatus = oldExecutionsStatus
		roseExecutionsListRepositoryID = oldExecutionsRepositoryID
		roseExecutionsListType = oldExecutionsType
		if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
			f.Changed = oldAPIURLChanged
		}
	})
	return srv
}

// roseRepositoryDetailResponse is the stub envelope shared by the
// repositories get and findings get tests.
const roseRepositoryDetailResponse = `{"data":{
	"repository":{"id":"repo-1","repo_full_name":"acme/checkout","repo_url":"https://github.com/acme/checkout","is_active":true,"vcs_provider":"github","repository_access_status":"active","last_scanned_at":"2026-08-22 02:10:44.123+00","last_scanned_commit_sha":"4f2e9c1ab34f2e9c1ab34f2e9c1ab34f2e9c1ab3","dashboard_issue_number":87,"active_findings_count":2,"finding_counts":{"critical":1,"high":1,"medium":0,"low":0,"suggestion":0}},
	"instrumentation_metadata":{"otel_present":true,"signals":["traces","metrics"],"detected_sdks":["go.opentelemetry.io/otel"],"instrumentation_types":["manual"],"summary_text":"HTTP spans and runtime metrics."},
	"findings":[
		{"id":"row-1","execution_id":"exec-1","finding_id":"otel-3f9a1c2b7d4e","severity":"critical","title":"PII logged","display_title":"PII (email) logged in span attributes","summary":"user.email exported on every request.","locations":[{"file":"internal/http/middleware.go","line":88,"description":"sets span attribute"}],"why":"Span attributes are retained long-term.","fix":"Drop or hash the attribute.","category":"Sensitive Data","checked":null,"pr_number":142,"pr_status":"open","fix_status":"pending","created_at":"2026-08-20 14:03:11.5+00","updated_at":"2026-08-21 09:47:02.9+00"},
		{"id":"row-2","execution_id":"exec-1","finding_id":"otel-8b2e4d1f0a6c","severity":"high","title":"Missing status code","display_title":null,"summary":null,"locations":null,"why":null,"fix":null,"category":null,"checked":true,"pr_number":null,"pr_status":null,"fix_status":null,"created_at":null,"updated_at":null}
	]},"meta":{"timestamp":"2026-08-22T02:20:00Z","trace_id":"trace-repo"}}`
