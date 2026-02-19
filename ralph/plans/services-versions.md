# Plan: services-versions

## Goal
List related versions of a service (same name, different version/env).

## Usage
```
ollygarden services versions <service-id> [flags]
```

## Flags / Args
| Name | Type | Default | Required | Source |
|------|------|---------|----------|--------|
| `service-id` | UUID (positional) | — | yes | CLI.md §3.6 |
| `--limit` | int | 20 | no | CLI.md §3.6, OpenAPI (1-50) |

Global `--json`, `--quiet` apply.

Note: No `--offset` for this endpoint (API doesn't support it).

## API Endpoint
- Method: `GET`
- Path: `/api/v1/services/{id}/versions?limit=`
- Query params: `limit` (int, 1-50, default 20)
- Response `data`: `[]models.Service` — same schema as `services list`

| Field | Type | Nullable |
|-------|------|----------|
| `id` | UUID | no |
| `name` | string | no |
| `version` | string | yes (empty) |
| `environment` | string | yes (empty) |
| `namespace` | string | yes (empty) |
| `organization_id` | string | no |
| `created_at` | datetime | no |
| `updated_at` | datetime | no |
| `first_seen_at` | datetime | no |
| `last_seen_at` | datetime | no |
| `instrumentation_score` | object | yes (null) |

Standard `meta` included in response.

## Human Output
**Table** (5 columns):

| ID | VERSION | ENVIRONMENT | LAST SEEN | SCORE |
|----|---------|-------------|-----------|-------|

- Reuse `serviceItem` and `serviceScoreCompact` from `services_list.go` (same response model).
- Show em dash for nil score.
- No pagination hint (no `--offset` available).

## Error Code Mapping
| API Error Code | HTTP Status | Exit Code |
|----------------|-------------|-----------|
| `INVALID_PARAMETERS` | 400 | 2 |
| `INVALID_API_KEY` | 401 | 3 |
| `SERVICE_NOT_FOUND` | 404 | 4 |
| `RATE_LIMIT_EXCEEDED` | 429 | 5 |
| `INTERNAL_ERROR` | 500 | 6 |

## Destructive?
No.

## Files to Create/Modify
- `cmd/services_versions.go` — new command file
- `cmd/services_versions_test.go` — new test file

## Implementation Steps

### 1. `cmd/services_versions.go`
1. Package var: `servicesVersionsLimit int`.
2. Define `servicesVersionsCmd` cobra command: `Use: "versions <service-id>"`, `Short: "List related service versions"`, `Args: cobra.ExactArgs(1)`, `RunE: runServicesVersions`.
3. `init()`: `servicesCmd.AddCommand(servicesVersionsCmd)`, register `--limit` flag (int, default 20, "Maximum number of versions (1-50)").
4. `runServicesVersions`:
   - Validate `--limit` (1-50), exit 2 on invalid.
   - Extract `serviceID := args[0]`.
   - Create client & formatter.
   - Build query params: `limit`.
   - `c.Get(cmd.Context(), "/services/"+serviceID+"/versions", query)`.
   - Parse response, handle errors (same pattern as `services_list.go`).
   - JSON mode: print full envelope.
   - Quiet mode: return nil.
   - Unmarshal into `[]serviceItem` (reuse from `services_list.go`).
   - Build table with headers `ID, VERSION, ENVIRONMENT, LAST SEEN, SCORE`.
   - Note: use `VERSION` column instead of `NAME` (unlike `services list`) since all versions share the same name.
   - Em dash for nil score.
   - `f.PrintTable(headers, rows)`.
   - No pagination hint.

### 2. `cmd/services_versions_test.go`
Follow `services_list_test.go` pattern:
1. `setupServicesVersionsServer` helper — set env, swap `apiURL`, reset `servicesVersionsLimit`, cleanup.
2. Reuse `svcJSON`, `itoa`, `btoa`, `intPtr` helpers (already in `services_list_test.go`, same package).
3. Helper `versionsListResponse(versions string, total int, hasMore bool) string`.
4. Tests:
   - `TestServicesVersionsHuman` — verify table with VERSION, ENVIRONMENT columns, check request path is `/api/v1/services/<id>/versions`
   - `TestServicesVersionsJSON` — verify full envelope passthrough
   - `TestServicesVersionsQuiet` — verify empty stdout
   - `TestServicesVersionsNilScore` — score null → em dash
   - `TestServicesVersionsLimit` — verify `limit` query param sent
   - `TestServicesVersionsInvalidLimit` — `--limit 0` → error, `--limit 51` → error
   - `TestServicesVersionsMissingArg` — no arg → usage error
   - `TestServicesVersions404` — SERVICE_NOT_FOUND → exit 4
   - `TestServicesVersions401` — INVALID_API_KEY → exit 3
   - `TestServicesVersions500` — INTERNAL_ERROR → exit 6
   - `TestServicesVersionsHelp` — --help shows usage and `--limit`

## Verification
- `go build ./...` → compiles
- `go test ./...` → all pass
- `go vet ./...` → no issues
- `ollygarden services versions --help` → shows correct usage/flags

## Feedback Loop
If verification fails:
1. Check compilation errors first
2. Check test failures — read test output carefully
3. If blocked after 3 attempts, mark blocked with reason
