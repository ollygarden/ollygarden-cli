# Plan: analytics-services

## Goal
List service usage analytics (bytes, percentages, signal breakdown) for the organization.

## USAGE
```
ollygarden analytics services [flags]
```

## Flags / Args
| Name | Type | Default | Required | Source |
|------|------|---------|----------|--------|
| `--limit` | int | 50 | no | CLI.md §3.10, OpenAPI: min=1 max=100 |

No `--offset`, no `--sort`, no filters. No positional args.

## API Endpoint
- Method: GET
- Path: `/api/v1/analytics/services?limit=`
- Request body: none
- Response `data` (object, not array):
  ```json
  {
    "period_start": "2024-01-01T00:00:00Z",
    "period_end": "2024-01-08T00:00:00Z",
    "services": [
      {
        "name": "api-gateway",
        "namespace": "production",
        "environment": "prod",
        "total_bytes": 1234567890,
        "total_percent": 45.2,
        "metrics_bytes": 500000000,
        "metrics_percent": 40.5,
        "metrics_count": 1000000,
        "traces_bytes": 600000000,
        "traces_percent": 48.6,
        "traces_count": 50000,
        "logs_bytes": 134567890,
        "logs_percent": 10.9,
        "logs_count": 200000,
        "versions": {"1.0.0": "uuid", "2.0.0": "uuid"},
        "latest_version": {"id": "uuid", "version": "2.0.0"}
      }
    ]
  }
  ```

## Human Output

**Period header** (stderr, before table):
```
Period: 2024-01-01T00:00:00Z to 2024-01-08T00:00:00Z
```

**Table** (5 columns):
| Column | Source field | Notes |
|--------|------------|-------|
| NAME | `name` | |
| ENVIRONMENT | `environment` | |
| TOTAL | `total_bytes` | Human-readable bytes (e.g., "1.2 GB") |
| TOTAL % | `total_percent` | Formatted as "45.2%" |
| VERSION | `latest_version.version` | em dash if nil |

Rationale: Signal-level breakdown (metrics/traces/logs) is available via `--json`. Table focuses on the most actionable info — which services consume the most.

**No pagination hint**: Spec has no `--offset`, so no pagination flow.

## Error Code Mapping
| API Error Code | Exit Code |
|----------------|-----------|
| `INVALID_PARAMETERS` | 2 |
| `INVALID_API_KEY` | 3 |
| `RATE_LIMIT_EXCEEDED` | 5 |
| `INTERNAL_ERROR` | 6 |
| `DATABASE_ERROR` | 6 |

Standard mapping — no new error codes needed.

## Destructive?
No.

## Files to Create/Modify
- `cmd/analytics_services.go` — new file: command definition, flag, `runAnalyticsServices`, types, byte formatting helper
- `cmd/analytics_services_test.go` — new file: tests (human, JSON, quiet, flags, limit validation, 401, 500, help)

No modifications to existing files — `analyticsCmd` parent already registered in `cmd/groups.go`.

## Implementation Steps

1. **Create `cmd/analytics_services.go`**:
   - Define flag var: `analyticsServicesLimit int`
   - Define types:
     - `analyticsServicesData` — top-level data: `period_start`, `period_end`, `services []analyticsServiceItem`
     - `analyticsServiceItem` — name, namespace, environment, total_bytes (int64), total_percent (float64), latest_version (*latestVersion)
     - `latestVersion` — id, version
   - Define `analyticsServicesCmd` cobra.Command: Use="services", Short="List service analytics", Args=cobra.NoArgs, RunE=runAnalyticsServices
   - `init()`: register under `analyticsCmd`, add `--limit` flag (default 50)
   - `runAnalyticsServices`:
     - Validate `--limit` (1-100), exit 2 on invalid
     - Build query: `limit=N`
     - `c.Get(ctx, "/analytics/services", query)`
     - Parse response → handle API errors
     - JSON mode → print envelope
     - Quiet mode → return nil
     - Unmarshal `apiResp.Data` into `analyticsServicesData`
     - Print period header to stderr
     - Build table rows (5 cols: NAME, ENVIRONMENT, TOTAL, TOTAL %, VERSION)
     - `formatBytes(n int64) string` helper — local to file, converts bytes to human-readable (B, KB, MB, GB, TB)

2. **Create `cmd/analytics_services_test.go`**:
   - `setupAnalyticsServicesServer` — same pattern as `setupServicesServer`
   - `analyticsServicesResponse` helper — builds JSON with period + services array
   - Test cases:
     - `TestAnalyticsServicesHuman` — verifies table output, period header on stderr
     - `TestAnalyticsServicesJSON` — verifies raw envelope
     - `TestAnalyticsServicesQuiet` — empty stdout
     - `TestAnalyticsServicesFlags` — `--limit` sent as query param
     - `TestAnalyticsServicesInvalidLimit` — limit=0 and limit=101 rejected before network
     - `TestAnalyticsServicesNilVersion` — em dash when latest_version is null
     - `TestAnalyticsServices401` — exit code 3
     - `TestAnalyticsServices500` — exit code 6
     - `TestAnalyticsServicesHelp` — --help shows usage and --limit

## Verification
- `go build ./...` → compiles
- `go test ./...` → all pass
- `go vet ./...` → no issues
- `ollygarden analytics services --help` → shows usage with --limit flag

## Feedback Loop
If verification fails:
1. Check compilation errors first
2. Check test failures — read test output carefully
3. If blocked after 3 attempts, mark blocked with reason
