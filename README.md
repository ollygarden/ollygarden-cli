# ollygarden

CLI client for the OllyGarden REST API. Query services, insights, analytics, and manage webhooks from your terminal.

This is the actively maintained public OllyGarden CLI. Repository-specific development and safety guidance lives in [AGENTS.md](AGENTS.md); contribution expectations are in [CONTRIBUTING.md](CONTRIBUTING.md).

## Install

**macOS / Linux**

```bash
curl -fsSL https://raw.githubusercontent.com/ollygarden/ollygarden-cli/main/install.sh | sh
```

**Windows (PowerShell)**

```powershell
irm https://raw.githubusercontent.com/ollygarden/ollygarden-cli/main/install.ps1 | iex
```

The installer selects the correct architecture, verifies the release checksum,
installs to `%LOCALAPPDATA%\OllyGarden`, and adds it to your user `Path`.

To install manually, download a Windows zip from
[releases](https://github.com/ollygarden/ollygarden-cli/releases/latest),
extract it, and add its directory to your user `Path`.

**From source** — `go install github.com/ollygarden/ollygarden-cli/cmd/ollygarden@latest` (reports version `dev`).

Both installers honor `OLLYGARDEN_VERSION` and `OLLYGARDEN_INSTALL_DIR`.

Release binaries can update themselves on macOS, Linux, and Windows:

```bash
ollygarden update
```

The command verifies the release checksum and staged binary before replacing
the current executable. Homebrew or other package-manager installations that
use a symlink must be updated through that package manager. Development builds
reporting version `dev` should be refreshed with `go install` instead.

Release builds also check for a newer stable version during normal interactive
commands. The check runs concurrently and failures stay silent. When an update
is available, the CLI prints:

```text
Update Available
New version v0.3.0 is available. Run `ollygarden update`.

Changelog: https://github.com/ollygarden/ollygarden-cli/releases/tag/v0.3.0
```

The notice is suppressed for `--json`, `--quiet`, non-interactive output,
help, version, completion, and update commands.

## Configuration

Two important environment variables should be set depending on the environment you are working in. The URL and API key differ by environment.

| Setting | Flag | Env var | Default |
|---------|------|---------|---------|
| API key | — | `OLLYGARDEN_API_KEY` | *(required)* |
| API URL | `--api-url` | `OLLYGARDEN_API_URL` | `https://api.ollygarden.cloud` |

To point at a different environment (e.g. internal):

```bash
export OLLYGARDEN_API_URL=https://api.internal.ollygarden.cloud
ollygarden services list
```

Or per-command:

```bash
ollygarden services list --api-url https://api.internal.ollygarden.cloud
```

Flag takes precedence over env var, which takes precedence over the default.

## Auth

Save credentials to disk so you don't have to export `OLLYGARDEN_API_KEY` in every shell:

```bash
# Interactive (hidden prompt):
ollygarden auth login

# Or pipe a token:
echo "$OLLYGARDEN_API_KEY" | ollygarden auth login

# Or read from a file:
ollygarden auth login --token-file /path/to/token
```

Get a token at <https://ollygarden.app/settings>.

Multiple contexts (e.g. for different orgs or environments) coexist in one config:

```bash
ollygarden auth login --context prod
ollygarden auth login --context internal --api-url https://api.internal.ollygarden.cloud
ollygarden auth use-context prod
ollygarden auth list-contexts
```

Per-invocation override without changing the active context:

```bash
ollygarden --context internal services list
```

Show what's active (probes the API by default):

```bash
ollygarden auth status            # validates the token via /organization
ollygarden auth status --no-probe # offline check
```

Remove credentials:

```bash
ollygarden auth logout                       # remove current context
ollygarden auth logout --context internal    # remove a specific context
ollygarden auth logout --all --confirm       # remove everything
```

**Storage:** YAML at `os.UserConfigDir()/ollygarden/config.yaml` with mode `0600`. Override the path with `OLLYGARDEN_CONFIG`.

**Precedence:** `OLLYGARDEN_API_KEY` env var (when set) always wins over saved contexts, so CI runs and ad-hoc invocations continue to work.

## Usage

```bash
ollygarden organization                        # your org tier, features, score
ollygarden services list                       # all services
ollygarden services get <service-id>           # single service details
ollygarden insights list --status active       # active insights
ollygarden insights summary <insight-id>       # AI-generated summary
ollygarden analytics services                  # per-service analytics
ollygarden update                              # install a newer stable CLI release
ollygarden webhooks create --name alerts \
  --url https://example.com/hook               # create a webhook
```

Every command supports `--help` for full flag details.

## Output Modes

```bash
ollygarden services list                       # human-readable tables (default)
ollygarden services list --json                # full API JSON envelope
ollygarden services list --json | jq '.data'   # pipe to jq
ollygarden services list -q                    # quiet: exit code only
```

## Pagination

All list commands support `--limit` and `--offset`:

```bash
ollygarden services list --limit 10 --offset 20
```

## Shell Completion

```bash
ollygarden completion bash > /etc/bash_completion.d/ollygarden
ollygarden completion zsh > "${fpath[1]}/_ollygarden"
ollygarden completion fish > ~/.config/fish/completions/ollygarden.fish
```

## Development

Use the Go version declared in `go.mod`. Before submitting changes, follow the
validation and CLI compatibility rules in [CONTRIBUTING.md](CONTRIBUTING.md) and
keep command behavior synchronized with [specs/CLI.md](specs/CLI.md).
Maintainer release instructions are in [docs/RELEASING.md](docs/RELEASING.md).

## License

[Apache License 2.0](LICENSE)
