# Plan: webhooks-deliveries-get

## Goal
Show details of a single webhook delivery.

## USAGE
```
ollygarden webhooks deliveries get <webhook-id> <delivery-id>
```

## Flags / Args
| Name | Type | Default | Required | Source |
|------|------|---------|----------|--------|
| `webhook-id` | UUID (positional) | — | yes | CLI.md §3.18 |
| `delivery-id` | UUID (positional) | — | yes | CLI.md §3.18 |

No additional flags. Global flags (`--json`, `--quiet`) apply.

## API Endpoint
- Method: `GET`
- Path: `/api/v1/webhooks/{webhook_id}/deliveries/{delivery_id}`
- Request body: none
- Response (`data`):
  - `id` (uuid)
  - `created_at` (datetime)
  - `completed_at` (datetime, nullable)
  - `attempt_number` (int)
  - `status` (enum: `pending`, `success`, `failed`, `exhausted`)
  - `http_status_code` (int)
  - `error_message` (string, nullable)
  - `idempotency_key` (string)
  - `insight_id` (uuid)
  - `webhook_config_id` (uuid)
  - `organization_id` (string)

## Human Output
**Single resource** — key-value pairs:
```
ID:              <id>
Status:          <status>
HTTP Status:     <http_status_code> (or "—" if 0)
Attempts:        <attempt_number>
Error:           <error_message> (or "—" if null/empty)
Insight ID:      <insight_id>
Webhook ID:      <webhook_config_id>
Idempotency Key: <idempotency_key>
Created:         <created_at>
Completed:       <completed_at> (or "—" if null/empty)
```

## Error Code Mapping
| API Error Code | Exit Code |
|----------------|-----------|
| `INVALID_API_KEY` | 3 |
| `WEBHOOK_NOT_FOUND` | 4 |
| `DELIVERY_NOT_FOUND` | 4 |
| `INTERNAL_ERROR` | 6 |

## Destructive?
No.

## Files to Create/Modify
- `cmd/webhooks_deliveries_get.go` — new file: command definition + `runWebhooksDeliveriesGet`
- `cmd/webhooks_deliveries_get_test.go` — new file: table-driven tests

## Implementation Steps
1. Create `cmd/webhooks_deliveries_get.go`:
   - Define `deliveryDetail` struct with all response fields
   - Register `webhooksDeliveriesGetCmd` under `webhooksDeliveriesCmd` in `init()`
   - `Use: "get <webhook-id> <delivery-id>"`, `Short: "Show delivery details"`, `Args: cobra.ExactArgs(2)`
   - `RunE`: create client → `GET /webhooks/{wid}/deliveries/{did}` (no query params) → parse response → handle JSON/quiet/human modes
   - Human output: key-value pairs via `f.PrintKeyValue()`, show "—" for zero http_status_code and null/empty fields

2. Create `cmd/webhooks_deliveries_get_test.go`:
   - `setupWebhooksDeliveriesGetServer` helper (same pattern as list)
   - `deliveryGetResponse` helper to build single-resource JSON
   - Tests: human output, JSON mode, quiet mode, null `completed_at`/`error_message`, missing args, 401, 404 (both `WEBHOOK_NOT_FOUND` and `DELIVERY_NOT_FOUND`), 500, help text

## Verification
- `go build ./...` → compiles
- `go test ./...` → all pass
- `go vet ./...` → no issues
- `ollygarden webhooks deliveries get --help` → shows correct usage/flags

## Feedback Loop
If verification fails:
1. Check compilation errors first
2. Check test failures — read test output carefully
3. If blocked after 3 attempts, mark blocked with reason
