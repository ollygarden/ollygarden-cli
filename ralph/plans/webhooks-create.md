# Plan: webhooks-create

## Goal
Create a webhook via `POST /api/v1/webhooks`.

## USAGE
```
ollygarden webhooks create --name <name> --url <https-url> [flags]
```

## Flags / Args
| Name | Type | Default | Required | Source |
|------|------|---------|----------|--------|
| `--name` | string | | **yes** | Webhook name (max 255 chars) |
| `--url` | string | | **yes** | HTTPS URL for delivery |
| `--event-type` | string[] | `[]` (all) | no | Repeatable. Insight type IDs |
| `--environment` | string[] | `[]` (all) | no | Repeatable. Environments |
| `--min-severity` | string | `Low` | no | Enum: Low, Normal, Important, Critical |
| `--enabled` | bool | `false` | no | Bool flag (no value needed) |

No positional args.

## API Endpoint
- Method: POST
- Path: `/api/v1/webhooks`
- Request body:
  ```json
  {
    "name": "string (required, max 255)",
    "url": "string (required)",
    "is_enabled": "bool",
    "min_severity": "Low|Normal|Important|Critical",
    "event_types": ["string"],
    "environments": ["string"]
  }
  ```
- Response: 201 → `{data: WebhookConfig, meta}` — same schema as webhooks-get

## Human Output
**Single resource** (key-value pairs — reuse `webhookDetail` from `webhooks_get.go`):
- ID, Name, URL, Enabled, Severity, Event Types, Environments, Created, Updated

## Error Code Mapping
| API Error Code | Exit Code |
|----------------|-----------|
| `INVALID_PARAMETERS` / `MISSING_PARAMETER` / `INVALID_REQUEST` | 2 |
| `INVALID_API_KEY` | 3 |
| `RATE_LIMIT_EXCEEDED` | 5 |
| `DATABASE_ERROR` / `INTERNAL_ERROR` / `UPSTREAM_ERROR` | 6 |

## Destructive?
No — POST create is not destructive.

## Files to Create/Modify
- `cmd/webhooks_create.go` — **new** — command + run function
- `cmd/webhooks_create_test.go` — **new** — tests

## Implementation Steps

### 1. `cmd/webhooks_create.go`
1. Define package-level vars for all flags: `webhooksCreateName`, `webhooksCreateURL`, `webhooksCreateEventTypes` ([]string), `webhooksCreateEnvironments` ([]string), `webhooksCreateMinSeverity`, `webhooksCreateEnabled`.
2. Define `webhooksCreateCmd` cobra.Command:
   - `Use: "create"`, `Short: "Create a webhook"`, `Args: cobra.NoArgs`, `RunE: runWebhooksCreate`
3. `init()`: register under `webhooksCmd`, add all flags. Use `StringVar`, `StringArrayVar`, `BoolVar`. Mark `--name` and `--url` as required via `MarkFlagRequired`.
4. `runWebhooksCreate`:
   a. **Validate** — `--name` max 255 chars; `--min-severity` must be one of the 4 enum values (case-sensitive).
   b. **Build request body** — struct with JSON tags matching API `snake_case` fields. Only include `min_severity` if flag was explicitly set (default "Low" per spec); always include `is_enabled`.
   c. **Call** `c.Post(ctx, "/webhooks", body)`.
   d. **Parse** via `client.ParseResponse(resp)`. Handle `*client.APIError` → `f.PrintError`.
   e. **JSON mode** → `f.PrintJSON(raw)` and return.
   f. **Quiet mode** → return nil.
   g. **Human mode** → unmarshal into `webhookDetail` (reuse type from `webhooks_get.go`), print key-value pairs using same `joinOrAll` helper and same pairs as webhooks-get.

### 2. `cmd/webhooks_create_test.go`
Table-driven tests:
1. **Success — human mode**: mock 201, verify key-value output on stdout.
2. **Success — JSON mode**: mock 201, verify raw JSON envelope on stdout.
3. **Success — quiet mode**: mock 201, verify no stdout output.
4. **Missing --name**: expect exit 2 / usage error (cobra enforces required).
5. **Missing --url**: expect exit 2 / usage error.
6. **Invalid --min-severity**: expect exit 2 with message.
7. **--name too long (>255)**: expect exit 2 with message.
8. **Repeatable flags**: `--event-type a --event-type b` → verify request body has `["a","b"]`.
9. **Auth error (401)**: mock 401, verify exit 3 + error on stderr.
10. **Server error (500)**: mock 500, verify exit 6 + error on stderr.
11. **Help flag**: `--help` shows correct usage/flags.

Use `setupWebhooksCreateServer` helper following the pattern from `webhooks_get_test.go`.

## Verification
- `go build ./...` → compiles
- `go test ./...` → all pass
- `go vet ./...` → no issues
- `ollygarden webhooks create --help` → shows correct usage/flags

## Feedback Loop
If verification fails:
1. Check compilation errors first
2. Check test failures — read test output carefully
3. If blocked after 3 attempts, mark blocked with reason

## Unresolved Questions
None — spec is fully defined.
