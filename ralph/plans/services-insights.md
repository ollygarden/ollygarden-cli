# Plan: services-insights

## Goal
List insights for a specific service, filtered by status, with pagination.

## USAGE
```
ollygarden services insights <service-id> [flags]
```

## Flags / Args
| Name | Type | Default | Required | Source |
|------|------|---------|----------|--------|
| `service-id` | UUID (positional) | | **yes** | CLI.md §3.7 |
| `--status` | string | `active` | no | Comma-separated: `active`, `archived`, `muted` (OpenAPI enum) |
| `--limit` | int | 50 | no | 1-100 (OpenAPI min/max) |
| `--offset` | int | 0 | no | ≥0 (OpenAPI min) |

## API Endpoint
- Method: `GET`
- Path: `/api/v1/services/{id}/insights`
- Query params: `status`, `limit`, `offset`
- Response: `{ data: Insight[], meta: { timestamp, trace_id, total, has_more } }`

### `models.Insight` key fields
| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID | |
| `status` | enum | `active`, `archived`, `muted` |
| `insight_type` | object | Nested: `.display_name`, `.impact`, `.signal_type` |
| `service_name` | string | Populated for this endpoint |
| `service_environment` | string | |
| `detected_ts` | datetime | |
| `created_at` | datetime | |

## Human Output
**List**: table columns (5):

| ID | TYPE | IMPACT | SIGNAL | DETECTED |
|----|------|--------|--------|----------|
| `id` | `insight_type.display_name` | `insight_type.impact` | `insight_type.signal_type` | `detected_ts` |

Rationale: Service name/env is redundant (user already passed the service-id). Status is omitted since default filter is `active`; when mixed statuses are requested, user can use `--json`. Impact and signal type are the most actionable columns.

## Error Code Mapping
| API Error Code | Exit Code |
|----------------|-----------|
| `INVALID_API_KEY` | 3 |
| `INVALID_PARAMETERS` | 2 |
| `SERVICE_NOT_FOUND` | 4 |
| `RATE_LIMIT_EXCEEDED` | 5 |
| `INTERNAL_ERROR` / `DATABASE_ERROR` / `UPSTREAM_ERROR` | 6 |

## Destructive?
No.

## Files to Create/Modify
- `cmd/services_insights.go` — new command file
- `cmd/services_insights_test.go` — new test file

## Implementation Steps

1. **Create `cmd/services_insights.go`**:
   - Declare package-level vars: `servicesInsightsStatus` (string), `servicesInsightsLimit` (int), `servicesInsightsOffset` (int)
   - Define `insightItem` struct (only fields needed for table rendering):
     ```
     type insightItem struct {
         ID          string       `json:"id"`
         Status      string       `json:"status"`
         InsightType *insightTypeCompact `json:"insight_type"`
         DetectedTS  string       `json:"detected_ts"`
     }
     type insightTypeCompact struct {
         DisplayName string `json:"display_name"`
         Impact      string `json:"impact"`
         SignalType  string `json:"signal_type"`
     }
     ```
   - Register command in `init()`: `servicesCmd.AddCommand(servicesInsightsCmd)`
   - Set `Use: "insights <service-id>"`, `Short: "List insights for a service"`, `Args: cobra.ExactArgs(1)`, `RunE: runServicesInsights`
   - Add flags: `--status` (default `"active"`), `--limit` (default 50), `--offset` (default 0)
   - `runServicesInsights`:
     - Validate `--limit` 1-100, `--offset` ≥0
     - Build query params: `status`, `limit`, `offset`
     - GET `/services/{id}/insights`
     - Error handling: same pattern as `services_versions.go`
     - JSON mode: passthrough
     - Quiet mode: exit 0
     - Human mode: unmarshal `[]insightItem`, build table with 5 columns
     - Handle nil `insight_type` with em dash fallbacks
     - Print pagination hint when `meta.has_more`

2. **Create `cmd/services_insights_test.go`**:
   - `setupServicesInsightsServer` — same pattern as `setupServicesVersionsServer`
   - `insightJSON(id, status, displayName, impact, signalType, detectedTS)` helper
   - `insightsListResponse(insights, total, hasMore)` helper
   - Tests:
     - `TestServicesInsightsHuman` — 2 insights, verify table columns
     - `TestServicesInsightsJSON` — verify envelope passthrough
     - `TestServicesInsightsQuiet` — empty stdout
     - `TestServicesInsightsStatusFlag` — verify `?status=active,muted` sent
     - `TestServicesInsightsPagination` — `has_more: true` → stderr hint
     - `TestServicesInsightsFlags` — `--limit` and `--offset` passed to API
     - `TestServicesInsightsInvalidLimit` — 0 and 101 rejected before network
     - `TestServicesInsightsInvalidOffset` — -1 rejected before network
     - `TestServicesInsightsMissingArg` — no arg → error
     - `TestServicesInsights404` — `SERVICE_NOT_FOUND` → exit 4
     - `TestServicesInsights401` — `INVALID_API_KEY` → exit 3
     - `TestServicesInsights500` — `INTERNAL_ERROR` → exit 6
     - `TestServicesInsightsHelp` — shows usage, flags

## Verification
- `go build ./...` → compiles
- `go test ./...` → all pass
- `go vet ./...` → no issues
- `ollygarden services insights --help` → shows correct usage/flags

## Feedback Loop
If verification fails:
1. Check compilation errors first
2. Check test failures — read test output carefully
3. If blocked after 3 attempts, mark blocked with reason
