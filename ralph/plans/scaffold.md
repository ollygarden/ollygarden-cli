# Plan: scaffold

## Goal
Bootstrap the Go CLI project: module, entry point, root command with global flags, shared HTTP client, auth, and output formatter.

## Files to Create

### 1. `go.mod`
- Module: `github.com/ollygarden/ollygarden-cli`
- Go 1.23+
- Dependencies: `github.com/spf13/cobra`

### 2. `main.go`
- Calls `cmd.Execute()`
- Sets version via ldflags (`-X main.version=...`)
- Passes version string to `cmd.SetVersion()`

### 3. `cmd/root.go` — Root command + global flags
- `ollygarden` root command with `Use`, `Short`, `SilenceUsage: true`, `SilenceErrors: true`
- Global persistent flags:
  - `--api-url` (string, default `https://api.ollygarden.cloud`, env fallback `OLLYGARDEN_API_URL`)
  - `--json` (bool, default false)
  - `-q, --quiet` (bool, default false)
  - `--version` (handled by Cobra's built-in `Version` field)
- `PersistentPreRunE`: validate `OLLYGARDEN_API_KEY` is set → exit 3 if missing with message `Error: OLLYGARDEN_API_KEY not set. Export it: export OLLYGARDEN_API_KEY=og_sk_...`
- Register parent command groups: `services`, `insights`, `analytics`, `webhooks` (as empty group commands with `Use` and `Short` only — subcommands added later)
- Register `webhooks deliveries` as nested group under `webhooks`
- `Execute()` function: calls `rootCmd.Execute()`, maps returned errors to exit codes via `os.Exit()`

### 4. `cmd/groups.go` — Parent command groups
- `servicesCmd` — `Use: "services"`, `Short: "Manage services"`
- `insightsCmd` — `Use: "insights"`, `Short: "Manage insights"`
- `analyticsCmd` — `Use: "analytics"`, `Short: "View analytics"`
- `webhooksCmd` — `Use: "webhooks"`, `Short: "Manage webhooks"`
- `webhooksDeliveriesCmd` — `Use: "deliveries"`, `Short: "View webhook deliveries"`
- All added as children in `init()`

### 5. `internal/client/client.go` — HTTP client
- `Client` struct: `baseURL string`, `apiKey string`, `httpClient *http.Client`
- `New(baseURL, apiKey string) *Client`
- `Get(ctx, path string, query url.Values) (*http.Response, error)`
- `Post(ctx, path string, body any) (*http.Response, error)`
- `Put(ctx, path string, body any) (*http.Response, error)`
- `Delete(ctx, path string) (*http.Response, error)`
- Sets `Authorization: Bearer <key>` header on all requests
- Sets `User-Agent: ollygarden-cli/<version>`
- Base path: `/api/v1` appended to baseURL
- Response parsing helpers:
  - `APIResponse` struct: `Data json.RawMessage`, `Meta ResponseMeta`, `Links json.RawMessage`
  - `ErrorResponse` struct: `Error ErrorDetail`, `Meta ResponseMeta`
  - `ResponseMeta` struct: `Timestamp string`, `Total int`, `HasMore bool`, `TraceID string`
  - `ErrorDetail` struct: `Code string`, `Message string`, `Details map[string]string`
  - `ParseResponse(resp) (*APIResponse, error)` — reads body, checks status, returns parsed response or error
  - `ParseError(resp) (*ErrorResponse, error)` — reads error body

### 6. `internal/client/errors.go` — Error types & exit code mapping
- `APIError` type: embeds `ErrorResponse`, `StatusCode int`
- `func (e *APIError) Error() string` — returns human message
- `func (e *APIError) ExitCode() int` — maps per CLI.md §5:
  - 400 → 2
  - 401 → 3
  - 404 → 4
  - 429 → 5
  - 5xx → 6
  - other → 1
- `ExitCodeFromError(err) int` — unwraps to `*APIError` or returns 1

### 7. `internal/output/output.go` — Output formatter
- `Formatter` struct: `json bool`, `quiet bool`, `writer io.Writer` (stdout), `errWriter io.Writer` (stderr)
- `New(json, quiet bool) *Formatter`
- `PrintJSON(data json.RawMessage)` — writes raw JSON to stdout
- `PrintTable(headers []string, rows [][]string)` — writes aligned table to stdout (basic `text/tabwriter`)
- `PrintKeyValue(pairs []KVPair)` — writes `Key:  Value` pairs to stdout
- `PrintError(err error, jsonMode bool)` — writes error to stderr (human or JSON format per CLI.md §5)
- `PrintPaginationHint(total, offset, limit int)` — writes `# N more results. Use --offset X to see next page.` to stderr when `quiet` is false
- `IsQuiet() bool`

### 8. `internal/exitcode/exitcode.go` — Exit code constants
- Constants: `Success=0`, `General=1`, `Usage=2`, `Auth=3`, `NotFound=4`, `RateLimit=5`, `Server=6`

### 9. Tests

- `cmd/root_test.go`:
  - `--help` shows command groups (services, insights, analytics, webhooks)
  - `--version` prints version
  - Missing `OLLYGARDEN_API_KEY` → exit 3 with correct error message
  - `--api-url` overrides default

- `internal/client/client_test.go`:
  - Auth header set correctly
  - Base URL construction (with `/api/v1`)
  - Error response parsing
  - Exit code mapping (400→2, 401→3, 404→4, 429→5, 500→6)

- `internal/output/output_test.go`:
  - JSON mode prints raw JSON to stdout
  - Table formatting
  - Error output goes to stderr
  - Quiet mode suppresses non-essential output
  - Pagination hint formatting

## Implementation Steps

1. `go mod init github.com/ollygarden/ollygarden-cli` + `go get github.com/spf13/cobra`
2. Create `internal/exitcode/exitcode.go` — no dependencies, foundational
3. Create `internal/client/errors.go` — depends on exitcode
4. Create `internal/client/client.go` — HTTP client with auth, request methods, response parsing
5. Create `internal/output/output.go` — output formatting (table, JSON, error, pagination)
6. Create `cmd/root.go` — root command, global flags, `PersistentPreRunE` auth check, `Execute()`
7. Create `cmd/groups.go` — parent command groups (services, insights, analytics, webhooks, deliveries)
8. Create `main.go` — entry point calling `cmd.Execute()`
9. Create test files: `cmd/root_test.go`, `internal/client/client_test.go`, `internal/output/output_test.go`
10. Run `go build ./... && go test ./... && go vet ./...`

## Verification
- `go build ./...` → compiles
- `go test ./...` → all pass
- `go vet ./...` → no issues
- `ollygarden --help` → shows command groups
- `ollygarden --version` → prints version
- `ollygarden services --help` → shows "Manage services" with no subcommands yet

## Feedback Loop
If verification fails:
1. Check compilation errors first
2. Check test failures — read test output carefully
3. If blocked after 3 attempts, mark blocked with reason

## Unresolved Questions
None — spec is clear on all scaffold requirements.
