# Plan: organization

## Goal
Implement `ollygarden organization` — display org tier, features, and instrumentation score.

## USAGE
```
ollygarden organization [flags]
```

## Flags / Args
| Name | Type | Default | Required | Source |
|------|------|---------|----------|--------|
| *(none — global flags only)* | | | | CLI.md §3.1 |

## API Endpoint
- Method: `GET`
- Path: `/organization`
- Request body: none
- Response (`models.Organization`):
  - `tier` — `{ name: string, features: []string, allowed_insight_types: []string|null }`
  - `score` — `{ value: int, updated_at: string }` (nil if no score)

## Human Output
**Single resource** — key-value pairs:
```
Tier:            pro
Features:        webhooks, analytics, api_access
Insight Types:   all
Score:           82
Score Updated:   2026-02-18T12:00:00Z
```

Rules:
- `allowed_insight_types` nil/empty → display `all`
- `score` nil → display `—` for Score and Score Updated
- `features` → comma-separated

## Error Code Mapping
| API Error Code | Exit Code |
|----------------|-----------|
| `INVALID_API_KEY` (401) | 3 |
| `DATABASE_ERROR` / `INTERNAL_ERROR` (500) | 6 |

No 400/404/429 expected for this endpoint per OpenAPI spec.

## Destructive?
No.

## Files to Create/Modify
- `cmd/organization.go` — new command file
- `cmd/organization_test.go` — tests

## Implementation Steps

1. **Create `cmd/organization.go`**:
   - Define `organizationCmd` cobra.Command: `Use: "organization"`, `Short: "Show organization details"`, `Args: cobra.NoArgs`, `RunE: runOrganization`
   - `runOrganization`:
     1. Create client via `NewClient()`
     2. Create formatter via `output.New(jsonMode, quiet)`
     3. `client.Get(cmd.Context(), "/organization", nil)`
     4. `client.ParseResponse(resp)` → handle error (print via formatter, return `*client.APIError`)
     5. **JSON mode**: `formatter.PrintJSON(fullResponseBody)` — print the raw response body (full envelope) — then return
     6. **Quiet mode**: return nil (success, no output)
     7. **Human mode**: unmarshal `apiResp.Data` into a local struct, build `[]output.KVPair`, call `formatter.PrintKeyValue()`
   - Register: `rootCmd.AddCommand(organizationCmd)` in `init()`

2. **Create `cmd/organization_test.go`**:
   - Use `httptest.NewServer` to mock API responses (same pattern as `client_test.go`)
   - Test cases:
     - Happy path: human output contains tier, features, score
     - Happy path: `--json` prints full envelope
     - Happy path: `--quiet` produces no stdout
     - Score nil: displays `—`
     - Allowed insight types nil: displays `all`
     - 401 returns `*client.APIError` with exit code 3
     - 500 returns `*client.APIError` with exit code 6
     - `--help` shows usage

### JSON mode detail

For `--json`, the command must print the **full API response envelope** (not just `apiResp.Data`). This means we need the raw response body before it's parsed. Two approaches:

**Chosen approach**: Re-marshal the parsed `APIResponse` struct. Since `Data` is `json.RawMessage` and `Meta`/`Links` are typed, we can marshal the whole `APIResponse` back. This reuses `ParseResponse` (which also handles error status codes) and avoids duplicating error handling.

Implementation: after `ParseResponse` succeeds, `json.Marshal(apiResp)` → `formatter.PrintJSON()`.

### Test helper

The test needs to override the global `apiURL` and set `OLLYGARDEN_API_KEY` to point at the test server. Use:
- `t.Setenv("OLLYGARDEN_API_KEY", "test-key")`
- Set `apiURL` to `srv.URL` (the package-level var in `cmd`)
- Use `executeCommand("organization")` / `executeCommand("organization", "--json")`

## Verification
- `go build ./...` → compiles
- `go test ./...` → all pass
- `go vet ./...` → no issues
- `ollygarden organization --help` → shows correct usage

## Feedback Loop
If verification fails:
1. Check compilation errors first
2. Check test failures — read test output carefully
3. If blocked after 3 attempts, mark blocked with reason
