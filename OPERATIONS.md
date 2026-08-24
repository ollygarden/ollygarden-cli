# Operations

Use the dev API for manual verification after local checks pass. Required
credentials are environment variables; never print or commit the API key.

## Dev API

### Prerequisites

| Variable | Value or purpose |
|---|---|
| `DEV_OLLYGARDEN_API_BASE_URL` | `https://api.dev.ollygarden.cloud` |
| `DEV_OLLYGARDEN_API_KEY` | Dev API credential |
| `DEV_OLLYGARDEN_OPENAPI_URL` | `https://api.dev.ollygarden.cloud/openapi.json` |

### Usage

Fetch the current dev schema before checking API-dependent changes:

```sh
curl -fsSL "$DEV_OLLYGARDEN_OPENAPI_URL" -o /tmp/ollygarden-openapi.json
```

Run the CLI against the dev API without copying the credential into saved
configuration:

```sh
OLLYGARDEN_API_URL="$DEV_OLLYGARDEN_API_BASE_URL" \
OLLYGARDEN_API_KEY="$DEV_OLLYGARDEN_API_KEY" \
  go run ./cmd/ollygarden services list --json
```

Start with read-only commands. Only test writes when the change requires them,
and avoid destructive operations against shared dev data.
