# Plan: webhooks-update

## Goal
Update an existing webhook via `PUT /api/v1/webhooks/{webhook_id}` (partial update — only changed flags sent).

## USAGE
```
ollygarden webhooks update <webhook-id> [flags]
```

## Flags / Args
| Name | Type | Default | Required | Source |
|------|------|---------|----------|--------|
| `webhook-id` | UUID (positional) | | **yes** | Webhook config ID |
| `--name` | string | | no | New name (max 255 chars) |
| `--url` | string | | no | New HTTPS URL |
| `--event-type` | string[] | | no | Repeatable. Replaces event_types |
| `--environment` | string[] | | no | Repeatable. Replaces environments |
| `--min-severity` | string | | no | Enum: Low, Normal, Important, Critical |
| `--enabled` | bool | | no | Enable/disable (`--enabled` or `--enabled=false`) |

At least one flag must be provided — exit 2 otherwise.

## API Endpoint
- Method: PUT
- Path: `/api/v1/webhooks/{webhook_id}`
- Request body (partial — only include changed fields):
  ```json
  {
    "name": "string (max 255)",
    "url": "string",
    "is_enabled": true,
    "min_severity": "Low|Normal|Important|Critical",
    "event_types": ["string"],
    "environments": ["string"]
  }
  ```
- Response: 200 → `{data: WebhookConfig, meta}` — same schema as webhooks-get

## Human Output
**Single resource** (key-value pairs — reuse `webhookDetail` from `webhooks_get.go`):
- ID, Name, URL, Enabled, Severity, Event Types, Environments, Created, Updated

Same output format as `webhooks create` and `webhooks get`.

## Error Code Mapping
| API Error Code | Exit Code |
|----------------|-----------|
| `INVALID_PARAMETERS` / `MISSING_PARAMETER` / `INVALID_REQUEST` | 2 |
| `INVALID_API_KEY` | 3 |
| `WEBHOOK_NOT_FOUND` | 4 |
| `RATE_LIMIT_EXCEEDED` | 5 |
| `DATABASE_ERROR` / `INTERNAL_ERROR` / `UPSTREAM_ERROR` | 6 |

## Destructive?
No — `update` is not destructive per CLI_GUIDELINES.md §5.

## Files to Create/Modify
- `cmd/webhooks_update.go` — **new** — command + run function
- `cmd/webhooks_update_test.go` — **new** — tests

## Implementation Steps

### 1. `cmd/webhooks_update.go`
1. Define package-level vars for all flags: `webhooksUpdateName`, `webhooksUpdateURL`, `webhooksUpdateEventTypes` ([]string), `webhooksUpdateEnvironments` ([]string), `webhooksUpdateMinSeverity`, `webhooksUpdateEnabled`.
2. Define `webhooksUpdateCmd` cobra.Command:
   - `Use: "update <webhook-id>"`, `Short: "Update a webhook"`, `Args: cobra.ExactArgs(1)`, `RunE: runWebhooksUpdate`
3. `init()`: register under `webhooksCmd`, add all flags. Use `StringVar`, `StringArrayVar`, `BoolVar`. No `MarkFlagRequired` — all optional.
4. `runWebhooksUpdate`:
   a. Extract `webhookID := args[0]`.
   b. **Check at least one flag changed** — use `cmd.Flags().Changed(flagName)` for each of the 6 flags. If none changed → return `fmt.Errorf("Error: at least one flag is required")`.
   c. **Build request body as `map[string]any`** — only include fields whose flags were changed:
      - If `--name` changed: validate max 255 chars, add `"name"` to map.
      - If `--url` changed: add `"url"` to map.
      - If `--event-type` changed: add `"event_types"` (use value as-is; nil→`[]string{}`).
      - If `--environment` changed: add `"environments"` (same nil→empty treatment).
      - If `--min-severity` changed: validate enum, add `"min_severity"` to map.
      - If `--enabled` changed: add `"is_enabled"` to map.
   d. **Call** `c.Put(ctx, "/webhooks/"+webhookID, body)`.
   e. **Parse** via `client.ParseResponse(resp)`. Handle `*client.APIError` → `f.PrintError`.
   f. **JSON mode** → `f.PrintJSON(raw)` and return.
   g. **Quiet mode** → return nil.
   h. **Human mode** → unmarshal into `webhookDetail` (reuse type from `webhooks_get.go`), print key-value pairs using same `joinOrAll` helper and same 9 pairs as create/get.

### 2. `cmd/webhooks_update_test.go`
Use `setupWebhooksUpdateServer` helper (same pattern as `setupWebhooksCreateServer` — reset flag vars + globals in cleanup).

Tests:
1. **Human mode**: PUT with `--name` + `--enabled`, verify key-value output + correct method/path.
2. **JSON mode**: verify raw JSON envelope on stdout.
3. **Quiet mode**: verify no stdout output.
4. **Partial body — only changed flags sent**: set `--name` only, capture request body, verify only `"name"` key present (no `url`, `is_enabled`, etc.).
5. **No flags provided**: expect error containing "at least one flag".
6. **Missing positional arg**: expect error.
7. **Invalid --min-severity**: expect error.
8. **--name too long (>255)**: expect error.
9. **Repeatable flags**: `--event-type a --event-type b` → verify request body `event_types: ["a","b"]`.
10. **Auth error (401)**: mock 401, verify exit 3.
11. **Not found (404)**: mock 404, verify exit 4.
12. **Server error (500)**: mock 500, verify exit 6.
13. **Help flag**: `--help` shows correct usage/flags.

## Verification
- `go build ./...` → compiles
- `go test ./...` → all pass
- `go vet ./...` → no issues
- `ollygarden webhooks update --help` → shows correct usage/flags

## Feedback Loop
If verification fails:
1. Check compilation errors first
2. Check test failures — read test output carefully
3. If blocked after 3 attempts, mark blocked with reason

## Unresolved Questions
None.
