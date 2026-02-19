# Task Creation Prompt

Generate the `ollygarden` CLI task backlog from `specs/CLI.md`.

## Prompt

> Read `specs/CLI.md` (command tree + subcommand reference). Generate one task per subcommand, plus a scaffold task first.
>
> Output in this exact format:
> ```
> - [ ] **task-id**: <one-line goal>
>   - Spec: specs/CLI.md section N.N
>   - Endpoint: <HTTP method> <path>
>   - Scope: <files/packages affected>
>   - Accept: <verification criteria>
> ```
>
> Rules:
> - Phase 1 = scaffold (Go module, main.go, root cmd, HTTP client, auth, output formatter, --json/--quiet/--version)
> - Phase 2 = one task per subcommand from the CLI.md command tree
> - Order by dependency (scaffold first, then simple reads, then writes)
> - Scope tells the agent where to look and what to create
> - Accept tells the agent when to stop
> - Group into phases with `## Phase N — <name>` headers

## Example Output

```markdown
# Tasks

## Phase 1 — Scaffolding
- [ ] **scaffold**: Go module, main.go, root command, HTTP client, auth (OLLYGARDEN_API_KEY), output formatter, --json/--quiet/--version
  - Scope: go.mod, main.go, cmd/, internal/
  - Accept: `go build ./...` passes, `ollygarden --help` shows command groups

## Phase 2 — Read Commands
- [ ] **services-list**: `ollygarden services list`
  - Spec: specs/CLI.md §3.2
  - Endpoint: GET /api/v1/services
  - Scope: cmd/services_list.go, internal/
  - Accept: `ollygarden services list --help` shows flags, `go test ./...` passes

- [ ] **services-get**: `ollygarden services get <id>`
  - Spec: specs/CLI.md §3.5
  - Endpoint: GET /api/v1/services/{id}
  - Scope: cmd/services_get.go, internal/
  - Accept: `ollygarden services get --help` shows usage, `go test ./...` passes

## Phase 3 — Write Commands
- [ ] **webhooks-create**: `ollygarden webhooks create`
  - Spec: specs/CLI.md §3.12
  - Endpoint: POST /api/v1/webhooks
  - Scope: cmd/webhooks_create.go, internal/
  - Accept: `ollygarden webhooks create --help` shows flags, `go test ./...` passes
```

## How Ralph Reads It

- Picks the **first unchecked `[ ]`** task top to bottom
- Ignores `[x]` (completed) and headers
- Ordering = priority
