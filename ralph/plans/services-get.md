# Plan: services-get

## Goal
Show details of a single service by ID (key-value output).

## Usage
```
ollygarden services get <service-id>
```

## Flags / Args
| Name | Type | Default | Required | Source |
|------|------|---------|----------|--------|
| `service-id` | UUID (positional) | — | yes | CLI.md §3.5 |

No flags. Global `--json`, `--quiet` apply.

## API Endpoint
- Method: `GET`
- Path: `/api/v1/services/{id}`
- Query params: none
- Response `data` (`models.Service`):

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

`instrumentation_score` fields: `id`, `score` (0-100), `calculated_timestamp`, `calculation_window_seconds`, `evaluated_rule_ids`, `created_at`.

Response also includes `links.insights` (string URL) and standard `meta`.

## Human Output
Key-value pairs:

```
ID:           550e8400-e29b-41d4-a716-446655440000
Name:         payment-service
Version:      v1.2.3
Environment:  production
Namespace:    backend
First Seen:   2026-01-15T10:30:00Z
Last Seen:    2026-02-18T14:22:00Z
Score:        82
```

- Omit `organization_id`, `created_at`, `updated_at` (internal/noise).
- Show `Score: —` (em dash) when `instrumentation_score` is null.
- Show `Version: —` when empty string.
- Show `Environment: —` when empty string.
- Show `Namespace: —` when empty string.

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
- `cmd/services_get.go` — new command file
- `cmd/services_get_test.go` — new test file

## Implementation Steps

### 1. `cmd/services_get.go`
1. Define `serviceDetail` struct with all `models.Service` fields (reuse `serviceScoreCompact` if it exists, or define inline).
2. Define `servicesGetCmd` cobra command: `Use: "get <service-id>"`, `Short: "Show service details"`, `Args: cobra.ExactArgs(1)`, `RunE: runServicesGet`.
3. `init()`: `servicesCmd.AddCommand(servicesGetCmd)`.
4. `runServicesGet`:
   - Extract `serviceID := args[0]`
   - Create client & formatter (same pattern as `organization.go`)
   - `c.Get(cmd.Context(), "/services/"+serviceID, nil)`
   - Parse response, handle errors (same pattern)
   - JSON mode: print full envelope
   - Quiet mode: return nil
   - Unmarshal into `serviceDetail`
   - Build `[]output.KVPair` with 8 fields (ID, Name, Version, Environment, Namespace, First Seen, Last Seen, Score)
   - Use em dash for empty/nil values
   - `f.PrintKeyValue(pairs)`

### 2. `cmd/services_get_test.go`
Follow `organization_test.go` pattern exactly:
1. `setupServicesGetServer` helper — set env, swap `apiURL`, cleanup.
2. Helper `serviceResponse(id, name, env, version, namespace string, score *int) string` to build JSON.
3. Tests:
   - `TestServicesGetHuman` — all fields present, verify key-value output
   - `TestServicesGetJSON` — verify full envelope passthrough
   - `TestServicesGetQuiet` — verify empty stdout
   - `TestServicesGetNilScore` — score null → em dash
   - `TestServicesGetEmptyOptionalFields` — empty version/env/namespace → em dash
   - `TestServicesGetMissingArg` — no arg → usage error
   - `TestServicesGet404` — SERVICE_NOT_FOUND → exit 4
   - `TestServicesGet401` — INVALID_API_KEY → exit 3
   - `TestServicesGet500` — INTERNAL_ERROR → exit 6
   - `TestServicesGetHelp` — --help shows usage

## Verification
- `go build ./...` → compiles
- `go test ./...` → all pass
- `go vet ./...` → no issues
- `ollygarden services get --help` → shows correct usage

## Feedback Loop
If verification fails:
1. Check compilation errors first
2. Check test failures — read test output carefully
3. If blocked after 3 attempts, mark blocked with reason
