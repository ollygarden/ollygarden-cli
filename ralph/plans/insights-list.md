# Plan: insights-list

## Goal
List all insights across the organization with filtering, sorting, and pagination.

## USAGE
```
ollygarden insights list [flags]
```

## Flags / Args
| Name | Type | Default | Required | Source |
|------|------|---------|----------|--------|
| `--limit` | int | 20 | no | Pagination (1-100) |
| `--offset` | int | 0 | no | Pagination (≥0) |
| `--service-id` | string (UUID) | | no | API `service_id` |
| `--status` | string | | no | Comma-separated: `active`, `archived`, `muted` |
| `--signal-type` | string | | no | `trace`, `metric`, `log` |
| `--impact` | string | | no | Comma-separated: `Critical`, `Important`, `Normal`, `Low` |
| `--date-from` | string (RFC3339) | | no | API `date_from` |
| `--date-to` | string (RFC3339) | | no | API `date_to` |
| `--sort` | string | `-created_at` | no | Prefix `+`/`-`. Fields: `created_at`, `detected_ts`, `updated_at`, `impact`, `signal_type` |

## API Endpoint
- Method: GET
- Path: `/api/v1/insights`
- Query params: `limit`, `offset`, `service_id`, `status`, `signal_type`, `impact`, `date_from`, `date_to`, `sort`
- Response: `{data: []Insight, meta: {timestamp, trace_id, total, has_more}, links}`

### Insight object (key fields for table)
| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | |
| `status` | string | `active`/`archived`/`muted` |
| `service_name` | string | Populated by API |
| `insight_type.display_name` | string | Human-readable name |
| `insight_type.impact` | string | `Critical`/`Important`/`Normal`/`Low` |
| `insight_type.signal_type` | string | `trace`/`metric`/`log` |
| `detected_ts` | RFC3339 | When detected |

## Human Output
**List**: table columns (6):

| ID | TYPE | IMPACT | SIGNAL | SERVICE | DETECTED |
|----|------|--------|--------|---------|----------|

- `ID` → `id`
- `TYPE` → `insight_type.display_name`
- `IMPACT` → `insight_type.impact`
- `SIGNAL` → `insight_type.signal_type`
- `SERVICE` → `service_name`
- `DETECTED` → `detected_ts`

This adds SERVICE column vs `services insights` (which doesn't need it since it's scoped to one service). Otherwise same structure.

## Error Code Mapping
| API Error Code | Exit Code |
|----------------|-----------|
| `INVALID_PARAMETERS` | 2 |
| `INVALID_API_KEY` | 3 |
| `INSIGHT_NOT_FOUND` | 4 |
| `RATE_LIMIT_EXCEEDED` | 5 |
| `DATABASE_ERROR` / `INTERNAL_ERROR` / `UPSTREAM_ERROR` | 6 |

## Destructive?
No.

## Files to Create/Modify
- `cmd/insights_list.go` — new file: command, flags, run function, table output
- `cmd/insights_list_test.go` — new file: table-driven tests

## Implementation Steps

1. **Create `cmd/insights_list.go`**:
   - Package-level vars for all 9 flags (`insightsListLimit`, `insightsListOffset`, `insightsListServiceID`, `insightsListStatus`, `insightsListSignalType`, `insightsListImpact`, `insightsListDateFrom`, `insightsListDateTo`, `insightsListSort`)
   - Define `insightsListItem` struct with fields: `ID`, `Status`, `ServiceName`, `InsightType *insightTypeCompact`, `DetectedTS`. Reuse existing `insightTypeCompact` from `services_insights.go`.
   - Register command under `insightsCmd` (already defined in `groups.go`)
   - `Use: "list"`, `Short: "List insights"`, `Args: cobra.NoArgs`, `RunE: runInsightsList`
   - In `init()`: register all 9 flags with defaults matching CLI.md
   - In `runInsightsList`:
     - Validate `--limit` (1-100), `--offset` (≥0)
     - Build `url.Values` — only set optional filters when non-empty
     - `c.Get(ctx, "/insights", query)`
     - Standard error handling (same pattern as `services_insights.go`)
     - JSON mode: pass through
     - Quiet mode: exit
     - Human mode: unmarshal `[]insightsListItem`, build table with 6 columns (ID, TYPE, IMPACT, SIGNAL, SERVICE, DETECTED)
     - Pagination hint when `meta.has_more`

2. **Create `cmd/insights_list_test.go`**:
   - `setupInsightsListServer` helper (same pattern as `setupServicesInsightsServer`)
   - Tests:
     - `TestInsightsListHuman` — 2 items, verify table contains all fields
     - `TestInsightsListJSON` — verify envelope passthrough
     - `TestInsightsListQuiet` — verify empty output
     - `TestInsightsListFilterFlags` — verify `service_id`, `status`, `signal_type`, `impact`, `date_from`, `date_to`, `sort` are sent as query params
     - `TestInsightsListPagination` — verify hint on stderr
     - `TestInsightsListInvalidLimit` — 0 and 101
     - `TestInsightsListInvalidOffset` — -1
     - `TestInsightsListHelp` — verify usage text and all flags
     - `TestInsightsList401` — exit code 3
     - `TestInsightsList500` — exit code 6

## Verification
- `go build ./...` → compiles
- `go test ./...` → all pass
- `go vet ./...` → no issues
- `ollygarden insights list --help` → shows correct usage/flags

## Feedback Loop
If verification fails:
1. Check compilation errors first
2. Check test failures — read test output carefully
3. If blocked after 3 attempts, mark blocked with reason

## Unresolved Questions
None — spec is clear, patterns are established.
