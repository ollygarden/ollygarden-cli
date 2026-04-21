# OllyGarden CLI Specification v1

CLI client for the OllyGarden REST API.

## USAGE

```
ollygarden [global flags] <command> <subcommand> [args] [flags]
```

## 1. Command Tree

```
ollygarden
├── organization                        # GET /organization
├── services
│   ├── list                            # GET /services
│   ├── grouped                         # GET /services/grouped
│   ├── search [query]                  # GET /services/search
│   ├── get <id>                        # GET /services/{id}
│   ├── versions <id>                   # GET /services/{id}/versions
│   └── insights <id>                   # GET /services/{id}/insights
├── insights
│   ├── list                            # GET /insights
│   ├── get <id>                        # GET /insights/{id}
│   └── summary <id>                    # GET /insights/{id}/summary
├── analytics
│   └── services                        # GET /analytics/services
└── webhooks
    ├── list                            # GET /webhooks
    ├── create                          # POST /webhooks
    ├── get <id>                        # GET /webhooks/{id}
    ├── update <id>                     # PUT /webhooks/{id}
    ├── delete <id>                     # DELETE /webhooks/{id}
    ├── test <id>                       # POST /webhooks/{id}/test
    └── deliveries
        ├── list <webhook-id>           # GET /webhooks/{id}/deliveries
        └── get <webhook-id> <did>      # GET /webhooks/{id}/deliveries/{did}
```

## 2. Global Flags

| Flag | Env Var | Type | Default | Description |
|---|---|---|---|---|
| *(none)* | `OLLYGARDEN_API_KEY` | string | **required** | API key (env-only, no flag — avoids process table/shell history leaks) |
| `--api-url` | `OLLYGARDEN_API_URL` | string | `https://api.ollygarden.cloud` | Base URL for the API |
| `--json` | | bool | `false` | Output raw JSON (full API response envelope) |
| `-q, --quiet` | | bool | `false` | Suppress all non-essential output |
| `-h, --help` | | bool | | Show help |
| `--version` | | bool | | Print version and exit |

**Precedence**: flag > env var > built-in default.

Auth: `Authorization: Bearer <key>` header. Key format: `og_sk_{6char}_{32hex}`.

## 3. Subcommand Reference

### 3.1 `ollygarden organization`

```
ollygarden organization [flags]
```

No additional flags. Returns org tier, features, and instrumentation score.

| API | `GET /api/v1/organization` |
|---|---|

---

### 3.2 `ollygarden services list`

```
ollygarden services list [flags]
```

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `--limit` | int | 50 | no | Max items (1-100) |
| `--offset` | int | 0 | no | Pagination offset |

| API | `GET /api/v1/services?limit=&offset=` |
|---|---|

---

### 3.3 `ollygarden services grouped`

```
ollygarden services grouped [flags]
```

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `--limit` | int | 50 | no | Max groups (1-100) |
| `--offset` | int | 0 | no | Pagination offset |
| `--sort` | string | `insights-first` | no | `insights-first`, `name-asc`, `name-desc`, `created-asc`, `created-desc` |

| API | `GET /api/v1/services/grouped?limit=&offset=&sort=` |
|---|---|

---

### 3.4 `ollygarden services search`

```
ollygarden services search [query] [flags]
ollygarden services search --query <text> [flags]
```

Both positional arg and `--query` flag accepted. Positional takes precedence.

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `--query` | string | | **yes** (or positional) | Search query |
| `--limit` | int | 20 | no | Max results (1-100) |
| `--offset` | int | 0 | no | Pagination offset |
| `--environment` | string | | no | Filter by environment |
| `--namespace` | string | | no | Filter by namespace |

Note: global `-q` is `--quiet`. No `-q` shorthand for `--query` to avoid ambiguity.

| API | `GET /api/v1/services/search?q=&limit=&offset=&environment=&namespace=` |
|---|---|

---

### 3.5 `ollygarden services get`

```
ollygarden services get <service-id>
```

| Arg | Type | Required | Description |
|---|---|---|---|
| `service-id` | UUID | **yes** | Service ID |

| API | `GET /api/v1/services/{id}` |
|---|---|

---

### 3.6 `ollygarden services versions`

```
ollygarden services versions <service-id> [flags]
```

| Arg/Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `service-id` | UUID | | **yes** | Service ID |
| `--limit` | int | 20 | no | Max versions (1-50) |

| API | `GET /api/v1/services/{id}/versions?limit=` |
|---|---|

---

### 3.7 `ollygarden services insights`

```
ollygarden services insights <service-id> [flags]
```

| Arg/Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `service-id` | UUID | | **yes** | Service ID |
| `--status` | string | `active` | no | Comma-separated: `active`, `archived`, `muted` |
| `--limit` | int | 50 | no | Max items (1-100) |
| `--offset` | int | 0 | no | Pagination offset |

| API | `GET /api/v1/services/{id}/insights?status=&limit=&offset=` |
|---|---|

---

### 3.8 `ollygarden insights list`

```
ollygarden insights list [flags]
```

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `--limit` | int | 20 | no | Max items (1-100) |
| `--offset` | int | 0 | no | Pagination offset |
| `--service-id` | UUID | | no | Filter by service |
| `--status` | string | | no | Comma-separated: `active`, `archived`, `muted` |
| `--signal-type` | string | | no | `trace`, `metric`, `log` |
| `--impact` | string | | no | Comma-separated: `Critical`, `Important`, `Normal`, `Low` |
| `--date-from` | RFC3339 | | no | Filter created_at >= |
| `--date-to` | RFC3339 | | no | Filter created_at <= |
| `--sort` | string | `-created_at` | no | Prefix `+`/`-` for ASC/DESC. Fields: `created_at`, `detected_ts`, `updated_at`, `impact`, `signal_type` |

| API | `GET /api/v1/insights?limit=&offset=&service_id=&status=&signal_type=&impact=&date_from=&date_to=&sort=` |
|---|---|

---

### 3.9 `ollygarden insights get`

```
ollygarden insights get <insight-id>
```

| Arg | Type | Required | Description |
|---|---|---|---|
| `insight-id` | UUID | **yes** | Insight ID |

| API | `GET /api/v1/insights/{id}` |
|---|---|

---

### 3.10 `ollygarden insights summary`

```bash
ollygarden insights summary <insight-id>
```

| Arg | Type | Required | Description |
|---|---|---|---|
| `insight-id` | UUID | **yes** | Insight ID |

Returns an AI-generated summary for the insight. The summary includes contextual explanation of why the insight matters, its specific impact, and a recommended next step. Summaries are cached; on cache miss, a new one is generated via Lotus.

| API | `GET /api/v1/insights/{id}/summary` |
|---|---|

---

### 3.11 `ollygarden analytics services`

```
ollygarden analytics services [flags]
```

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `--limit` | int | 50 | no | Max services (1-100) |

| API | `GET /api/v1/analytics/services?limit=` |
|---|---|

---

### 3.12 `ollygarden webhooks list`

```
ollygarden webhooks list [flags]
```

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `--limit` | int | 50 | no | Max items (1-100) |
| `--offset` | int | 0 | no | Pagination offset |

| API | `GET /api/v1/webhooks?limit=&offset=` |
|---|---|

---

### 3.13 `ollygarden webhooks create`

```
ollygarden webhooks create --name <name> --url <https-url> [flags]
```

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `--name` | string | | **yes** | Webhook name (max 255 chars) |
| `--url` | string | | **yes** | HTTPS URL for delivery |
| `--event-type` | string[] | `[]` (all) | no | Repeatable. Insight type IDs to subscribe to |
| `--environment` | string[] | `[]` (all) | no | Repeatable. Environments to subscribe to |
| `--min-severity` | string | `Low` | no | `Low`, `Normal`, `Important`, `Critical` |
| `--enabled` | bool | `false` | no | Disabled by default; pass `--enabled` to activate immediately |

| API | `POST /api/v1/webhooks` (JSON body) |
|---|---|

---

### 3.14 `ollygarden webhooks get`

```
ollygarden webhooks get <webhook-id>
```

| Arg | Type | Required | Description |
|---|---|---|---|
| `webhook-id` | UUID | **yes** | Webhook config ID |

| API | `GET /api/v1/webhooks/{webhook_id}` |
|---|---|

---

### 3.15 `ollygarden webhooks update`

```
ollygarden webhooks update <webhook-id> [flags]
```

All flags optional (partial update). Only provided flags are sent in the request body.

| Arg/Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `webhook-id` | UUID | | **yes** | Webhook config ID |
| `--name` | string | | no | New name |
| `--url` | string | | no | New HTTPS URL |
| `--event-type` | string[] | | no | Repeatable. Replaces event_types |
| `--environment` | string[] | | no | Repeatable. Replaces environments |
| `--min-severity` | string | | no | `Low`, `Normal`, `Important`, `Critical` |
| `--enabled` | bool | | no | Enable/disable |

| API | `PUT /api/v1/webhooks/{webhook_id}` (JSON body, partial) |
|---|---|

---

### 3.16 `ollygarden webhooks delete`

```
ollygarden webhooks delete <webhook-id> [--confirm]
```

| Arg/Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `webhook-id` | UUID | | **yes** | Webhook config ID |
| `--confirm` | bool | `false` | no | Skip interactive confirmation |

**Destructive operation** — see [Safety Rules](#6-safety-rules-for-destructive-operations).

| API | `DELETE /api/v1/webhooks/{webhook_id}` |
|---|---|

---

### 3.17 `ollygarden webhooks test`

```
ollygarden webhooks test <webhook-id>
```

| Arg | Type | Required | Description |
|---|---|---|---|
| `webhook-id` | UUID | **yes** | Webhook config ID |

| API | `POST /api/v1/webhooks/{webhook_id}/test` |
|---|---|

---

### 3.18 `ollygarden webhooks deliveries list`

```
ollygarden webhooks deliveries list <webhook-id> [flags]
```

| Arg/Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `webhook-id` | UUID | | **yes** | Webhook config ID |
| `--limit` | int | 50 | no | Max items (1-100) |
| `--offset` | int | 0 | no | Pagination offset |

| API | `GET /api/v1/webhooks/{webhook_id}/deliveries?limit=&offset=` |
|---|---|

---

### 3.19 `ollygarden webhooks deliveries get`

```
ollygarden webhooks deliveries get <webhook-id> <delivery-id>
```

| Arg | Type | Required | Description |
|---|---|---|---|
| `webhook-id` | UUID | **yes** | Webhook config ID |
| `delivery-id` | UUID | **yes** | Delivery ID |

| API | `GET /api/v1/webhooks/{webhook_id}/deliveries/{delivery_id}` |
|---|---|

## 4. I/O Contract

| Rule | Behavior |
|---|---|
| **Human mode** (default, TTY) | Formatted tables for lists, key-value pairs for single resources. Colors when `NO_COLOR` is not set. |
| **`--json`** | Prints the full API response envelope (`{data, meta, links}`) to stdout. No colors, no table formatting. |
| **`--quiet`** | Suppress informational messages on stderr. On success: exit 0 with no output (unless `--json`). |
| **stdout** | Data output only (tables or JSON). |
| **stderr** | Diagnostics, errors, progress messages, confirmation prompts. |
| **TTY detection** | Auto-detect via `isatty(stdout)`. Non-TTY disables colors and table truncation. Prompts only when stdin is TTY. |
| **`NO_COLOR`** | Respected. When set, disable all ANSI color codes regardless of TTY. |
| **Pagination hint** | When `meta.has_more` is true in human mode, print `# N more results. Use --offset X to see next page.` on stderr. |

## 5. Exit Codes

| Exit Code | Meaning | When |
|---|---|---|
| `0` | Success | Command completed successfully |
| `1` | General error | Unclassified failures, internal errors |
| `2` | Usage error | Bad flags, missing required args, invalid parameter values |
| `3` | Auth error | Missing/invalid/expired API key |
| `4` | Not found | Resource not found (404) |
| `5` | Rate limited | `RATE_LIMIT_EXCEEDED` (60 req/min per key) |
| `6` | Server error | API returned 5xx |

### API Error Code Mapping

| API Error Code | HTTP Status | Exit Code |
|---|---|---|
| `INVALID_API_KEY` | 401 | 3 |
| `INVALID_PARAMETERS` | 400 | 2 |
| `MISSING_PARAMETER` | 400 | 2 |
| `INVALID_REQUEST` | 400 | 2 |
| `SERVICE_NOT_FOUND` | 404 | 4 |
| `INSIGHT_NOT_FOUND` | 404 | 4 |
| `WEBHOOK_NOT_FOUND` | 404 | 4 |
| `DELIVERY_NOT_FOUND` | 404 | 4 |
| `RATE_LIMIT_EXCEEDED` | 429 | 5 |
| `DATABASE_ERROR` | 500 | 6 |
| `INTERNAL_ERROR` | 500 | 6 |
| `UPSTREAM_ERROR` | 502 | 6 |
| `SERVICE_UNAVAILABLE` | 503 | 6 |

### Error Output

**Human mode** (stderr):
```
Error: <human message>
```

**`--json` mode** (stderr):
```json
{"error":{"code":"SERVICE_NOT_FOUND","message":"Service not found"},"meta":{"timestamp":"...","trace_id":"..."}}
```

Missing API key (exit 3):
```
Error: OLLYGARDEN_API_KEY not set. Export it: export OLLYGARDEN_API_KEY=og_sk_...
```

## 6. Safety Rules for Destructive Operations

Only `webhooks delete` is destructive in this API surface.

| Rule | Implementation |
|---|---|
| **Interactive confirmation** | When stdin is a TTY: prompt `Delete webhook "<name>" (id: <id>)? [y/N]:`. Default is No. |
| **`--confirm` flag** | Bypasses the interactive prompt. Required for non-interactive/scripted use. |
| **Non-TTY without `--confirm`** | Exit code `2`: `Error: --confirm required for non-interactive webhook deletion` |
| **`--quiet` interaction** | `--quiet` does not suppress the confirmation prompt. |

## 7. Config / Env Rules

```
Flag value  >  Environment variable  >  Built-in default
```

| Setting | Flag | Env Var | Default |
|---|---|---|---|
| API key | *(none)* | `OLLYGARDEN_API_KEY` | *(required)* |
| API URL | `--api-url` | `OLLYGARDEN_API_URL` | `https://api.ollygarden.cloud` |

No config file for secrets (by design).

## 8. Examples

```bash
# 1. Check org tier and instrumentation score
ollygarden organization

# 2. List first 10 services
ollygarden services list --limit 10

# 3. Search services in production, output as JSON
ollygarden services search payment --environment production --json

# 4. Get a specific service and pipe to jq for score
ollygarden services get 550e8400-e29b-41d4-a716-446655440000 --json | jq '.data.instrumentation_score.score'

# 5. List critical active insights from the last 7 days
ollygarden insights list --status active --impact Critical --date-from 2026-02-12T00:00:00Z --sort -detected_ts

# 6. Get insight details and extract remediation instructions
ollygarden insights get a1b2c3d4-5678-90ab-cdef-111111111111 --json | jq '.data.insight_type.remediation_instructions'

# 6b. Get AI-generated summary for an insight
ollygarden insights summary a1b2c3d4-5678-90ab-cdef-111111111111

# 7. Create a webhook (disabled by default), test it, then enable
ollygarden webhooks create \
  --name "PagerDuty Prod Critical" \
  --url "https://events.pagerduty.com/integration/abc/enqueue" \
  --min-severity Critical \
  --environment production
ollygarden webhooks test <id>
ollygarden webhooks update <id> --enabled

# 8. Delete a webhook non-interactively (scripted)
ollygarden webhooks delete d4e5f6a7-8901-2345-6789-abcdef012345 --confirm

# 9. Check recent deliveries
ollygarden webhooks deliveries list <id> --limit 5

# 10. List all services grouped, sorted by name, extract names with jq
ollygarden services grouped --sort name-asc --json | jq '.data[].name'
```

## 9. Implementation Notes

- **Language**: Go with Cobra (matches existing olive submodule)
- **API base path**: `/api/v1`
- **Auth token format**: `og_sk_{6char}_{32hex}`
- **Rate limit**: 60 req/min per key
- **Response envelope**: `{data, meta{timestamp, total, has_more, trace_id}, links}`
- **Error envelope**: `{error{code, message, details}, meta{timestamp, trace_id}}`
- **Types**: API response types are currently hand-defined inline in each command file. Future: generate from `olive/docs/openapi.json` via `oapi-codegen`.
