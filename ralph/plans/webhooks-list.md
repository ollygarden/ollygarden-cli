# Plan: webhooks-list

## Goal
List webhook configurations with pagination.

## USAGE
```
ollygarden webhooks list [flags]
```

## Flags / Args
| Name | Type | Default | Required | Source |
|------|------|---------|----------|--------|
| `--limit` | int | 50 | no | CLI.md §3.11, OpenAPI min:1 max:100 |
| `--offset` | int | 0 | no | CLI.md §3.11, OpenAPI min:0 |

## API Endpoint
- Method: GET
- Path: `/api/v1/webhooks?limit=&offset=`
- Request body: none
- Response: `{ data: WebhookConfig[], meta: { timestamp, trace_id, total, has_more } }`

### WebhookConfig fields
| Field | Type | Notes |
|-------|------|-------|
| `id` | UUID string | |
| `name` | string | |
| `url` | string | HTTPS URL |
| `is_enabled` | bool | |
| `min_severity` | string | Low/Normal/Important/Critical |
| `environments` | string[] | empty = all |
| `event_types` | string[] | empty = all |
| `organization_id` | string | |
| `created_at` | RFC3339 | |
| `updated_at` | RFC3339 | |

## Human Output
**List**: table columns (5):

| ID | NAME | URL | ENABLED | SEVERITY |
|----|------|-----|---------|----------|

- URL: truncate if needed (full data via `--json`)
- ENABLED: render `is_enabled` as `true`/`false`
- Pagination hint on stderr when `meta.has_more` is true

## Error Code Mapping
| API Error Code | Exit Code |
|----------------|-----------|
| `INVALID_PARAMETERS` | 2 |
| `INVALID_API_KEY` | 3 |
| `RATE_LIMIT_EXCEEDED` | 5 |
| `INTERNAL_ERROR` / `DATABASE_ERROR` | 6 |

## Destructive?
No.

## Files to Create/Modify
- `cmd/webhooks_list.go` — new file, command + run function
- `cmd/webhooks_list_test.go` — new file, full test suite

## Implementation Steps
1. Create `cmd/webhooks_list.go`:
   - Define `webhooksListLimit`, `webhooksListOffset` package vars
   - Define `webhookItem` struct (id, name, url, is_enabled, min_severity, created_at — only fields needed for table)
   - Define `webhooksListCmd` cobra command: Use=`list`, Short=`List webhooks`, Args=`cobra.NoArgs`, RunE=`runWebhooksList`
   - `init()`: register under `webhooksCmd`, add `--limit` (default 50) and `--offset` (default 0) flags
   - `runWebhooksList()`: validate flags → NewClient → build query → GET `/webhooks` → ParseResponse → JSON/quiet/human branches → PrintTable → PrintPaginationHint

2. Create `cmd/webhooks_list_test.go`:
   - `setupWebhooksListServer()` helper (save/restore globals: apiURL, limit, offset, jsonMode, quiet)
   - `webhooksListResponse()` JSON builder
   - `webhookJSON()` single-item JSON builder
   - Tests: human mode, JSON mode, quiet mode, pagination hint, flag passthrough, invalid limit, invalid offset, 401 error, 500 error, help text

Follow `cmd/services_list.go` pattern exactly.

## Verification
- `go build ./...` → compiles
- `go test ./...` → all pass
- `go vet ./...` → no issues
- `ollygarden webhooks list --help` → shows correct usage/flags

## Feedback Loop
If verification fails:
1. Check compilation errors first
2. Check test failures — read test output carefully
3. If blocked after 3 attempts, mark blocked with reason
