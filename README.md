# ollygarden

CLI client for the OllyGarden observability API. Query services, insights, analytics, and manage webhooks from your terminal.

## Install

```bash
go install github.com/ollygarden/ollygarden-cli/cmd/ollygarden@latest
```

Or build from source:

```bash
git clone https://github.com/ollygarden/ollygarden-cli.git
cd ollygarden-cli
go build -o ollygarden ./cmd/ollygarden
```

## Auth

```bash
export OLLYGARDEN_API_KEY=og_sk_...
```

No config files. No flags. Env var only.

## Usage

```bash
ollygarden organization                        # your org tier, features, score
ollygarden services list                       # all services
ollygarden services get <service-id>           # single service details
ollygarden insights list --status active       # active insights
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

## Exit Codes

| Code | Meaning     |
|------|-------------|
| 0    | Success     |
| 1    | General     |
| 2    | Usage       |
| 3    | Auth        |
| 4    | Not found   |
| 5    | Rate limit  |
| 6    | Server      |

## Shell Completion

```bash
ollygarden completion bash > /etc/bash_completion.d/ollygarden
ollygarden completion zsh > "${fpath[1]}/_ollygarden"
ollygarden completion fish > ~/.config/fish/completions/ollygarden.fish
```
