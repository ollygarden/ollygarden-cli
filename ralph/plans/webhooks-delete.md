# Plan: webhooks-delete

## Goal
Delete a webhook configuration with interactive confirmation (destructive operation).

## USAGE
```
ollygarden webhooks delete <webhook-id> [--confirm]
```

## Flags / Args
| Name | Type | Default | Required | Source |
|------|------|---------|----------|--------|
| `webhook-id` | UUID (positional) | — | **yes** | CLI.md §3.15 |
| `--confirm` | bool | `false` | no | CLI.md §3.15 |

## API Endpoint
- Method: `DELETE`
- Path: `/api/v1/webhooks/{webhook_id}`
- Request body: none
- Response: `204 No Content` (empty body on success)
- Errors: 401 (INVALID_API_KEY), 404 (WEBHOOK_NOT_FOUND), 500 (INTERNAL_ERROR)

## Human Output
No data output on success — `204 No Content` means nothing to print.
- Human mode: print `Deleted webhook "<name>" (id: <id>).` to stderr (informational), exit 0.
- `--json`: no output (no response body to print), exit 0.
- `--quiet`: no output, exit 0.

**Pre-delete**: GET the webhook first to obtain `name` for the confirmation prompt.

## Confirmation Flow (Destructive)
1. **TTY + no `--confirm`**: GET webhook → prompt on stderr: `Delete webhook "<name>" (id: <id>)? [y/N]: ` → read stdin → only `y`/`Y` proceeds.
2. **TTY + `--confirm`**: skip prompt, delete immediately.
3. **Non-TTY + no `--confirm`**: exit 2 with `Error: --confirm required for non-interactive webhook deletion`.
4. **`--quiet`**: does NOT suppress the confirmation prompt.

## Error Code Mapping
| API Error Code | Exit Code |
|----------------|-----------|
| `INVALID_API_KEY` | 3 |
| `WEBHOOK_NOT_FOUND` | 4 |
| `INTERNAL_ERROR` | 6 |
| `DATABASE_ERROR` | 6 |
| `RATE_LIMIT_EXCEEDED` | 5 |

## Destructive?
**Yes** — full confirmation flow per CLI_GUIDELINES.md §5.

## Dependencies
- Need TTY detection: add `golang.org/x/term` to `go.mod` (stdlib-adjacent, no vendoring issues). Use `term.IsTerminal(int(os.Stdin.Fd()))`.

## Files to Create/Modify
- `cmd/webhooks_delete.go` — new command file
- `cmd/webhooks_delete_test.go` — new test file
- `go.mod` / `go.sum` — add `golang.org/x/term` dependency

## Implementation Steps

### 1. Add `golang.org/x/term` dependency
```
go get golang.org/x/term
```

### 2. Create `cmd/webhooks_delete.go`
- Package var: `webhooksDeleteConfirm bool`
- Command: `Use: "delete <webhook-id>"`, `Short: "Delete a webhook"`, `Args: cobra.ExactArgs(1)`, `RunE: runWebhooksDelete`
- `init()`: register under `webhooksCmd`, add `--confirm` bool flag
- `runWebhooksDelete`:
  1. Parse `webhookID` from `args[0]`
  2. Create formatter `f` and client `c`
  3. **If not `--confirm`**: check TTY via `term.IsTerminal(int(os.Stdin.Fd()))`
     - Non-TTY → return error `"Error: --confirm required for non-interactive webhook deletion"` (exit 2 — cobra returns usage errors as exit 2)
     - TTY → GET `/webhooks/{id}` to fetch webhook name → prompt `Delete webhook "<name>" (id: <id>)? [y/N]: ` on stderr → read line from stdin → if not `y`/`Y`, print `Aborted.` to stderr, return nil (exit 0)
  4. Call `c.Delete(ctx, "/webhooks/"+webhookID)`
  5. Check `resp.StatusCode == 204` → success
  6. If error: parse via `client.ParseResponse` → `f.PrintError` → return error
  7. On success:
     - `--json`: no output (empty body)
     - `--quiet`: no output
     - Human: `fmt.Fprintf(stderr, "Deleted webhook %q (id: %s).\n", name, id)` — but only if we fetched the name (i.e., non-`--confirm` path). For `--confirm` path (no prior GET), just print `Deleted webhook (id: <id>).`

**Design decision**: For the `--confirm` path, still GET the webhook first to show the name in the success message. This also validates the ID before deleting. Keeps both paths consistent.

### 3. Create `cmd/webhooks_delete_test.go`
Test cases (table-driven where appropriate):

| Test | Description |
|------|-------------|
| `TestWebhooksDeleteHuman` | TTY + confirm flag → GET + DELETE → success message |
| `TestWebhooksDeleteJSON` | `--json` + `--confirm` → no stdout output |
| `TestWebhooksDeleteQuiet` | `--quiet` + `--confirm` → no output |
| `TestWebhooksDelete404` | DELETE returns 404 → exit 4 |
| `TestWebhooksDelete401` | DELETE returns 401 → exit 3 |
| `TestWebhooksDelete500` | DELETE returns 500 → exit 6 |
| `TestWebhooksDeleteHelp` | `--help` → shows usage, flags |
| `TestWebhooksDeleteNoArgs` | no args → error |

**Note on TTY tests**: `executeCommand` doesn't wire up a real TTY, so the non-`--confirm` TTY prompt path is hard to unit-test without refactoring. Tests will use `--confirm` flag. The non-TTY-without-`--confirm` error case CAN be tested since `os.Stdin` in tests is not a TTY.

**Approach for testability**: Make the TTY check and stdin reader injectable via the command. Use a package-level `var stdinIsTerminal = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }` and `var stdinReader io.Reader = os.Stdin` so tests can override them. This avoids needing to refactor the entire command infrastructure. Add test:
| `TestWebhooksDeleteNonTTYNoConfirm` | Override stdinIsTerminal → false, no `--confirm` → exit 2 error |
| `TestWebhooksDeleteTTYConfirmPromptYes` | Override stdinIsTerminal → true, override stdinReader → "y\n", no `--confirm` → GET + DELETE succeeds |
| `TestWebhooksDeleteTTYConfirmPromptNo` | Override stdinIsTerminal → true, override stdinReader → "n\n" → aborted, no DELETE |

### 4. Handle the 204 response
`client.ParseResponse` likely expects a JSON body. Since DELETE returns 204 (no body), `ParseResponse` will fail. Handle this explicitly:
```go
if resp.StatusCode == http.StatusNoContent {
    // success — no body to parse
}
```
Check response status BEFORE calling `ParseResponse`. Only call `ParseResponse` for error responses.

## Verification
- `go build ./...` → compiles
- `go test ./...` → all pass
- `go vet ./...` → no issues
- `ollygarden webhooks delete --help` → shows correct usage/flags

## Feedback Loop
If verification fails:
1. Check compilation errors first
2. Check test failures — read test output carefully
3. If blocked after 3 attempts, mark blocked with reason

## Unresolved Questions
None — all patterns established, API clear, confirmation rules explicit.
