# Plan: webhooks-test

## Goal
Send a test event to a webhook URL to verify reachability.

## USAGE
```
ollygarden webhooks test <webhook-id>
```

## Flags / Args
| Name | Type | Default | Required | Source |
|------|------|---------|----------|--------|
| `webhook-id` | UUID (positional) | | **yes** | Webhook config ID |

No command-specific flags. Global flags only (`--json`, `--quiet`).

## API Endpoint
- Method: POST
- Path: `/api/v1/webhooks/{webhook_id}/test`
- Request body: none
- Response: 200 →
  ```json
  {
    "data": {
      "success": true,        // bool — 2xx = true
      "status_code": 200,     // int — HTTP status from webhook endpoint
      "response_body": "..."  // string — first 1KB of response
    },
    "meta": { "timestamp", "trace_id" }
  }
  ```

## Human Output
**Single resource** (key-value pairs):
- Success: `true`/`false`
- Status Code: `200`
- Response Body: truncated string (or `(empty)` if blank)

## Error Code Mapping
| API Error Code | Exit Code |
|----------------|-----------|
| `INVALID_API_KEY` | 3 |
| `WEBHOOK_NOT_FOUND` | 4 |
| `RATE_LIMIT_EXCEEDED` | 5 |
| `DATABASE_ERROR` / `INTERNAL_ERROR` / `UPSTREAM_ERROR` | 6 |

## Destructive?
No — POST test is idempotent and non-destructive.

## Files to Create/Modify
- `cmd/webhooks_test_cmd.go` — **new** — command + run function (named `_cmd` to avoid collision with Go's `_test.go` convention)
- `cmd/webhooks_test_cmd_test.go` — **new** — tests

## Implementation Steps

### 1. `cmd/webhooks_test_cmd.go`
1. Define `webhookTestResponse` struct: `Success bool`, `StatusCode int`, `ResponseBody string` with json tags `success`, `status_code`, `response_body`.
2. Define `webhooksTestCmd` cobra.Command:
   - `Use: "test <webhook-id>"`, `Short: "Test a webhook"`, `Args: cobra.ExactArgs(1)`, `RunE: runWebhooksTest`
3. `init()`: register under `webhooksCmd`.
4. `runWebhooksTest`:
   a. Extract `webhookID := args[0]`.
   b. Create formatter and client.
   c. Call `c.Post(cmd.Context(), "/webhooks/"+webhookID+"/test", nil)`.
   d. Parse via `client.ParseResponse(resp)`. Handle `*client.APIError` → `f.PrintError`.
   e. **JSON mode** → `f.PrintJSON(raw)` and return.
   f. **Quiet mode** → return nil.
   g. **Human mode** → unmarshal `data` into `webhookTestResponse`, print KV pairs:
      - `Success`: `strconv.FormatBool(tr.Success)`
      - `Status Code`: `strconv.Itoa(tr.StatusCode)`
      - `Response Body`: `tr.ResponseBody` (or `"(empty)"` if blank)

### 2. `cmd/webhooks_test_cmd_test.go`
Tests (following `webhooks_create_test.go` pattern):
1. **Success — human mode**: mock 200 with `{success:true, status_code:200, response_body:"OK"}`, verify KV output.
2. **Success — JSON mode**: mock 200, verify raw JSON envelope on stdout.
3. **Success — quiet mode**: mock 200, verify no stdout output.
4. **Success — failure result**: mock 200 with `{success:false, status_code:500, response_body:"err"}`, verify human output shows `false`.
5. **404 (webhook not found)**: mock 404, verify exit 4 + error on stderr.
6. **401 (auth error)**: mock 401, verify exit 3 + error on stderr.
7. **500 (server error)**: mock 500, verify exit 6 + error on stderr.
8. **No args**: expect cobra usage error.
9. **Help flag**: `--help` shows correct usage.

Use `setupWebhooksTestServer` helper following established pattern.

## Verification
- `go build ./...` → compiles
- `go test ./...` → all pass
- `go vet ./...` → no issues
- `ollygarden webhooks test --help` → shows correct usage

## Feedback Loop
If verification fails:
1. Check compilation errors first
2. Check test failures — read test output carefully
3. If blocked after 3 attempts, mark blocked with reason

## Unresolved Questions
None.
