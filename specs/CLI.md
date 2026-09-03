# OllyGarden CLI Specification v1

CLI client for the OllyGarden REST API.

## USAGE

```
ollygarden [global flags] <command> <subcommand> [args] [flags]
```

## 1. Command Tree

```
ollygarden
├── auth
│   ├── login                           # save credentials for a context
│   ├── logout                          # remove a context (or all)
│   ├── status                          # show active credential, optional probe
│   ├── use-context <name>              # set current-context
│   └── list-contexts                   # list saved contexts (no keys shown)
├── update                              # install latest stable CLI release
├── version                             # show CLI version and build info
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
├── rose
│   ├── findings
│   │   ├── summary                     # GET /rose/findings/summary
│   │   ├── list                        # GET /rose/findings
│   │   └── get <repo-id> <finding-id>  # GET /rose/repositories/{id} (finding extracted client-side)
│   ├── repositories
│   │   ├── list                        # GET /rose/repositories
│   │   └── get <id>                    # GET /rose/repositories/{id}
│   └── executions
│       ├── list                        # GET /rose/executions
│       └── get <id>                    # GET /rose/executions/{id}
```

## 2. Global Flags

| Flag | Env Var | Type | Default | Description |
|---|---|---|---|---|
| *(none)* | `OLLYGARDEN_API_KEY` | string | required if no saved context | API key (env-only). Still wins over saved contexts when set. |
| `--api-url` | `OLLYGARDEN_API_URL` | string | `https://api.ollygarden.cloud` | Base URL for the API |
| `--context` | `OLLYGARDEN_CONTEXT` | string | *(none)* | Use a specific saved context for this invocation |
| `--json` | | bool | `false` | Output raw JSON (full API response envelope) |
| `-q, --quiet` | | bool | `false` | Suppress all non-essential output |
| `-h, --help` | | bool | | Show help |
| `--version` | | bool | | Print version and exit |

**Precedence**: flag > env var > built-in default.

Auth: `Authorization: Bearer <key>` header. Key format: `og_sk_{6char}_{32hex}`.

## 3. Subcommand Reference

### 3.1 `ollygarden auth login`

```bash
ollygarden auth login [flags]
```

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `--api-url`     | string | `https://api.ollygarden.cloud` | no | Inherited global flag. |
| `--context`     | string | derived from API URL host | no | Name to assign this context. Overwrites if it exists. |
| `--token-file`  | string | *(none)* | no | Read the token from this file path instead of stdin/TTY. |
| `--no-activate` | bool   | false | no | Save the context without setting it as current-context. |

Token input precedence: `--token-file` > non-TTY stdin > TTY prompt. Token shape `og_sk_[A-Za-z0-9]{6}_[a-f0-9]{32}` is enforced before any network call. The token is validated against `GET /api/v1/organization` before being persisted.

| API | `GET /api/v1/organization` (validation only) |
|---|---|

---

### 3.2 `ollygarden auth logout`

```bash
ollygarden auth logout [flags]
```

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `--context`  | string | *(none)* | no | Name of the context to remove. |
| `--all`      | bool   | false    | no | Remove every saved context. |
| `--confirm`  | bool   | false    | no | Required for `--all` in non-interactive mode. |

When the last context is removed, the config file is deleted entirely.

---

### 3.3 `ollygarden auth status`

```bash
ollygarden auth status [flags]
```

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `--no-probe` | bool | false | no | Skip the `GET /organization` validation probe. |

Exit codes: `0` if logged in (and probe succeeded if probing); `3` if no credential is configured or the probe got 401.

---

### 3.4 `ollygarden auth use-context <name>`

Sets `current-context` to the named context. Exit `4` if the name doesn't exist.

---

### 3.5 `ollygarden auth list-contexts`

```bash
ollygarden auth list-contexts
```

No additional flags. Columns: `CURRENT` (`*` marker), `NAME`, `API URL`. **Keys are never shown** — use `auth status` to see the active key.

---

### 3.6 `ollygarden organization`

```
ollygarden organization [flags]
```

No additional flags. Returns org tier, features, and instrumentation score.

| API | `GET /api/v1/organization` |
|---|---|

---

### 3.7 `ollygarden services list`

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

### 3.8 `ollygarden services grouped`

```
ollygarden services grouped [flags]
```

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `--limit` | int | 50 | no | Max groups (1-100) |
| `--offset` | int | 0 | no | Pagination offset |
| `--sort` | string | `insights-first` | no | Legacy: `insights-first`, `name-asc`, `name-desc`, `created-asc`, `created-desc`; service view: `score`, `name`, `insight_count`, `last_seen` |
| `--query` | string | | no | Filter service names (`q` in the API) |
| `--view` | string | | no | Opt into the `service` identity view |
| `--environment` | string | | no | Service view: restrict facts to one environment |
| `--min-score` | int | | no | Service view: minimum instrumentation score (0-100) |
| `--max-score` | int | | no | Service view: maximum instrumentation score (0-100) |
| `--has-insight-type` | string | | no | Service view: require an active insight of this exact type |
| `--order` | string | | no | Service view sort direction: `asc`, `desc` |

The service-identity filters, `--order`, and expanded sort fields (`score`,
`name`, `insight_count`, `last_seen`) require `--view service`. Legacy sort
fields cannot be combined with an explicitly selected service view. Minimum
score cannot exceed maximum score.

| API | `GET /api/v1/services/grouped?q=&view=&environment=&min_score=&max_score=&has_insight_type=&order=&limit=&offset=&sort=` |
|---|---|

---

### 3.9 `ollygarden services search`

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

### 3.10 `ollygarden services get`

```
ollygarden services get <service-id>
```

| Arg | Type | Required | Description |
|---|---|---|---|
| `service-id` | UUID | **yes** | Service ID |

| API | `GET /api/v1/services/{id}` |
|---|---|

---

### 3.11 `ollygarden services versions`

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

### 3.12 `ollygarden services insights`

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

### 3.13 `ollygarden insights list`

```
ollygarden insights list [flags]
```

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `--limit` | int | 20 | no | Max items (1-100) |
| `--offset` | int | 0 | no | Pagination offset |
| `--service-id` | UUID | | no | Filter by service |
| `--status` | string | | no | Comma-separated: `active`, `archived`, `muted` |
| `--insight-type` | string | | no | 1-100 comma-separated exact insight type names (each 1-128 characters) |
| `--signal-type` | string | | no | `trace`, `metric`, `log` |
| `--impact` | string | | no | Comma-separated: `Critical`, `Important`, `Normal`, `Low` |
| `--date-from` | RFC3339 | | no | Filter created_at >= |
| `--date-to` | RFC3339 | | no | Filter created_at <= |
| `--sort` | string | `-detected_ts` | no | Prefix `+`/`-` for ASC/DESC. Fields: `detected_ts`, `created_at`, `updated_at`, `impact`, `signal_type` |

| API | `GET /api/v1/insights?limit=&offset=&service_id=&status=&insight_type=&signal_type=&impact=&date_from=&date_to=&sort=` |
|---|---|

---

### 3.14 `ollygarden insights get`

```
ollygarden insights get <insight-id>
```

| Arg | Type | Required | Description |
|---|---|---|---|
| `insight-id` | UUID | **yes** | Insight ID |

| API | `GET /api/v1/insights/{id}` |
|---|---|

---

### 3.15 `ollygarden insights summary`

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

### 3.16 `ollygarden analytics services`

```
ollygarden analytics services [flags]
```

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `--limit` | int | 50 | no | Max services (1-100) |

| API | `GET /api/v1/analytics/services?limit=` |
|---|---|

---

### 3.17 `ollygarden webhooks list`

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

### 3.18 `ollygarden webhooks create`

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

### 3.19 `ollygarden webhooks get`

```
ollygarden webhooks get <webhook-id>
```

| Arg | Type | Required | Description |
|---|---|---|---|
| `webhook-id` | UUID | **yes** | Webhook config ID |

| API | `GET /api/v1/webhooks/{webhook_id}` |
|---|---|

---

### 3.20 `ollygarden webhooks update`

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

### 3.21 `ollygarden webhooks delete`

```
ollygarden webhooks delete <webhook-id> [--confirm]
```

| Arg/Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `webhook-id` | UUID | | **yes** | Webhook config ID |
| `--confirm` | bool | `false` | no | Skip interactive confirmation |

**Destructive operation** — see [Safety Rules](#7-safety-rules-for-destructive-operations).

| API | `DELETE /api/v1/webhooks/{webhook_id}` |
|---|---|

---

### 3.22 `ollygarden webhooks test`

```
ollygarden webhooks test <webhook-id>
```

| Arg | Type | Required | Description |
|---|---|---|---|
| `webhook-id` | UUID | **yes** | Webhook config ID |

| API | `POST /api/v1/webhooks/{webhook_id}/test` |
|---|---|

---

### 3.23 `ollygarden webhooks deliveries list`

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

### 3.24 `ollygarden webhooks deliveries get`

```
ollygarden webhooks deliveries get <webhook-id> <delivery-id>
```

| Arg | Type | Required | Description |
|---|---|---|---|
| `webhook-id` | UUID | **yes** | Webhook config ID |
| `delivery-id` | UUID | **yes** | Delivery ID |

| API | `GET /api/v1/webhooks/{webhook_id}/deliveries/{delivery_id}` |
|---|---|

---

### 3.25 Rose commands — shared behavior

The `rose` subtree fronts the Rose agent-server via the API's pass-through
routes (`/api/v1/rose/*`). Because the API forwards Rose responses verbatim,
these commands deviate from the rest of the CLI in two ways:

- **Pagination lives inside `data`**, not in `meta`: list responses are
  `data: {data: [...], pagination: {limit, offset, total, hasMore}}`.
  Pagination hints are derived from `data.pagination`, not `meta.has_more`.
- **Field names are mixed-case** as emitted by Rose (`executionType`,
  `repo_full_name`); `--json` passes them through unchanged.

Rose read commands are the only ones implemented so far; execution triggers
(review/fix) are intentionally out of scope for now.

---

### 3.26 `ollygarden rose findings summary`

```
ollygarden rose findings summary
```

Org-wide rollup of active findings: total plus counts faceted by severity
(`critical`, `high`, `medium`, `low`, `suggestion`) and category
(`Sensitive Data`, `Coverage & Correctness`, `Volume`, `Governance`,
`Custom`, plus a synthetic `Uncategorized`).

| API | `GET /api/v1/rose/findings/summary` |
|---|---|

---

### 3.27 `ollygarden rose findings list`

```
ollygarden rose findings list [flags]
```

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `--severity` | string | | no | Comma-separated: `critical`, `high`, `medium`, `low`, `suggestion` |
| `--category` | string | | no | Comma-separated, e.g. `"Sensitive Data,Volume"` |
| `--page` | int | 1 | no | Page number (≥1) |
| `--limit` | int | 50 | no | Results per page (1-100), sent as `page_size` |
| `--dismissed` | string | `false` | no | `false`, `true`, `all` |

The upstream endpoint paginates with `page`/`page_size` (there is no offset),
so this command exposes `--page` instead of `--offset`. The human-mode hint is
`# N more results. Use --page X to see next page.`

| API | `GET /api/v1/rose/findings?page=&page_size=&severity=&category=&dismissed=` |
|---|---|

---

### 3.28 `ollygarden rose findings get`

```
ollygarden rose findings get <repository-id> <finding-id>
```

| Arg | Type | Required | Description |
|---|---|---|---|
| `repository-id` | UUID | **yes** | Repository the finding belongs to |
| `finding-id` | string | **yes** | Finding ID (`otel-<12 hex>`) |

There is no per-finding API endpoint; full finding detail (summary, why, fix,
locations) is only embedded in the repository detail response. The CLI fetches
the repository and extracts the finding client-side; `--json` prints just that
finding wrapped in the standard `{data, meta}` envelope. A finding ID absent
from the repository exits 4 with code `FINDING_NOT_FOUND`.

| API | `GET /api/v1/rose/repositories/{repository_id}` |
|---|---|

---

### 3.29 `ollygarden rose repositories list`

```
ollygarden rose repositories list
```

Human mode flattens the installation nesting to one row per repository
(installation metadata remains available via `--json`). The endpoint does not
paginate meaningfully (single installation page), so no pagination flags.

| API | `GET /api/v1/rose/repositories` |
|---|---|

---

### 3.30 `ollygarden rose repositories get`

```
ollygarden rose repositories get <repository-id>
```

| Arg | Type | Required | Description |
|---|---|---|---|
| `repository-id` | UUID | **yes** | Repository ID |

Shows repository state, instrumentation metadata, and the active findings
table.

| API | `GET /api/v1/rose/repositories/{repository_id}` |
|---|---|

---

### 3.31 `ollygarden rose executions list`

```
ollygarden rose executions list [flags]
```

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `--limit` | int | 50 | no | Max items (1-100) |
| `--offset` | int | 0 | no | Pagination offset |
| `--status` | string | | no | `pending`, `running`, `completed`, `failed` |
| `--repository-id` | UUID | | no | Filter by repository |
| `--type` | string | | no | Comma-separated: `review`, `fix`, `instrumentation`, `deliveryhero-migrate-execute` |

| API | `GET /api/v1/rose/executions?limit=&offset=&status=&repositoryId=&executionType=` |
|---|---|

---

### 3.32 `ollygarden rose executions get`

```
ollygarden rose executions get <execution-id>
```

| Arg | Type | Required | Description |
|---|---|---|---|
| `execution-id` | UUID | **yes** | Execution ID |

Shows the execution summary plus its phase table. The event timeline and
agent activity are available via `--json`.

| API | `GET /api/v1/rose/executions/{execution_id}` |
|---|---|

---

### 3.33 `ollygarden update`

```
ollygarden update [flags]
```

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `--force` | bool | `false` | no | Reinstall the latest stable release when it is already current. |

Checks the latest stable GitHub release and exits successfully without making
changes when the running version is current or newer. When an update is
available, the command downloads the archive for the running OS and
architecture, verifies it against the release's `checksums.txt`, runs the
staged binary to verify its version, and then replaces the current executable.

The command supports Linux, macOS, and Windows on amd64 and arm64. It does not
require OllyGarden API credentials. Package-manager installations exposed
through a symlink must be upgraded with that package manager. Development
builds reporting version `dev` cannot self-update.

Human progress is written to stderr. JSON mode returns
`{data:{current_version,latest_version,executable,updated,forced,current_is_newer},meta}`
and suppresses human progress. `--quiet` suppresses successful output unless
`--json` is also set. Failures use exit code `1` and JSON error code
`UPDATE_FAILED`.

### Passive update notice

Every eligible interactive command starts one latest-release check concurrently
with normal command work. After a successful command, a completed check that
found a newer stable version prints this notice to stderr:

```text
Update Available
New version v0.3.0 is available. Run `ollygarden update`.

Changelog: https://github.com/ollygarden/ollygarden-cli/releases/tag/v0.3.0
```

The check has no cache and runs on every eligible invocation, matching Pi's
passive-check behavior. Network, HTTP, timeout, and malformed-response failures
are silent. The CLI waits at most 250 ms after successful command completion
for the concurrent check; a slow check never fails the command.

The check and notice are suppressed for `--json`, `--quiet`, non-TTY stdout,
help, `version`, `completion`, `update`, and development builds reporting
version `dev`. There is no update-check opt-out environment variable.

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
| **Pagination hint** | When `meta.has_more` is true in human mode, print `# N more results. Use --offset X to see next page.` on stderr. Rose lists read `data.pagination` instead (see §3.25); `rose findings list` hints with `--page`. |

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
| `7` | config | Local config file unreadable, malformed, or unwriteable |

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

### CLI-emitted error codes

These appear in JSON-mode error envelopes (`error.code`) for failures the CLI surfaces directly. Most are detected before any HTTP call (e.g. `NO_CREDENTIALS`, `INVALID_TOKEN_FORMAT`); `TOKEN_REJECTED` is emitted after `/organization` returns 401.

| Code | Exit | When |
|---|---|---|
| `NO_CREDENTIALS`        | 3 | No env var, no flag, no current-context |
| `INVALID_TOKEN_FORMAT`  | 2 | Token shape check failed |
| `TOKEN_REJECTED`        | 3 | `/organization` returned 401 |
| `CONTEXT_NOT_FOUND`     | 4 | A flag/env named a context that isn't in the file |
| `CONFIG_UNREADABLE`     | 7 | Config file exists but can't be read or parsed |
| `CONFIG_WRITE_FAILED`   | 7 | Atomic-rename or temp-file write failed |
| `TOKEN_FILE_NOT_FOUND`  | 2 | `--token-file PATH` doesn't exist or isn't readable |
| `CONFIRM_REQUIRED`      | 2 | `auth logout --all` in non-TTY without `--confirm` |
| `FINDING_NOT_FOUND`     | 4 | `rose findings get` — finding ID not present in the repository |
| `UPDATE_FAILED`         | 1 | CLI release discovery, verification, or executable replacement failed |

## 6. Credential Storage

Credentials are stored in a YAML file at `os.UserConfigDir()/ollygarden/config.yaml` with mode `0600`. Override the path with the `OLLYGARDEN_CONFIG` environment variable.

### File schema

```yaml
version: 1
current-context: prod
contexts:
  prod:
    api-url: https://api.ollygarden.cloud
    api-key: og_sk_xxxxxx_<32 hex>
  internal:
    api-url: https://api.internal.ollygarden.cloud
    api-key: og_sk_xxxxxx_<32 hex>
```

Writes are atomic (`config.yaml.tmp` → `fsync` → `rename`). When the last context is removed via `auth logout`, the file is deleted entirely.

### Resolution precedence

**API key:** `OLLYGARDEN_API_KEY` env > `--context NAME` > `OLLYGARDEN_CONTEXT` > `current-context` > error (`NO_CREDENTIALS`).

**API URL:** `--api-url` flag > `OLLYGARDEN_API_URL` env > selected context's `api-url` > built-in default `https://api.ollygarden.cloud`.

API key and API URL resolve independently — `--api-url=internal --context=prod` is allowed.

## 7. Safety Rules for Destructive Operations

Only `webhooks delete` is destructive in this API surface.

| Rule | Implementation |
|---|---|
| **Interactive confirmation** | When stdin is a TTY: prompt `Delete webhook "<name>" (id: <id>)? [y/N]:`. Default is No. |
| **`--confirm` flag** | Bypasses the interactive prompt. Required for non-interactive/scripted use. |
| **Non-TTY without `--confirm`** | Exit code `2`: `Error: --confirm required for non-interactive webhook deletion` |
| **`--quiet` interaction** | `--quiet` does not suppress the confirmation prompt. |

## 8. Config / Env Rules

```
Flag value  >  Environment variable  >  Built-in default
```

| Setting | Flag | Env Var | Default |
|---|---|---|---|
| API key | *(none)* | `OLLYGARDEN_API_KEY` | *(required)* |
| API URL | `--api-url` | `OLLYGARDEN_API_URL` | `https://api.ollygarden.cloud` |
| GitHub token | *(none)* | `GITHUB_TOKEN` | *(none)* — optional; raises GitHub API limits for explicit self-update discovery |

No config file for secrets (by design).

No OllyGarden config setting controls updates. `GITHUB_TOKEN`, when set, is
sent only to the `ollygarden update` command's GitHub API discovery request,
never to the passive check or release asset download hosts.

## 9. Examples

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
ollygarden insights list --status active --insight-type missing-service-name --impact Critical --date-from 2026-02-12T00:00:00Z --sort -detected_ts

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

# 10b. Find low-scoring production service identities with active insights
ollygarden services grouped --view service --environment production --max-score 50 --has-insight-type missing-service-name --sort score --order asc

# 11. Rose: findings overview, then drill into the critical ones
ollygarden rose findings summary
ollygarden rose findings list --severity critical,high --dismissed false

# 12. Rose: full detail for one finding (repository ID from the list output)
ollygarden rose findings get ddca7297-a16f-4ac4-bd31-d20c90f3cdaa otel-6576685e6e1f

# 13. Rose: recent failed executions for a repository
ollygarden rose executions list --status failed --repository-id ddca7297-a16f-4ac4-bd31-d20c90f3cdaa --limit 10

# 14. Install the latest stable CLI release when one is available
ollygarden update
```

## 10. Implementation Notes

- **Language**: Go with Cobra
- **API base path**: `/api/v1`
- **Auth token format**: `og_sk_{6char}_{32hex}`
- **Rate limit**: 60 req/min per key
- **Response envelope**: `{data, meta{timestamp, total, has_more, trace_id}, links}`
- **Error envelope**: `{error{code, message, details}, meta{timestamp, trace_id}}`
- **Types**: API response types are currently hand-defined inline in each command file. Future: generate from `https://api.ollygarden.cloud/openapi.json` via `oapi-codegen`.
