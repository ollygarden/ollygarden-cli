# Design: Interactive `auth login` & on-disk token storage

- **Linear**: [E-1910](https://linear.app/ollygarden/issue/E-1910/ollygarden-cli-interactive-auth-login-and-on-disk-token-storage)
- **Branch**: `jpkroehling/e-1910-cli-auth-login`
- **Status**: design approved, ready for implementation plan
- **Date**: 2026-04-30

## Problem

Today `ollygarden-cli` reads its API key from `OLLYGARDEN_API_KEY` only (`cmd/root.go:39-44`). There is no on-disk persistence and no way to keep more than one credential active. Users juggling prod (`api.ollygarden.cloud`) and internal (`api.internal.ollygarden.cloud`) must export the variable in every shell.

The CLI's primary consumer is coding agents (Claude Code and similar). The current setup forces every agent invocation to receive the token through the parent shell's environment, which is workable but brittle: agents cannot persist a credential across sessions, cannot introspect "am I configured?", and cannot operate against two organizations within one task without mutating shell state.

## Goals

- Add an interactive `ollygarden auth login` flow that persists credentials to disk with mode `0600`.
- Support multiple named contexts (kubeconfig-style) so prod and internal can coexist.
- Make the entire surface usable non-interactively, because agents are the primary caller.
- Preserve the existing env-var path: `OLLYGARDEN_API_KEY` continues to work and continues to win over saved contexts.
- Add `auth status`, `auth logout`, `auth use-context`, `auth list-contexts` for credential lifecycle and introspection.

## Non-goals (v1)

- Browser-based OAuth/PKCE login. Olive currently only accepts `og_sk_*` keys (`olive/internal/auth/apikey.go`); a CLI-auth endpoint on Olive/Petal is a separate workstream.
- OS keychain integration (macOS Keychain, Linux Secret Service, Windows Credential Manager). Plain file at mode `0600` matches the prevailing posture (gcx, `~/.aws/credentials`, `~/.kube/config`) and keeps Linux-server / WSL / container use working without a fallback layer. The schema leaves room to add this later as an opt-in storage backend.
- System-wide config layer or a per-workdir `./.ollygarden.yaml`.
- TLS / mTLS configuration.
- Per-context defaults (e.g. default `--limit`, default service ID). Schema is intentionally just credentials; future fields can be added in a backwards-compatible point release.
- A new "network/transient" exit code. Current network failures roll into exit 1; this is a separate, CLI-wide discussion.
- An `ollygarden auth token` command. `gh` ships this and it's the cleanest agent primitive ("`OLLYGARDEN_API_KEY=$(ollygarden auth token --context internal) some-other-tool`"). Excluded from v1 to keep scope tight; agents already have env var + on-disk config. Add when a concrete chained-tool use case appears.

## Considered alternatives (rejected)

- **JSON instead of YAML for the config file.** Mercury uses JSON. Pros: no parser quirks, `jq`-friendly, every language has stdlib support. Cons: less convention-aligned with kubeconfig/gcx — the "kubeconfig-style multi-context" mental model expects YAML, and we already chose YAML to mirror the `--kebab-case` flag convention. Both are workable; we kept YAML for consistency with the gcx reference and prevailing convention for this kind of file. Revisit only if a real friction emerges.
*(Earlier draft considered hand-rolled `$HOME/.config/ollygarden/` to mirror gcx; we flipped to `os.UserConfigDir()` because it's more idiomatic Go and cross-platform-correct out of the box. macOS users get `~/Library/Application Support/ollygarden/`, which is the platform-native location even if some dev-tool authors prefer `~/.config`. The stdlib helper is the right call.)*

## References and lessons learned

This design draws on three CLIs we surveyed:

- **`grafana/gcx`** (`internal/config`, `internal/login`) — primary structural reference. We borrow: kubeconfig-style multi-context, secret-tagged struct fields, `0600` mode, env-var-still-wins precedence, atomic-write pattern. We skip: PKCE flow, sentinel-retry pattern, TLS, system+local config layers, refresh-token machinery.
- **`cli/cli` (`gh`)** — the gold-standard CLI auth UX. We adopt: schema versioning (gh uses a `version` key + a `Migrate` system; we'll start with the field and add migration logic when we need it). We deliberately skip: OS keyring as the *default* (we have headless-Linux/container/CI agent users), multi-account-per-host (out of scope), `--show-token` on `auth status` (we'd have to expose it through a separate `auth token` command if we add one later — explicit YAGNI for v1), `auth setupgit` / `gitcredential` (irrelevant). We note for future work: `gh auth token` is a strong agent primitive that prints the active token to stdout — worth adding once we have a use case beyond `OLLYGARDEN_API_KEY=$(...)`.
- **`MercuryTechnologies/mercury-cli`** (`internal/auth/credentials.go`) — confirms the keyring-first + plaintext-fallback pattern we describe in the "future storage backends" section below. We adopt three real-world details from Mercury's implementation: deleting the credentials file when the last context is removed (cleaner state), being honest with the user when the plaintext fallback was used ("Storage: plaintext file — system keyring unavailable"), and using a package-level `pathFunc` test seam so unit tests can redirect writes into `t.TempDir()`. We also note Mercury's 3-second timeout wrapping every keyring operation: "Timeouts protect against Secret Service / kwalletd hangs on Linux." When we add keyring later, we'll need this. Mercury chose JSON over YAML for the credentials file and `os.UserConfigDir()` over hand-rolled `$HOME/.config/` — both are reasonable alternatives we considered and explicitly did not adopt (see "Considered alternatives" below).

## Approach

**Two new internal packages, one new command group, one modified entry point, two spec doc updates.**

- `internal/config/` — schema, loader, writer. Pure filesystem; no HTTP. Independently testable.
- `internal/auth/` — login orchestration, key masking, credential resolution. Depends on `internal/config` and `internal/client`. The only place that combines API calls with file writes.
- `cmd/auth_*.go` — five new subcommands under an `auth` parent.
- `cmd/root.go` — `PersistentPreRunE` becomes a small resolver that calls `auth.Resolve` and never touches the file directly.

The split exists so that any future caller of the file format (a debug tool, an MCP server, an `ollygarden config view` command) can import `internal/config` without dragging in the HTTP-bearing login flow.

## Architecture

```text
ollygarden-cli/
├── cmd/
│   ├── root.go                 # MODIFIED — credential resolution, error msg
│   ├── auth.go                 # NEW — `auth` parent command
│   ├── auth_login.go           # NEW
│   ├── auth_logout.go          # NEW
│   ├── auth_status.go          # NEW
│   ├── auth_use_context.go     # NEW
│   ├── auth_list_contexts.go   # NEW
│   └── *_test.go               # NEW per command
├── internal/
│   ├── config/                 # NEW — schema + on-disk I/O (no network)
│   │   ├── config.go           # types: Config, Context
│   │   ├── path.go             # config file path resolution + OLLYGARDEN_CONFIG
│   │   ├── loader.go           # Load(), Write() with 0600 perms + atomic rename
│   │   └── *_test.go
│   └── auth/                   # NEW — login flow, masking, resolution
│       ├── login.go            # Login(LoginInputs) (LoginResult, error)
│       ├── resolve.go          # Resolve(ResolveInputs) (Credentials, error)
│       ├── mask.go             # MaskKey("og_sk_abc123_xx..xx") -> "og_sk_abc123_••••"
│       └── *_test.go
└── specs/
    ├── CLI.md                  # MODIFIED — auth subtree, credential storage section
    └── CLI_GUIDELINES.md       # MODIFIED — auth-cmd notes, secret storage notes
```

Existing commands (`services`, `insights`, `analytics`, `webhooks`, `organization`) and `internal/client/`, `internal/output/`, `internal/exitcode/` are untouched.

## Config file schema

### Path resolution

1. `$OLLYGARDEN_CONFIG` (full path, env override — for agent sandboxes that want a scoped file).
2. `filepath.Join(os.UserConfigDir(), "ollygarden", "config.yaml")` (default).

`os.UserConfigDir()` returns the platform-correct directory: `$XDG_CONFIG_HOME` (or `~/.config`) on Linux, `~/Library/Application Support` on macOS, `%AppData%` on Windows. Stdlib, no extra dependency, no hand-rolled `$HOME` joining. Mercury made the same choice.

### Format

YAML. File mode `0600`. Parent dir created with `0700`. Writes are atomic: write to `config.yaml.tmp` in the same dir → `fsync` → `os.Rename`. If `fsync` fails on an exotic filesystem, log a debug warning and continue.

```yaml
# {os.UserConfigDir()}/ollygarden/config.yaml
# Linux:   ~/.config/ollygarden/config.yaml
# macOS:   ~/Library/Application Support/ollygarden/config.yaml
# Windows: %AppData%\ollygarden\config.yaml
version: 1
current-context: prod
contexts:
  prod:
    api-url: https://api.ollygarden.cloud
    api-key: og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  internal:
    api-url: https://api.internal.ollygarden.cloud
    api-key: og_sk_def456_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
```

### Go types

```go
// internal/config/config.go

type Config struct {
    Version        int                 `yaml:"version"`
    CurrentContext string              `yaml:"current-context"`
    Contexts       map[string]*Context `yaml:"contexts"`
    // Source is the path the file was loaded from. Not serialized.
    Source string `yaml:"-"`
}

type Context struct {
    Name   string `yaml:"-"`        // map key, populated post-load
    APIURL string `yaml:"api-url"`
    APIKey string `yaml:"api-key"`  // sensitive — never log, never include in errors
}
```

**Schema versioning:** `version: 1` is written at every save. Loader behavior:

- Missing `version` → assumed `1` (forward-compat for files written by an early prerelease).
- `version: 1` → load directly into the struct above.
- `version > 1` → return a clear error: `Config file version N is newer than this CLI supports. Upgrade ollygarden CLI.` (Exit 7, `CONFIG_UNREADABLE`.)

The version field exists so we can introduce a real `Migrate` system later (à la `gh`'s `internal/config/migration/`) without writing a "detect schema shape" parser. Cost today: one int field + a one-line write and a one-switch read.

Field naming: kebab-case in YAML to match the existing `--kebab-case` flag convention (CLI_GUIDELINES.md §2).

### Behaviors

- Empty file or missing file → `Config{Version: 1, Contexts: map[string]*Context{}}`, no error. First `auth login` writes it.
- Corrupted YAML → return a typed error pointing at the file path. Do not auto-recover.
- Unknown fields → ignore (forward compat).
- Windows file modes: POSIX modes do not apply the same way; mode `0600` is best-effort. Documented limitation, not enforced in tests.
- **When the last context is removed** (via `auth logout` or `auth logout --all`), delete the config file rather than leaving an empty `contexts: {}`. Mercury does this; cleaner state for the "I'm completely logged out" case. A subsequent `auth login` recreates the file.

### Future storage backends (architectural intent, not v1 work)

`internal/config` exposes a `Source` interface that `Load` and `Write` go through. v1 ships a single `FileSource` implementation (`os.ReadFile` / atomic `os.Rename`). This is the same seam `gh` uses for keyring fallback and Mercury uses for keyring + plaintext-file dual storage.

A future PR can add a `KeyringSource` that keeps non-secret fields (`version`, `current-context`, per-context `api-url`) in the YAML file but delegates `api-key` to the OS keyring (macOS Keychain, Linux Secret Service via `libsecret`, Windows Credential Manager). The schema doesn't change; resolution stays the same; only the `Source` implementation differs.

Two implementation lessons recorded now so we don't relearn them:

- **3-second timeout on every keyring operation.** Mercury wraps every `keyring.Get`/`Set`/`Delete` in a goroutine + `select` with a 3-second timeout. Comment from their code: *"Timeouts protect against Secret Service / kwalletd hangs on Linux."* Linux D-Bus / `gnome-keyring` / `kwalletd` can hang indefinitely on misconfigured systems; without the timeout, every CLI invocation could stall. This is the kind of detail that's invisible until production.
- **Be honest about which storage was used.** Mercury's `SaveToken` returns `(insecureFallback bool, err)`, and the human-mode login output prints `Storage: plaintext file — system keyring unavailable` with the file path when fallback was used. Users in regulated environments need to know whether their token landed in the keychain or in a `0600` file; agents may want to record this in their telemetry.

Why we're not shipping keyring in v1: our primary consumer (coding agents) often runs in containers, CI runners, WSL, or headless Linux without an active keyring service. Plaintext fallback would be the de-facto path for most invocations, so the keyring path adds dependency surface (`zalando/go-keyring`), platform-specific build complications (`libsecret-dev` on Debian, etc.), and three new failure modes for marginal real-world benefit. We add it when an enterprise customer asks. The architectural intent above means that's a contained change.

## `auth` subcommands

All commands support the global `--json` flag and respect `--quiet`.

### `ollygarden auth login`

Three input modes for the token, picked in this order:

1. `--token-file PATH` — read token from file. Preserves the "no secrets as flags" rule; a path is not a secret. Agents drop the token into `/tmp/og-token` and pass the path.
2. Stdin is not a TTY (piped) — read one line from stdin. Enables `echo $TOKEN | ollygarden auth login`.
3. Stdin is a TTY — print the token URL hint to stderr, then `term.ReadPassword`.

Other flags:

- `--api-url URL` (default `https://api.ollygarden.cloud`).
- `--context NAME` — overrides auto-derived name. Collisions with an existing context always overwrite (CI-friendly; this is the only way the user gets to choose the name).
- `--no-activate` — add the context but do not change `current-context`. For agent setup scripts that pre-populate multiple contexts.

Auto-context-naming rule: strip leading `api.` from the URL host, replace remaining `.` with `-`. `api.ollygarden.cloud` → `ollygarden-cloud`. `api.internal.ollygarden.cloud` → `internal-ollygarden-cloud`.

Collision rule when `--context` is **not** set: if a context with the auto-derived name already exists, **overwrite it**. Re-logging-in to the same target (token rotation) is the common case; appending `-2`/`-3` would silently spawn dead contexts. Two genuinely different URLs that derive the same name is vanishingly rare; users in that situation must pass `--context NAME` explicitly.

Behavior, in strict order:

1. Shape-check token: regex `^og_sk_[A-Za-z0-9]{6}_[a-f0-9]{32}$`. Fail fast (exit 2, `INVALID_TOKEN_FORMAT`) before any network call.
2. `GET /api/v1/organization` with the token. 200 → continue. 401 → exit 3, `TOKEN_REJECTED`. Other → existing error map.
3. Atomic write to config file. **If the write fails, print the write error and exit 7 (`CONFIG_WRITE_FAILED`) — do not print success.** The user may have a valid token that simply couldn't be persisted; they can retry or fix the underlying file-system issue.
4. Only after successful write, print the success line / JSON result.

Output (human):

```text
✓ Logged in to "Acme Corp" as context "prod" (https://api.ollygarden.cloud).
```

Output (JSON, for agents):

```json
{"data":{"context":"prod","api_url":"https://api.ollygarden.cloud","organization":"Acme Corp","key_masked":"og_sk_abc123_••••","activated":true},"meta":{}}
```

### `ollygarden auth logout`

- Default (no args): removes the current context and unsets the `current-context` pointer. Reversible by re-running `auth login`.
- `--context NAME`: removes the named context. If it was the current context, also unsets `current-context`.
- `--all`: removes every context. **Requires** `--confirm` in non-TTY (per CLI_GUIDELINES.md §5); prompts on TTY. Default and `--context` removal do not need confirmation; only `--all` does.
- After removal, when other contexts remain and `current-context` is now unset, the human-mode output prints a hint listing them and how to activate one. JSON mode just returns the structured result without prose.
- Exit 0 on success. Exit 4 (`CONTEXT_NOT_FOUND`) if `--context NAME` doesn't exist.

Example human output after a default logout when other contexts remain:

```text
✓ Logged out of "prod".
No current context set. Available: internal, dev. Activate with `ollygarden auth use-context NAME`.
```

### `ollygarden auth status`

Prints the active credential's source, URL, and masked key.

- Default: probes `GET /api/v1/organization` to confirm the token works. Matches `gh auth status` precedent.
- `--no-probe`: skip the network call. Agents use this for cheap "am I configured?" checks before deciding to call other commands.
- Exit codes:
  - `0` — logged in (and probe succeeded if probing).
  - `3` — no current context configured, or probe got 401.
  - Other codes per the existing error map for network/server failures.

Output (human):

```text
Source:        env (OLLYGARDEN_API_KEY)
API URL:       https://api.ollygarden.cloud
Key:           og_sk_abc123_••••
Organization:  Acme Corp
```

When env var wins but a context exists, output also notes the saved context that *would have* won:

```text
Source:        env (OLLYGARDEN_API_KEY) — overrides saved context "prod"
```

JSON:

```json
{"data":{"source":"context","context":"prod","api_url":"https://api.ollygarden.cloud","key_masked":"og_sk_abc123_••••","probed":true,"organization":"Acme Corp"},"meta":{}}
```

When `source: "env"` and a saved context exists, the JSON includes both `source: "env"` and `context: "prod"`.

### `ollygarden auth use-context <name>`

Sets `current-context: <name>`. Exit 4 (`CONTEXT_NOT_FOUND`) if name doesn't exist. No confirmation, no probe — it's a one-line file edit.

### `ollygarden auth list-contexts`

Table (human) or JSON. Columns: `CURRENT` (marker `*` for the active context, blank otherwise), `NAME`, `API URL`. **No keys shown** — this is a directory, not a credential dump. Use `auth status` to see the active key.

JSON envelope: `{data: [...], meta: {}}` to stay consistent with the rest of the CLI.

## Resolution rules

What `PersistentPreRunE` does for every non-`auth` command. Two things to resolve: API URL and API key. Both follow the same precedence shape with one twist for `--context`.

### API key

| Source | Wins when |
|---|---|
| `OLLYGARDEN_API_KEY` env var | always wins if set |
| Context selected by `--context NAME` flag | env unset, flag set |
| Context selected by `OLLYGARDEN_CONTEXT` env var | env unset, flag unset, context env set |
| `current-context` from config file | env unset, no flag, no context env |
| *none* | → exit 3, `NO_CREDENTIALS` |

### API URL

| Source | Wins when |
|---|---|
| `--api-url` flag | always wins if set |
| `OLLYGARDEN_API_URL` env var | flag unset, env set |
| `api-url` from selected context (same selection rules as the key) | flag unset, env unset, context selected |
| `https://api.ollygarden.cloud` (built-in default) | nothing else set |

**API URL and API key resolve independently.** A user can `--api-url=https://api.internal.ollygarden.cloud --context=prod` to send a prod key against the internal API for debugging. Cross-source is allowed, not flagged. This is documented in CLI.md.

### `--context NAME` (new global flag)

Per-invocation override. Does not change `current-context` on disk. Pairs with agent workflows where one task touches two organizations:

```bash
ollygarden --context internal services list
ollygarden --context prod services list
```

Works on every command including `auth status`. Exit 4 (`CONTEXT_NOT_FOUND`) if the named context doesn't exist.

### `OLLYGARDEN_CONTEXT` env var

Same effect as `--context`, lower precedence than the flag. Useful for shell sessions that want to pin a context for a sequence of commands without touching disk.

### Edge cases

- **Both `--context` and `--api-url` passed**: both honored independently per the tables. No warning.
- **`--context NAME` set together with `OLLYGARDEN_API_KEY`**: env var still wins for the *key*; `--context` selects which context's *URL* applies (subject to `--api-url`/`OLLYGARDEN_API_URL` overriding). Most surprising interaction in the spec; explicitly documented.
- **Config file unreadable** (perms wrong, corrupted YAML): hard error, exit 7 (`CONFIG_UNREADABLE`). Do not silently fall through to env-only — masking real problems is hostile to agents.
- **Config file missing**: not an error. Falls through to env. First-time install path.

## Errors and exit codes

This feature adds **one new exit code** and **eight new CLI-emitted error codes** (in the JSON envelope's `error.code`).

### New exit code

| Exit | Name | Meaning |
|---|---|---|
| **7** | **config** | Local config file unreadable, malformed, or unwriteable. Inspect or remove the file at `os.UserConfigDir()/ollygarden/config.yaml` (path printed in the error message). Distinct from auth (3, "your token is bad") and usage (2, "your invocation is bad"). |

CLI.md §5 must be updated with this row.

### New CLI-emitted error codes

| Code | Exit | When | Message |
|---|---|---|---|
| `NO_CREDENTIALS` | 3 | Resolution finds no env, no flag, no current-context | `No credentials configured. Run "ollygarden auth login" or set OLLYGARDEN_API_KEY. Get a token at https://ollygarden.app/settings.` |
| `INVALID_TOKEN_FORMAT` | 2 | Token shape check fails (login or env) | `Invalid token format. Expected og_sk_xxxxxx_<32 hex>.` |
| `TOKEN_REJECTED` | 3 | `/organization` returns 401 during login or `auth status --probe` | `Token rejected by API. The token may be revoked or expired.` |
| `CONTEXT_NOT_FOUND` | 4 | `--context NAME`, `auth use-context NAME`, or `auth logout --context NAME` references a missing context | `Context "internal" not found. Run "ollygarden auth list-contexts" to see available contexts.` |
| `CONFIG_UNREADABLE` | **7** | Config file exists but can't be read or parsed | `Cannot read config file at <path>: <reason>. Inspect or remove the file to recover.` |
| `CONFIG_WRITE_FAILED` | **7** | Atomic-rename or temp-file write fails (perms, disk full) | `Cannot write config file at <path>: <reason>.` |
| `TOKEN_FILE_NOT_FOUND` | 2 | `--token-file PATH` doesn't exist or isn't readable | `Cannot read token file <path>: <reason>.` |
| `CONFIRM_REQUIRED` | 2 | `auth logout --all` in non-TTY without `--confirm` | `Refusing to remove all contexts without --confirm in non-interactive mode.` |

### Error envelope

Already standardized in CLI_GUIDELINES.md §4. Reused unchanged:

```json
{"error":{"code":"NO_CREDENTIALS","message":"...","details":{}},"meta":{}}
```

Human mode: `Error: <message>` to stderr, no envelope, exit code per the table.

### `AuthError` removal

The current `cmd/root.go:48` `AuthError` is a placeholder. It is replaced by the typed errors emitted from `internal/auth.Resolve`. The exit-code routing in `Execute()` keeps working — it just switches on the new typed errors.

## Testing strategy

### 1. Unit tests, colocated

**`internal/config/`:**

- `Load` round-trips: write a YAML file with `t.TempDir()`, load it, assert struct equality. Cover empty file, missing file, corrupted YAML (assert `CONFIG_UNREADABLE` typed error), unknown fields, `version` missing (treat as 1), `version > 1` (`CONFIG_UNREADABLE`).
- `Write` atomicity: stub `os.Rename` to fail mid-write — assert original file intact, no `.tmp` left behind. Assert mode `0600` on file, `0700` on parent dir. Skip permission assertions on Windows.
- `Write` with empty contexts deletes the file (Mercury behavior).
- `Path` resolution: `OLLYGARDEN_CONFIG` set, unset, `$HOME` unset.

**Test seam:** package-level `var pathFunc = defaultPath` lifted from Mercury's `internal/auth/credentials.go`. Tests swap it to redirect writes into `t.TempDir()` instead of touching the real `os.UserConfigDir()/ollygarden/`. Cleaner than env-var manipulation in tests.

**`internal/auth/`:**

- `Resolve` precedence: table-driven test covering every row in the precedence tables. Inputs are pure values; no I/O. **Highest-value test in the PR** — every command's auth depends on this being right.
- `MaskKey`: `og_sk_abc123_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx` → `og_sk_abc123_••••`. Edge cases: empty, malformed, already-masked.
- `Login`: with a `httptest.Server` returning 200/401/500 for `/api/v1/organization`, assert the right `LoginResult` or typed error. Stub config writer with a `Source` interface so we never touch real disk.

### 2. Cmd-layer tests (one `_test.go` per subcommand)

Reuse the `testify` + table-driven style from `cmd/insights_get_test.go`. For each `auth_*` command:

- Flag parsing.
- Human output format (compare against fixture).
- `--json` output format.
- Error mapping → exit code (via the existing `*client.APIError` / typed-error path in `Execute()`).

Specific scenarios worth their own test cases (agent-facing surface):

- `auth login --token-file <path>` — happy path + missing file + unreadable file.
- `auth login` with non-TTY stdin — reads stdin.
- `auth login` with TTY stdin — assert behavior with a fake `term.IsTerminal` and stubbed `term.ReadPassword` via an `internal/auth.Prompter` interface.
- `auth status` with no creds → exit 3, error envelope correct.
- `auth status --no-probe` → no HTTP call (verify with `httptest.Server` that records hits).
- `auth status --json` with both env and current-context → both `source: "env"` and `context: "prod"` present.
- `auth logout --all` non-TTY without `--confirm` → exit 2, `CONFIRM_REQUIRED`.
- `--context NAME` with non-existent name → exit 4, `CONTEXT_NOT_FOUND`.

### 3. Integration smoke

`cmd/auth_integration_test.go`, tagged `//go:build integration`. Spins up `rootCmd.Execute()` with `t.TempDir()` as `$HOME`, a `httptest.Server` for the API, and shells through realistic flows:

1. `auth login` → file appears with `0600` → `auth status` reports the context → `organization` command succeeds with the saved key.
2. `OLLYGARDEN_API_KEY=other ollygarden auth status` → reports `source: "env"`, `context: "prod"` (saved one is named but env wins).
3. `auth login --context prod` then `auth login --context dev` then `auth use-context prod` → file holds both, `current-context: prod`.

Skipped by default in `go test ./...`; run via `go test -tags=integration ./...` and from CI.

### Explicitly not tested

- Real TTY behavior (terminal raw mode) — relying on the prompter interface; `term.ReadPassword` is a thin wrapper we trust.
- Cross-platform file modes on Windows — ACLs apply, not POSIX modes; documented limitation.
- Real network calls to `api.ollygarden.cloud` — all HTTP is `httptest`-stubbed.

### Pre-merge gate

`go build ./... && go test ./... && go vet ./...` (already required by CLAUDE.md). Integration tag is opt-in so default invocation stays fast.

## Spec updates

### `specs/CLI.md`

- §1 (Command Tree): add `auth` subtree with `login`, `logout`, `status`, `use-context`, `list-contexts`.
- §2 (Global Flags): add `--context` global flag and `OLLYGARDEN_CONTEXT` env var. Note that `OLLYGARDEN_API_KEY` env var still wins.
- §3: add new subsections for each `auth` subcommand (flags, behavior, output examples, exit codes).
- §5 (Error Handling): add exit code 7 (config) row. Add a "CLI-emitted error codes" subsection enumerating the eight new codes.
- **New §6 (Credential Storage)**, with subsequent sections renumbered: describe the file path, mode `0600`, atomic-write semantics, `OLLYGARDEN_CONFIG` override, and the precedence tables for resolution.

### `specs/CLI_GUIDELINES.md`

- §4 (Error Handling Rules): one-line pointer to the new "CLI-emitted error codes" section in CLI.md, noting these apply to errors emitted before any HTTP call.
- §5 (Destructive Operation Safety): note that `auth logout --all` follows the destructive-op pattern; default `auth logout` does not.
- **New §8 (Auth Commands)**: brief notes on the `auth` subgroup convention and the agent-facing requirements (non-interactive paths, structured output) for any future subcommand under it. Currently §7 is "API Types"; this becomes §8.
