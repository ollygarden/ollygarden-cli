# Plan: services-search

## Goal
Search services by name with optional environment/namespace filters.

## USAGE
```
ollygarden services search [query] [flags]
ollygarden services search --query <text> [flags]
```

Both positional arg and `--query` flag accepted. Positional takes precedence.

## Flags / Args
| Name | Type | Default | Required | Source |
|------|------|---------|----------|--------|
| `[query]` (positional) | string | | yes (or `--query`) | CLI.md §3.4 |
| `--query` | string | | yes (or positional) | CLI.md §3.4 |
| `--limit` | int | 20 | no | OpenAPI: min 1, max 100 |
| `--offset` | int | 0 | no | OpenAPI: min 0 |
| `--environment` | string | | no | OpenAPI: filter by environment |
| `--namespace` | string | | no | OpenAPI: filter by namespace |

Note: no `-q` shorthand for `--query` (conflicts with global `--quiet`).

## API Endpoint
- Method: GET
- Path: `/api/v1/services/search`
- Query params: `q` (required), `limit`, `offset`, `environment`, `namespace`
- Response: `{data: []models.Service, meta: {timestamp, trace_id, total, has_more}}`

`models.Service` fields: `id`, `name`, `environment`, `namespace`, `version`, `first_seen_at`, `last_seen_at`, `created_at`, `updated_at`, `organization_id`, `instrumentation_score`.

## Human Output
**List**: table columns (5):

| ID | NAME | ENVIRONMENT | LAST SEEN | SCORE |

Same columns as `services list` — reuse `serviceItem` and `serviceScoreCompact` types from `services_list.go`.

## Error Code Mapping
| API Error Code | Exit Code |
|----------------|-----------|
| `INVALID_API_KEY` (401) | 3 |
| `INVALID_PARAMETERS` (400) | 2 |
| `MISSING_PARAMETER` (400) | 2 |
| `INTERNAL_ERROR` (500) | 6 |

## Destructive?
No.

## Files to Create/Modify
- `cmd/services_search.go` — new command file
- `cmd/services_search_test.go` — new test file

No modifications to existing files needed. Types `serviceItem` and `serviceScoreCompact` already defined in `services_list.go` are reused.

## Implementation Steps

### cmd/services_search.go

1. Declare package-level vars: `servicesSearchQuery`, `servicesSearchLimit` (default 20), `servicesSearchOffset`, `servicesSearchEnvironment`, `servicesSearchNamespace`.
2. Define `servicesSearchCmd` cobra command:
   - `Use: "search [query]"`
   - `Short: "Search services"`
   - `Args: cobra.MaximumNArgs(1)` — 0 or 1 positional arg
   - `RunE: runServicesSearch`
3. `init()`: register under `servicesCmd`, define flags `--query`, `--limit`, `--offset`, `--environment`, `--namespace`.
4. `runServicesSearch`:
   - Resolve query: if positional arg provided, use it; else use `--query` flag. If neither → return usage error.
   - Validate `--limit` (1-100), `--offset` (≥0).
   - Build `url.Values`: set `q`, `limit`, `offset`; conditionally set `environment`, `namespace`.
   - Call `c.Get(ctx, "/services/search", query)`.
   - Handle errors via same pattern as `services_list.go` / `services_grouped.go`.
   - JSON mode: marshal and print full envelope.
   - Quiet mode: return nil.
   - Human mode: unmarshal `data` into `[]serviceItem`, render table with same 5 columns as `services list`, print pagination hint if `has_more`.

### cmd/services_search_test.go

Follow `services_list_test.go` / `services_grouped_test.go` pattern:
1. `setupServicesSearchServer` helper — save/restore `servicesSearchQuery`, `servicesSearchLimit`, `servicesSearchOffset`, `servicesSearchEnvironment`, `servicesSearchNamespace` + `jsonMode`, `quiet`.
2. Reuse `svcJSON`, `intPtr`, `itoa`, `btoa` helpers from `services_list_test.go`. Create `servicesSearchResponse` helper (same pattern as `servicesListResponse`).
3. Tests:
   - `TestServicesSearchHumanPositional` — positional query, verify table output
   - `TestServicesSearchHumanFlag` — `--query` flag, verify table output
   - `TestServicesSearchJSON` — `--json`, verify envelope
   - `TestServicesSearchQuiet` — `--quiet`, verify empty stdout
   - `TestServicesSearchPagination` — `has_more=true`, verify hint on stderr
   - `TestServicesSearchNilScore` — nil score → em dash
   - `TestServicesSearchFlags` — `--limit`, `--offset`, `--environment`, `--namespace` forwarded to API
   - `TestServicesSearchQueryParam` — verify `q=` param sent to server
   - `TestServicesSearchMissingQuery` — no positional, no `--query` → error
   - `TestServicesSearchInvalidLimit` — `--limit 0` → error, no server call
   - `TestServicesSearchInvalidOffset` — `--offset -1` → error, no server call
   - `TestServicesSearch401` — exit code 3
   - `TestServicesSearch500` — exit code 6
   - `TestServicesSearchHelp` — `--help` shows usage, flags

## Verification
- `go build ./...` → compiles
- `go test ./...` → all pass
- `go vet ./...` → no issues
- `ollygarden services search --help` → shows correct usage/flags

## Feedback Loop
If verification fails:
1. Check compilation errors first
2. Check test failures — read test output carefully
3. If blocked after 3 attempts, mark blocked with reason
