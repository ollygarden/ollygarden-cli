# CLI Extension Guidelines

How to extend the `ollygarden` CLI when the OpenAPI spec changes. All patterns reference [`CLI.md`](./CLI.md).

## 1. Mapping OpenAPI Endpoints to Commands

### Naming

| API Pattern | CLI Command | Style |
|---|---|---|
| `GET /resources` | `ollygarden resources list` | Noun-first, `list` verb |
| `GET /resources/{id}` | `ollygarden resources get <id>` | `get` + positional UUID |
| `POST /resources` | `ollygarden resources create` | `create` verb |
| `PUT /resources/{id}` | `ollygarden resources update <id>` | `update` + positional UUID |
| `DELETE /resources/{id}` | `ollygarden resources delete <id>` | `delete` + positional UUID |
| `POST /resources/{id}/action` | `ollygarden resources action <id>` | Action as verb |
| `GET /resources/{id}/children` | `ollygarden resources children <id>` | Sub-resource as subcommand |

### Nesting Rules

- **Max depth**: 3 levels (`ollygarden <noun> <verb>` or `ollygarden <noun> <sub-noun> <verb>`)
- **Sub-resources** with their own CRUD get a nested command group (e.g., `webhooks deliveries list`)
- **Single-endpoint resources** (like `organization`) are a direct command, no `get` verb needed

### Single-Resource Endpoints

If an API path has only one GET (e.g., `GET /organization`), map it directly:
```
ollygarden organization    # not "ollygarden organization get"
```

## 2. Flag Design Rules

### Pagination

All list endpoints must support:

| Flag | Type | Default | Constraint |
|---|---|---|---|
| `--limit` | int | 20-50 (match API default) | 1-100 |
| `--offset` | int | 0 | >= 0 |

### Filters

- Map each API query parameter to a `--kebab-case` flag
- API `snake_case` params become CLI `--kebab-case` flags (e.g., `signal_type` -> `--signal-type`)
- Enum params: validate client-side before sending
- Multi-value params: use comma-separated strings (e.g., `--status active,muted`)
- Date params: accept RFC3339 format

### Sort Flags

```
--sort <field>          # ascending
--sort -<field>         # descending (prefix -)
--sort +<field>         # ascending (explicit)
```

### Positional Args

- **Resource IDs** are always positional: `ollygarden resources get <id>`
- **Search queries** accept both positional and `--query` flag
- Max 2 positional args per command (e.g., `webhooks deliveries get <webhook-id> <delivery-id>`)

### Repeatable Flags

For array API fields, use repeatable flags:
```bash
ollygarden webhooks create --event-type foo --event-type bar
```

### Short Flags

- `-q` is reserved globally for `--quiet`
- `-h` is reserved globally for `--help`
- Only add short flags for frequently-used flags, avoid conflicts with global shorts

## 3. Output Format Rules

### Human Mode (TTY, default)

**List commands**: tabular output with these rules:
- Max 5-6 columns to avoid wrapping
- Pick the most useful fields; full data is available via `--json`
- Truncate long strings (e.g., URLs) with `...`
- Right-align numeric columns
- Use color for status/severity indicators (respect `NO_COLOR`)

**Single-resource commands**: key-value pairs:
```
ID:        550e8400-...
Name:      my-service
Status:    active
Created:   2026-01-15T10:30:00Z
```

### JSON Mode (`--json`)

- Print the **full API response envelope** to stdout: `{data, meta, links}`
- No transformation, no field selection — pass through exactly what the API returns
- Errors go to stderr as JSON: `{error, meta}`

### Quiet Mode (`--quiet`)

- Suppress informational messages on stderr
- Success: exit 0, no output (unless `--json` is also set)
- Errors still print to stderr

## 4. Error Handling Rules

### HTTP Status to Exit Code

| HTTP Status | Exit Code | Meaning |
|---|---|---|
| 400 | 2 | Usage/validation error |
| 401 | 3 | Auth error |
| 404 | 4 | Not found |
| 429 | 5 | Rate limited |
| 5xx | 6 | Server error |
| Network/timeout | 1 | General error |

### Adding New Error Codes

When the API adds a new error code:

1. Check its HTTP status
2. Map to the corresponding exit code per the table above
3. If it doesn't fit existing categories, use exit code `1` (general error)
4. Do NOT add new exit codes without updating `CLI.md`

### Error Output

- Human mode: `Error: <message>` on stderr
- JSON mode: full error envelope on stderr
- Always include the API's `trace_id` in error output for debugging

## 5. Destructive Operation Safety

Any command that **deletes** or **irreversibly modifies** data must follow these rules:

| Rule | Implementation |
|---|---|
| Interactive confirmation | Prompt when stdin is TTY. Default answer is No. |
| `--confirm` flag | Required for non-interactive/CI use |
| Non-TTY without `--confirm` | Exit code 2 with error message |
| `--quiet` | Does NOT suppress confirmation prompts |

### Prompt format

```
<Action> <resource-type> "<name>" (id: <id>)? [y/N]:
```

### Which operations are destructive?

- `DELETE` methods
- Any `PUT`/`POST` that is irreversible (evaluate case-by-case)
- `create` and `update` are NOT destructive (they can be undone by another update/delete)

## 6. Checklist: Adding a New Subcommand

When the OpenAPI spec adds a new endpoint:

- [ ] **Map the endpoint** to a command using the naming rules in section 1
- [ ] **Define flags** per section 2 (pagination, filters, sort, positional args)
- [ ] **Define table columns** for human output (max 5-6, most useful fields)
- [ ] **Map error codes** — any new API error codes? Add to the exit code table in `CLI.md`
- [ ] **Destructive?** — if DELETE or irreversible, add confirmation flow per section 5
- [ ] **Update `CLI.md`** — add the command to the command tree and full subcommand reference
- [ ] **Add example** — at least one example invocation in `CLI.md` section 8
- [ ] **Test** — verify human output, `--json` output, error cases, and `--quiet` behavior
