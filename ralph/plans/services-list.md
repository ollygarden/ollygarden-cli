# Plan: services-list

## Goal
List paginated services for the authenticated organization.

## USAGE
```
ollygarden services list [flags]
```

## Flags / Args
| Name | Type | Default | Required | Source |
|------|------|---------|----------|--------|
| `--limit` | int | 50 | no | CLI.md §3.2 / OpenAPI `limit` (1-100) |
| `--offset` | int | 0 | no | CLI.md §3.2 / OpenAPI `offset` (≥0) |

No positional args. `cobra.NoArgs`.

## API Endpoint
- Method: `GET`
- Path: `/api/v1/services?limit=&offset=`
- Request body: none
- Response: `{ data: Service[], meta: { timestamp, trace_id, total, has_more }, links }`

### Service schema (from OpenAPI)
| Field | Type | Notes |
|-------|------|-------|
| `id` | string (UUID) | |
| `name` | string | service.name from OTel |
| `environment` | string | can be empty |
| `namespace` | string | can be empty |
| `version` | string | can be empty |
| `organization_id` | string | |
| `created_at` | date-time | |
| `updated_at` | date-time | |
| `first_seen_at` | date-time | |
| `last_seen_at` | date-time | |
| `instrumentation_score` | object \| null | `{ id, score (0-100), created_at, calculated_timestamp, calculation_window_seconds, evaluated_rule_ids }` |

## Human Output
**List**: table with 5 columns:

| ID | NAME | ENVIRONMENT | LAST SEEN | SCORE |
|----|------|-------------|-----------|-------|

- `ID`: full UUID (pipeable to `services get`)
- `NAME`: `service.name`
- `ENVIRONMENT`: empty string if absent
- `LAST SEEN`: `last_seen_at` timestamp
- `SCORE`: `instrumentation_score.score` (int 0-100), or `—` if null

## Error Code Mapping
| API Error Code | HTTP Status | Exit Code |
|----------------|-------------|-----------|
| `INVALID_PARAMETERS` | 400 | 2 |
| `INVALID_API_KEY` | 401 | 3 |
| `RATE_LIMIT_EXCEEDED` | 429 | 5 |
| `INTERNAL_ERROR` | 500 | 6 |

All mapped via existing `APIError.ExitCode()` — no new codes needed.

## Destructive?
No.

## Files to Create/Modify
- `cmd/services_list.go` — **create**: command definition, flag registration, `runServicesList`
- `cmd/services_list_test.go` — **create**: full test suite

## Implementation Steps

1. **Create `cmd/services_list.go`**:
   - Define package-scoped `servicesListLimit` and `servicesListOffset` vars (prefixed to avoid collision with future commands sharing the `cmd` package)
   - Define `servicesListCmd` cobra command: `Use: "list"`, `Short: "List services"`, `Args: cobra.NoArgs`, `RunE: runServicesList`
   - `init()`: register on `servicesCmd.AddCommand(servicesListCmd)`, register `--limit` (default 50) and `--offset` (default 0)
   - Define `serviceItem` struct for JSON unmarshalling:
     ```go
     type serviceItem struct {
         ID                   string               `json:"id"`
         Name                 string               `json:"name"`
         Environment          string               `json:"environment"`
         LastSeenAt           string               `json:"last_seen_at"`
         InstrumentationScore *serviceScoreCompact  `json:"instrumentation_score"`
     }
     type serviceScoreCompact struct {
         Score int `json:"score"`
     }
     ```
   - `runServicesList`:
     1. Validate flags: `limit` in 1-100, `offset` ≥ 0 → exit 2 on failure
     2. `c := NewClient()`, `f := output.New(...)`
     3. Build `url.Values` with `limit` and `offset`
     4. `c.Get(ctx, "/services", query)`
     5. `client.ParseResponse(resp)` → handle `*APIError` via `f.PrintError`
     6. JSON mode → `f.PrintJSON` full envelope, return
     7. Quiet mode → return nil
     8. Unmarshal `apiResp.Data` into `[]serviceItem`
     9. Build table rows: score = `fmt.Sprint(score)` or `—` if nil
     10. `f.PrintTable(headers, rows)`
     11. If `apiResp.Meta.HasMore` → `f.PrintPaginationHint(total, offset, limit)`

2. **Create `cmd/services_list_test.go`**:
   - `setupServicesServer` helper (mirrors `setupOrgServer` pattern)
   - `servicesListResponse` helper to build test JSON
   - Tests:
     - `TestServicesListHuman` — 2 services, verify table output contains names, scores
     - `TestServicesListJSON` — verify full envelope passthrough
     - `TestServicesListQuiet` — verify empty stdout
     - `TestServicesListPagination` — `has_more: true`, verify stderr hint
     - `TestServicesListNilScore` — service with null instrumentation_score, verify `—`
     - `TestServicesListFlags` — verify `limit` and `offset` sent as query params
     - `TestServicesListInvalidLimit` — limit=0 → exit 2
     - `TestServicesListInvalidOffset` — offset=-1 → exit 2
     - `TestServicesList401` — verify exit code 3
     - `TestServicesList500` — verify exit code 6
     - `TestServicesListHelp` — verify help text

## Verification
- `go build ./...` → compiles
- `go test ./...` → all pass
- `go vet ./...` → no issues
- `ollygarden services list --help` → shows correct usage/flags

## Feedback Loop
If verification fails:
1. Check compilation errors first
2. Check test failures — read test output carefully
3. If blocked after 3 attempts, mark blocked with reason
