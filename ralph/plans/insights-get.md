# Plan: insights-get

## Goal
Retrieve and display a single insight by ID.

## USAGE
```
ollygarden insights get <insight-id>
```

## Flags / Args
| Name | Type | Default | Required | Source |
|------|------|---------|----------|--------|
| `insight-id` | UUID (positional) | — | **yes** | CLI.md §3.9 |

No additional flags. Global `--json`, `--quiet` apply.

## API Endpoint
- Method: `GET`
- Path: `/api/v1/insights/{id}`
- Request body: none
- Response: `{ data: models.Insight, meta, links }`

### models.Insight fields
| Field | Type | Notes |
|-------|------|-------|
| `id` | uuid | |
| `status` | enum | `active`, `archived`, `muted` |
| `service_id` | uuid | |
| `service_name` | string | populated with service info |
| `service_environment` | string | |
| `service_namespace` | string | |
| `service_version` | string | |
| `insight_type` | object | nested `models.InsightType` |
| `attributes` | object | dynamic key-value |
| `trace_id` | string | OTel trace ID |
| `detected_ts` | datetime | |
| `telemetry_ts` | datetime | nullable |
| `created_at` | datetime | |
| `updated_at` | datetime | |

### models.InsightType fields
| Field | Type |
|-------|------|
| `id` | uuid |
| `name` | string |
| `display_name` | string |
| `description` | string |
| `impact` | enum (`Low`, `Normal`, `Important`, `Critical`) |
| `signal_type` | enum (`trace`, `metric`, `log`) |
| `remediation_instructions` | string |

## Human Output
**Single resource** — key-value pairs:
```
ID:           a1b2c3d4-...
Status:       active
Type:         High Error Rate
Impact:       Critical
Signal:       trace
Service:      payment-service (svc-uuid)
Environment:  production
Detected:     2026-02-19T10:00:00Z
Created:      2026-02-19T09:00:00Z
Updated:      2026-02-19T12:00:00Z
```

Rationale: Show the most actionable fields. Full data (attributes, remediation_instructions, etc.) available via `--json`.

## Error Code Mapping
| API Error Code | Exit Code |
|----------------|-----------|
| `INVALID_PARAMETERS` (400) | 2 |
| `INVALID_API_KEY` (401) | 3 |
| `INSIGHT_NOT_FOUND` (404) | 4 |
| `RATE_LIMIT_EXCEEDED` (429) | 5 |
| `INTERNAL_ERROR` / `DATABASE_ERROR` / `UPSTREAM_ERROR` (5xx) | 6 |

## Destructive?
No.

## Files to Create/Modify
- `cmd/insights_get.go` — new file: command definition, `insightDetail` struct, `runInsightsGet`
- `cmd/insights_get_test.go` — new file: tests (human, json, quiet, 404, 401, 500, missing arg, help)

## Implementation Steps
1. Create `cmd/insights_get.go`:
   - Define `insightDetail` struct matching the full Insight model (reuse `insightTypeCompact` from `services_insights.go` for the nested type).
   - Define `insightsGetCmd` cobra command: `Use: "get <insight-id>"`, `Short: "Show insight details"`, `Args: cobra.ExactArgs(1)`, `RunE: runInsightsGet`.
   - In `init()`, add to `insightsCmd`.
   - `runInsightsGet`: follows `services_get.go` pattern exactly:
     - Extract `args[0]` as insight ID
     - `c.Get(ctx, "/insights/"+id, nil)`
     - Parse response, handle errors with `PrintError`
     - JSON mode: marshal full envelope to stdout
     - Quiet mode: return nil
     - Human mode: unmarshal into `insightDetail`, build `[]output.KVPair`, call `PrintKeyValue`

2. Create `cmd/insights_get_test.go`:
   - `setupInsightsGetServer` helper (follows `setupServicesGetServer` pattern)
   - `insightGetResponse` helper to build JSON response
   - Tests: `TestInsightsGetHuman`, `TestInsightsGetJSON`, `TestInsightsGetQuiet`, `TestInsightsGetNilInsightType`, `TestInsightsGetEmptyOptionalFields`, `TestInsightsGetMissingArg`, `TestInsightsGet404`, `TestInsightsGet401`, `TestInsightsGet500`, `TestInsightsGetHelp`

## Verification
- `go build ./...` → compiles
- `go test ./...` → all pass
- `go vet ./...` → no issues
- `ollygarden insights get --help` → shows correct usage/flags

## Feedback Loop
If verification fails:
1. Check compilation errors first
2. Check test failures — read test output carefully
3. If blocked after 3 attempts, mark blocked with reason
