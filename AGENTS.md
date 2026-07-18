# Repository guidance

## Purpose

This repository contains the public `ollygarden` CLI for the OllyGarden REST
API. It supports interactive terminal use and automation across services,
insights, analytics, organization, authentication, and webhook workflows.

## General

- `specs/CLI.md` is the source of truth for every command. Read it before implementing or modifying any subcommand.
- `specs/CLI_GUIDELINES.md` defines extension rules. Follow the 8-point checklist (§6) when adding a subcommand.
- API schemas live at `https://api.ollygarden.cloud/openapi.json` — fetch it when you need request/response types (see "Before Adding a New Command" below).
- Before defining a new type, helper, or utility, check whether one already exists in `internal/`. Prefer reuse over duplication.
- Add code comments to explain the **why**, not the **what**.
- The canonical repository skill tree is `.agents/skills`. `.claude/skills` is a compatibility symlink; add or edit skills through `.agents/skills`.
- After a code change, run formatting and the validation commands below before finishing.

```bash
gofmt -w -- file1.go file2.go
go mod tidy
git diff --exit-code -- go.mod go.sum
go build ./...
go vet ./...
go test ./...
go test -race ./...
golangci-lint run
```

## Code Structure

- CLI entry point: `cmd/ollygarden/main.go` → `cmd/root.go` → subcommand files.
- One file per subcommand: `cmd/<noun>_<verb>.go` (e.g., `cmd/services_list.go`).
- Shared logic lives in `internal/`: HTTP client (`internal/client/`), output formatter (`internal/output/`), auth (`internal/auth/`).
- Use `spf13/cobra` for command registration. Every command must set `Use`, `Short`, `Args`, and `RunE`.
- Keep command files thin: parse flags → call client → format output → handle errors. No business logic in `cmd/`.
- API response types are currently hand-defined inline in each command file (e.g., `insightDetail`, `insightSummaryDetail`). A future improvement is to generate them from `https://api.ollygarden.cloud/openapi.json` via `oapi-codegen`.

## HTTP Client

- Single shared client in `internal/client/` — all commands reuse it.
- Auth: `Authorization: Bearer <key>` header. `OLLYGARDEN_API_KEY` takes precedence over saved contexts; credentials can also come from the mode-`0600` user config. Never accept secrets as flags or print unmasked keys.
- Base URL: `--api-url` flag > `OLLYGARDEN_API_URL` env > `https://api.ollygarden.cloud`.
- API base path: `/api/v1`. All endpoints are prefixed with this.
- Response envelope: `{data, meta, links}`. Error envelope: `{error{code, message, details}, meta}`.
- Parse API error codes and map to exit codes per `specs/CLI.md` §5.

## Output

- Human mode (default, TTY): tables for lists (max 5-6 columns), key-value pairs for single resources.
- `--json`: print full API response envelope to stdout. No transformation. Errors to stderr as JSON.
- `--quiet`: suppress informational stderr. Success = exit 0, no output (unless `--json`).
- stdout = data only. stderr = errors, diagnostics, prompts, pagination hints.
- Respect `NO_COLOR` env var. Detect TTY via `isatty(stdout)`. Non-TTY disables colors and truncation.
- Pagination hint: when `meta.has_more` is true in human mode, print `# N more results. Use --offset X to see next page.` on stderr.

## Errors & Exit Codes

- Exit codes: 0=success, 1=general, 2=usage, 3=auth, 4=not-found, 5=rate-limit, 6=server. See `specs/CLI.md` §5 for full mapping.
- Human errors to stderr: `Error: <message>`. Include `trace_id` when available.
- Never add new exit codes without updating `specs/CLI.md`.
- Validate flags early, before any network I/O. Bad input → exit 2 with actionable message.

## Flags & Args

- API `snake_case` params → CLI `--kebab-case` flags. See `specs/CLI_GUIDELINES.md` §2.
- All list commands: `--limit` (int, 1-100) and `--offset` (int, ≥0). Match API defaults per `specs/CLI.md`.
- Resource IDs are always positional args, never flags. Max 2 positional args per command.
- Repeatable flags for array fields: `--event-type foo --event-type bar`.
- `-q` is globally reserved for `--quiet`. `-h` for `--help`. No other global short flags.
- Sort flags: `--sort <field>` (asc), `--sort -<field>` (desc), `--sort +<field>` (explicit asc).

## Testing

- Every subcommand must have a test file: `cmd/<noun>_<verb>_test.go`.
- Use `github.com/stretchr/testify` for assertions.
- Table-driven tests for flag parsing, output formatting, and error mapping.
- Test both human and `--json` output modes.
- Test error cases: missing auth, bad flags, 404, 429, 5xx.
- Colocate tests next to implementation files.

## Safety

- `webhooks delete` is the only destructive command. It requires interactive confirmation (TTY) or `--confirm` flag.
- Non-TTY without `--confirm` → exit 2: `Error: --confirm required for non-interactive webhook deletion`.
- `--quiet` never suppresses confirmation prompts.
- Prompt format: `Delete webhook "<name>" (id: <id>)? [y/N]:` — default No.
- If a new DELETE endpoint is added, apply the same confirmation pattern.

## Before Adding a New Command

Before implementing a new CLI command, fetch the latest OpenAPI schema so you're working against the live API:

```bash
# 1. Fetch the current schema (cache it locally, don't commit it)
curl -fsSL https://api.ollygarden.cloud/openapi.json -o /tmp/openapi.json

# 2. Check which endpoints exist vs CLI commands
# Compare /tmp/openapi.json paths against specs/CLI.md command tree

# 3. Verify the endpoint you need exists in /tmp/openapi.json
# If it doesn't, the endpoint must be added to the API first — the CLI cannot call endpoints that don't exist

# 4. After confirming the endpoint exists, follow the 8-point checklist in specs/CLI_GUIDELINES.md §6
```

## Specs (read before any CLI work)

- `specs/CLI.md` — command tree, flags, output format, exit codes, examples
- `specs/CLI_GUIDELINES.md` — extension rules, 8-point checklist for new subcommands
- `https://api.ollygarden.cloud/openapi.json` — API schemas, request/response types (remote, fetch on demand)

## Pull requests

Follow [CONTRIBUTING.md](CONTRIBUTING.md). Keep CLI surface changes synchronized
with the specifications, describe user-visible compatibility effects, and list
the exact validation performed.

## Documentation and compatibility checks

For repository-guidance or skill changes, run these checks from the repository
root. Set `BASE_SHA` to the pull request base commit when checking an already
committed branch diff.

```bash
test -f AGENTS.md
test ! -L AGENTS.md
test -L CLAUDE.md
test -e CLAUDE.md
test "$(readlink CLAUDE.md)" = AGENTS.md
cmp -s AGENTS.md CLAUDE.md
test -d .agents/skills
test ! -L .agents/skills
test -L .claude/skills
test -e .claude/skills
test "$(readlink .claude/skills)" = ../.agents/skills
git diff --check
test -z "${BASE_SHA:-}" || git diff --check "${BASE_SHA}...HEAD"
perl -MFile::Basename=dirname -MFile::Spec -ne 'while (/\[[^]]+\]\(([^)#]+)(?:#[^)]+)?\)/g) { $target = $1; next if $target =~ m{^(?:https?://|mailto:)}; $path = File::Spec->catfile(dirname($ARGV), $target); die "$ARGV: missing $target\n" unless -e $path }' AGENTS.md README.md CONTRIBUTING.md
```
