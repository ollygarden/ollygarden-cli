# Plan: webhooks-get

## Goal
Show details of a single webhook configuration by ID.

## USAGE
```
ollygarden webhooks get <webhook-id>
```

## Flags / Args
| Name | Type | Default | Required | Source |
|------|------|---------|----------|--------|
| `webhook-id` | UUID (positional) | — | **yes** | CLI.md §3.13 |

No additional flags beyond globals (`--json`, `--quiet`).

## API Endpoint
- Method: `GET`
- Path: `/api/v1/webhooks/{webhook_id}`
- Request body: none
- Response: `models.WebhookConfig` — fields: `id`, `name`, `url`, `is_enabled`, `min_severity`, `event_types` (string[]), `environments` (string[]), `organization_id`, `created_at`, `updated_at`

## Human Output
**Single resource** — key-value pairs:
```
ID:           <uuid>
Name:         <name>
URL:          <url>
Enabled:      <true/false>
Severity:     <min_severity>
Event Types:  <comma-joined or "all">
Environments: <comma-joined or "all">
Created:      <created_at>
Updated:      <updated_at>
```

Empty arrays for `event_types`/`environments` mean "all" — display as `all`.

## Error Code Mapping
| API Error Code | Exit Code |
|----------------|-----------|
| `INVALID_API_KEY` | 3 |
| `INVALID_PARAMETERS` | 2 |
| `WEBHOOK_NOT_FOUND` | 4 |
| `RATE_LIMIT_EXCEEDED` | 5 |
| `INTERNAL_ERROR` | 6 |

## Destructive?
No.

## Files to Create/Modify
- `cmd/webhooks_get.go` — new file, command implementation
- `cmd/webhooks_get_test.go` — new file, tests

## Implementation Steps

1. **Create `cmd/webhooks_get.go`**:
   - Define `webhookDetail` struct with JSON tags matching `models.WebhookConfig` fields: `id`, `name`, `url`, `is_enabled`, `min_severity`, `event_types` ([]string), `environments` ([]string), `created_at`, `updated_at`
   - Register `webhooksGetCmd` under `webhooksCmd` with `Use: "get <webhook-id>"`, `Short: "Show webhook details"`, `Args: cobra.ExactArgs(1)`, `RunE: runWebhooksGet`
   - `runWebhooksGet`: extract `args[0]` → `webhookID`, call `c.Get("/webhooks/"+webhookID, nil)`, parse response, handle errors, JSON/quiet/human output
   - Human output: key-value pairs via `f.PrintKeyValue`. For `event_types`/`environments`, join with `, ` or show `all` if empty. Use `strconv.FormatBool` for `is_enabled`.

2. **Create `cmd/webhooks_get_test.go`**:
   - `setupWebhooksGetServer` helper (same pattern as `setupInsightsGetServer`)
   - `webhookGetResponse` helper building full JSON envelope
   - Tests:
     - `TestWebhooksGetHuman` — verify key-value output contains all fields
     - `TestWebhooksGetJSON` — verify full envelope passthrough
     - `TestWebhooksGetQuiet` — verify empty stdout
     - `TestWebhooksGetEmptyArrays` — verify "all" for empty event_types/environments
     - `TestWebhooksGetMissingArg` — verify error on no args
     - `TestWebhooksGet404` — `WEBHOOK_NOT_FOUND` → exit 4
     - `TestWebhooksGet401` — `INVALID_API_KEY` → exit 3
     - `TestWebhooksGet500` — `INTERNAL_ERROR` → exit 6
     - `TestWebhooksGetHelp` — verify help text

## Verification
- `go build ./...` → compiles
- `go test ./...` → all pass
- `go vet ./...` → no issues
- `ollygarden webhooks get --help` → shows correct usage/flags

## Feedback Loop
If verification fails:
1. Check compilation errors first
2. Check test failures — read test output carefully
3. If blocked after 3 attempts, mark blocked with reason
