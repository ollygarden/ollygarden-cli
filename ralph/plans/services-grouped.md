# Plan: services-grouped

## Goal
List services grouped by name with insight counts, version counts, and instrumentation scores.

## USAGE
```
ollygarden services grouped [flags]
```

## Flags / Args
| Name | Type | Default | Required | Source |
|------|------|---------|----------|--------|
| `--limit` | int | 50 | no | CLI.md §3.3 / OpenAPI `limit` (1-100) |
| `--offset` | int | 0 | no | CLI.md §3.3 / OpenAPI `offset` (≥0) |
| `--sort` | string | `insights-first` | no | CLI.md §3.3 / OpenAPI enum: `insights-first`, `name-asc`, `name-desc`, `created-asc`, `created-desc` |

No positional args. `cobra.NoArgs`.

## API Endpoint
- Method: `GET`
- Path: `/api/v1/services/grouped?limit=&offset=&sort=`
- Request body: none
- Response: `{ data: GroupedService[], meta: { timestamp, trace_id, total, has_more }, links: { insights } }`

### GroupedService schema (from OpenAPI)
| Field | Type | Notes |
|-------|------|-------|
| `name` | string | service.name from OTel |
| `environment` | string | |
| `namespace` | string | |
| `latest_id` | string (UUID) | most recently seen version |
| `version_count` | int | number of versions |
| `insights_count` | int | active insights for latest version |
| `instrumentation_score` | object \| omitted | `{ id, score (0-100), calculated_timestamp, calculation_window_seconds, created_at, evaluated_rule_ids }` |

## Human Output
**List**: table with 5 columns:

| NAME | ENVIRONMENT | VERSIONS | INSIGHTS | SCORE |
|------|-------------|----------|----------|-------|

- `NAME`: `name`
- `ENVIRONMENT`: `environment`
- `VERSIONS`: `version_count`
- `INSIGHTS`: `insights_count`
- `SCORE`: `instrumentation_score.score` (0-100), or `—` if omitted

Rationale: no ID column — grouped services don't have a single ID (they aggregate versions). `latest_id` is available via `--json`.

## Error Code Mapping
| API Error Code | HTTP Status | Exit Code |
|----------------|-------------|-----------|
| `INVALID_PARAMETERS` | 400 | 2 |
| `INVALID_API_KEY` / `UNAUTHORIZED` | 401 | 3 |
| `RATE_LIMIT_EXCEEDED` | 429 | 5 |
| `INTERNAL_ERROR` / `INTERNAL_SERVER_ERROR` | 500 | 6 |

All mapped via existing `APIError.ExitCode()` — no new codes needed.

## Destructive?
No.

## Files to Create/Modify
- `cmd/services_grouped.go` — **create**: command definition, flag registration, `runServicesGrouped`
- `cmd/services_grouped_test.go` — **create**: full test suite

## Implementation Steps

1. **Create `cmd/services_grouped.go`**:
   - Define package-scoped vars: `servicesGroupedLimit`, `servicesGroupedOffset`, `servicesGroupedSort`
   - Define `servicesGroupedCmd` cobra command: `Use: "grouped"`, `Short: "List services grouped by name"`, `Args: cobra.NoArgs`, `RunE: runServicesGrouped`
   - `init()`: register on `servicesCmd.AddCommand(servicesGroupedCmd)`, register flags:
     - `--limit` (default 50)
     - `--offset` (default 0)
     - `--sort` (default `insights-first`)
   - Define `groupedServiceItem` struct:
     ```go
     type groupedServiceItem struct {
         Name                 string               `json:"name"`
         Environment          string               `json:"environment"`
         Namespace            string               `json:"namespace"`
         LatestID             string               `json:"latest_id"`
         VersionCount         int                  `json:"version_count"`
         InsightsCount        int                  `json:"insights_count"`
         InstrumentationScore *serviceScoreCompact `json:"instrumentation_score"`
     }
     ```
     Reuse `serviceScoreCompact` from `services_list.go`.
   - `runServicesGrouped`:
     1. Validate: `limit` in 1-100, `offset` ≥ 0, `sort` in allowed enum → exit 2 on failure
     2. `c := NewClient()`, `f := output.New(...)`
     3. Build `url.Values`: `limit`, `offset`, `sort`
     4. `c.Get(ctx, "/services/grouped", query)`
     5. `client.ParseResponse(resp)` → handle `*APIError` via `f.PrintError`
     6. JSON mode → `f.PrintJSON` full envelope, return
     7. Quiet mode → return nil
     8. Unmarshal `apiResp.Data` into `[]groupedServiceItem`
     9. Build table rows: score = int or `—` if nil
     10. `f.PrintTable(headers, rows)`
     11. If `apiResp.Meta.HasMore` → `f.PrintPaginationHint(total, offset, limit)`

2. **Create `cmd/services_grouped_test.go`**:
   - `setupServicesGroupedServer` helper (save/restore `servicesGroupedLimit`, `servicesGroupedOffset`, `servicesGroupedSort`)
   - `groupedServiceJSON` helper to build test JSON for a single grouped service
   - `groupedListResponse` helper wrapping data in envelope
   - Tests:
     - `TestServicesGroupedHuman` — 2 groups, verify table has names, version counts, insight counts, scores
     - `TestServicesGroupedJSON` — verify full envelope passthrough
     - `TestServicesGroupedQuiet` — verify empty stdout
     - `TestServicesGroupedPagination` — `has_more: true`, verify stderr hint
     - `TestServicesGroupedNilScore` — omitted instrumentation_score → `—`
     - `TestServicesGroupedFlags` — verify `limit`, `offset`, `sort` sent as query params
     - `TestServicesGroupedSortFlag` — verify custom sort value sent correctly
     - `TestServicesGroupedInvalidLimit` — limit=0 → error
     - `TestServicesGroupedInvalidOffset` — offset=-1 → error
     - `TestServicesGroupedInvalidSort` — bad sort value → exit 2
     - `TestServicesGrouped401` — verify exit code 3
     - `TestServicesGrouped500` — verify exit code 6
     - `TestServicesGroupedHelp` — verify help text

## Verification
- `go build ./...` → compiles
- `go test ./...` → all pass
- `go vet ./...` → no issues
- `ollygarden services grouped --help` → shows correct usage/flags

## Feedback Loop
If verification fails:
1. Check compilation errors first
2. Check test failures — read test output carefully
3. If blocked after 3 attempts, mark blocked with reason
