# Tasks

## Phase 1 — Scaffolding
- [x] **scaffold**: Go module, main.go, root command, HTTP client, auth (OLLYGARDEN_API_KEY), output formatter, --json/--quiet/--version
  - Scope: go.mod, main.go, cmd/, internal/
  - Accept: `go build ./...` passes, `ollygarden --help` shows command groups

## Phase 2 — Read Commands
- [x] **organization**: `ollygarden organization`
  - Spec: specs/CLI.md §3.1
  - Endpoint: GET /api/v1/organization
  - Scope: cmd/organization.go
  - Accept: `ollygarden organization --help` shows usage, `go test ./...` passes

- [ ] **services-list**: `ollygarden services list`
  - Spec: specs/CLI.md §3.2
  - Endpoint: GET /api/v1/services
  - Scope: cmd/services_list.go
  - Accept: `ollygarden services list --help` shows flags, `go test ./...` passes

- [ ] **services-grouped**: `ollygarden services grouped`
  - Spec: specs/CLI.md §3.3
  - Endpoint: GET /api/v1/services/grouped
  - Scope: cmd/services_grouped.go
  - Accept: `ollygarden services grouped --help` shows flags, `go test ./...` passes

- [ ] **services-search**: `ollygarden services search [query]`
  - Spec: specs/CLI.md §3.4
  - Endpoint: GET /api/v1/services/search
  - Scope: cmd/services_search.go
  - Accept: `ollygarden services search --help` shows flags, `go test ./...` passes

- [ ] **services-get**: `ollygarden services get <id>`
  - Spec: specs/CLI.md §3.5
  - Endpoint: GET /api/v1/services/{id}
  - Scope: cmd/services_get.go
  - Accept: `ollygarden services get --help` shows usage, `go test ./...` passes

- [ ] **services-versions**: `ollygarden services versions <id>`
  - Spec: specs/CLI.md §3.6
  - Endpoint: GET /api/v1/services/{id}/versions
  - Scope: cmd/services_versions.go
  - Accept: `ollygarden services versions --help` shows flags, `go test ./...` passes

- [ ] **services-insights**: `ollygarden services insights <id>`
  - Spec: specs/CLI.md §3.7
  - Endpoint: GET /api/v1/services/{id}/insights
  - Scope: cmd/services_insights.go
  - Accept: `ollygarden services insights --help` shows flags, `go test ./...` passes

- [ ] **insights-list**: `ollygarden insights list`
  - Spec: specs/CLI.md §3.8
  - Endpoint: GET /api/v1/insights
  - Scope: cmd/insights_list.go
  - Accept: `ollygarden insights list --help` shows flags, `go test ./...` passes

- [ ] **insights-get**: `ollygarden insights get <id>`
  - Spec: specs/CLI.md §3.9
  - Endpoint: GET /api/v1/insights/{id}
  - Scope: cmd/insights_get.go
  - Accept: `ollygarden insights get --help` shows usage, `go test ./...` passes

- [ ] **analytics-services**: `ollygarden analytics services`
  - Spec: specs/CLI.md §3.10
  - Endpoint: GET /api/v1/analytics/services
  - Scope: cmd/analytics_services.go
  - Accept: `ollygarden analytics services --help` shows flags, `go test ./...` passes

- [ ] **webhooks-list**: `ollygarden webhooks list`
  - Spec: specs/CLI.md §3.11
  - Endpoint: GET /api/v1/webhooks
  - Scope: cmd/webhooks_list.go
  - Accept: `ollygarden webhooks list --help` shows flags, `go test ./...` passes

- [ ] **webhooks-get**: `ollygarden webhooks get <id>`
  - Spec: specs/CLI.md §3.13
  - Endpoint: GET /api/v1/webhooks/{id}
  - Scope: cmd/webhooks_get.go
  - Accept: `ollygarden webhooks get --help` shows usage, `go test ./...` passes

- [ ] **webhooks-deliveries-list**: `ollygarden webhooks deliveries list <webhook-id>`
  - Spec: specs/CLI.md §3.17
  - Endpoint: GET /api/v1/webhooks/{id}/deliveries
  - Scope: cmd/webhooks_deliveries_list.go
  - Accept: `ollygarden webhooks deliveries list --help` shows flags, `go test ./...` passes

- [ ] **webhooks-deliveries-get**: `ollygarden webhooks deliveries get <webhook-id> <delivery-id>`
  - Spec: specs/CLI.md §3.18
  - Endpoint: GET /api/v1/webhooks/{id}/deliveries/{did}
  - Scope: cmd/webhooks_deliveries_get.go
  - Accept: `ollygarden webhooks deliveries get --help` shows usage, `go test ./...` passes

## Phase 3 — Write Commands
- [ ] **webhooks-create**: `ollygarden webhooks create`
  - Spec: specs/CLI.md §3.12
  - Endpoint: POST /api/v1/webhooks
  - Scope: cmd/webhooks_create.go
  - Accept: `ollygarden webhooks create --help` shows flags, `go test ./...` passes

- [ ] **webhooks-update**: `ollygarden webhooks update <id>`
  - Spec: specs/CLI.md §3.14
  - Endpoint: PUT /api/v1/webhooks/{id}
  - Scope: cmd/webhooks_update.go
  - Accept: `ollygarden webhooks update --help` shows flags, `go test ./...` passes

- [ ] **webhooks-delete**: `ollygarden webhooks delete <id>`
  - Spec: specs/CLI.md §3.15
  - Endpoint: DELETE /api/v1/webhooks/{id}
  - Scope: cmd/webhooks_delete.go
  - Accept: `ollygarden webhooks delete --help` shows flags, destructive confirmation works, `go test ./...` passes

- [ ] **webhooks-test**: `ollygarden webhooks test <id>`
  - Spec: specs/CLI.md §3.16
  - Endpoint: POST /api/v1/webhooks/{id}/test
  - Scope: cmd/webhooks_test.go
  - Accept: `ollygarden webhooks test --help` shows usage, `go test ./...` passes
