# Plan: webhooks-deliveries-list

## Goal
List webhook deliveries for a specific webhook configuration.

## USAGE
```
ollygarden webhooks deliveries list <webhook-id> [flags]
```

## Flags / Args
| Name | Type | Default | Required | Source |
|------|------|---------|----------|--------|
| `webhook-id` | UUID (positional) | | **yes** | CLI.md §3.17 |
| `--limit` | int | 50 | no | CLI.md §3.17, OpenAPI min:1 max:100 default:50 |
| `--offset` | int | 0 | no | CLI.md §3.17, OpenAPI min:0 default:0 |

## API Endpoint
- Method: `GET`
- Path: `/api/v1/webhooks/{webhook_id}/deliveries?limit=&offset=`
- Request body: none
- Response: `{data: WebhookDelivery[], meta: {timestamp, trace_id, total, has_more}}`

### WebhookDelivery fields (from OpenAPI)
| Field | Type | Description |
|-------|------|-------------|
| `id` | uuid | Delivery ID |
| `webhook_config_id` | uuid | Parent webhook |
| `insight_id` | uuid | Triggering insight |
| `status` | enum: pending/success/failed/exhausted | Delivery status |
| `http_status_code` | int | Endpoint response code |
| `attempt_number` | int | Retry count |
| `error_message` | string | Error if failed |
| `idempotency_key` | string | Dedup key |
| `organization_id` | string | Org |
| `created_at` | datetime | Created |
| `completed_at` | datetime | Completed |

## Human Output
**List**: table columns (5):

| Column | Field | Notes |
|--------|-------|-------|
| ID | `id` | UUID |
| STATUS | `status` | pending/success/failed/exhausted |
| HTTP | `http_status_code` | int, show `—` if 0 |
| ATTEMPTS | `attempt_number` | int |
| CREATED | `created_at` | timestamp |

## Error Code Mapping
| API Error Code | Exit Code |
|----------------|-----------|
| `INVALID_API_KEY` | 3 |
| `INVALID_PARAMETERS` | 2 |
| `WEBHOOK_NOT_FOUND` | 4 |
| `RATE_LIMIT_EXCEEDED` | 5 |
| `INTERNAL_ERROR` | 6 |
| `DATABASE_ERROR` | 6 |

## Destructive?
No.

## Files to Create/Modify
- `cmd/webhooks_deliveries_list.go` — new command file
- `cmd/webhooks_deliveries_list_test.go` — new test file

## Implementation Steps
1. Create `cmd/webhooks_deliveries_list.go`:
   - Define package-level vars: `webhooksDeliveriesListLimit`, `webhooksDeliveriesListOffset`
   - Define `deliveryItem` struct with fields: ID, Status, HTTPStatusCode, AttemptNumber, CreatedAt
   - Define `webhooksDeliveriesListCmd` cobra.Command: Use `list <webhook-id>`, Short `List deliveries for a webhook`, Args `cobra.ExactArgs(1)`, RunE `runWebhooksDeliveriesList`
   - `init()`: add command to `webhooksDeliveriesCmd` (already in groups.go), register `--limit` (default 50) and `--offset` (default 0) flags
   - `runWebhooksDeliveriesList`: validate limit 1-100, offset >= 0, extract webhook-id from args[0], build query params, GET `/webhooks/{id}/deliveries`, handle errors (same pattern as webhooks_list.go), render table with 5 columns, handle pagination hint

2. Create `cmd/webhooks_deliveries_list_test.go`:
   - `setupWebhooksDeliveriesListServer` helper (save/restore apiURL, limit, offset, jsonMode, quiet)
   - `deliveryJSON` helper to build a single delivery JSON string
   - `deliveriesListResponse` helper to build response envelope
   - Tests: Human, JSON, Quiet, Pagination, Flags, InvalidLimit, InvalidOffset, MissingArg, 401, 404, 500, Help

## Verification
- `go build ./...` → compiles
- `go test ./...` → all pass
- `go vet ./...` → no issues
- `ollygarden webhooks deliveries list --help` → shows correct usage/flags

## Feedback Loop
If verification fails:
1. Check compilation errors first
2. Check test failures — read test output carefully
3. If blocked after 3 attempts, mark blocked with reason
