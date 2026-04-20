# ollygarden

CLI client for the OllyGarden observability API. Query services, insights, analytics, and manage webhooks from your terminal.

## Install

```bash
go install github.com/ollygarden/ollygarden-cli/cmd/ollygarden@latest
```

Or build from source:

```bash
git clone https://github.com/ollygarden/ollygarden-cli.git
go install ./cmd/ollygarden
```

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

## Usage

```bash
ollygarden organization                        # your org tier, features, score
ollygarden services list                       # all services
ollygarden services get <service-id>           # single service details
ollygarden insights list --status active       # active insights
ollygarden insights summary <insight-id>       # AI-generated summary
ollygarden analytics services                  # per-service analytics
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
