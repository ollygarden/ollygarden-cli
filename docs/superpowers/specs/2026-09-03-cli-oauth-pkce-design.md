# Design: CLI browser OAuth with PKCE for user-scoped Olive features

- **Linear:** [E-3427](https://linear.app/ollygarden/issue/E-3427/design-cli-browseroauth-authentication-for-user-scoped-olive-features)
- **Status:** accepted design; implementation requires the follow-up issues listed below
- **Repositories inspected:** `ollygarden-cli` at `31c54fc`; Olive `origin/main` at `fefcb4e`

## Decision

Add an OAuth credential kind alongside existing API-key contexts. A human login uses Clerk's OAuth 2.0 Authorization Code grant as a **public client**, with an external browser, an ephemeral loopback callback, PKCE S256, and state validation. The CLI never accepts or stores a Clerk browser session cookie and contains no client secret.

This is not a replacement for API keys. Existing API-key config, environment variables, commands, and precedence remain compatible. User OAuth exists for explicitly approved user-scoped operations; unattended automation continues to use `OLLYGARDEN_API_KEY`.

Do not ship the CLI flow until the production Clerk OAuth application and Olive scope policy are configured and tested. In particular, do not substitute a Petal session JWT, copied browser cookie, long-lived ad hoc token, or a home-grown device code protocol.

## Authoritative contracts and constraints

- [Clerk's OAuth implementation](https://clerk.com/docs/guides/configure/auth-strategies/oauth/how-clerk-implements-oauth) supports public clients, Authorization Code with PKCE S256, refresh grants, `offline_access`, custom scopes, and organization selection through `user:org:read`. Its authorization-server metadata is at `/.well-known/oauth-authorization-server`; the CLI discovers and validates endpoints from there instead of constructing them.
- Clerk currently advertises only `authorization_code` and `refresh_token`, not the RFC 8628 device authorization grant. Headless behavior must respect that limitation.
- Clerk access tokens are one-day JWTs by default; refresh tokens do not expire. JWT access tokens cannot be revoked immediately. Revoking a refresh token revokes its associated grant, but an already issued JWT remains usable until expiry. Olive's current verifier requires RS256, `typ=at+jwt`, the configured issuer, a non-empty `sub`, `exp`, `user:org:read`, and `org_id`.
- Olive currently allows OAuth reads, favorite mutation, pagination-snapshot release, and webhook mutations with `webhooks:write`; other OAuth mutations are denied. Olive's Rose `RequireUser` group contains onboarding, settings, integrations, and admin routes. Approved mutations below therefore require deliberate Olive policy changes and custom scopes before CLI commands can work.

## Browser login protocol

`ollygarden auth login --oauth` (the implementation may later make browser OAuth the interactive default only after a separately reviewed compatibility decision) performs:

1. Fetch `https://clerk.ollygarden.app/.well-known/oauth-authorization-server` and require the expected HTTPS issuer, `response_types_supported=code`, `authorization_code`, public-client auth method `none`, and PKCE `S256`. Endpoints must be HTTPS and same-origin with the pinned issuer, except the local callback.
2. Bind a one-request HTTP listener to `127.0.0.1:0`, never `0.0.0.0`, and use the registered loopback redirect path. Generate independent cryptographically random PKCE verifier and state values (at least 256 bits each); keep both only in memory.
3. Open the system browser to the discovered authorization endpoint with `response_type=code`, the fixed CLI client ID, exact redirect URI, S256 challenge, state, and the least-privilege scopes for the selected feature set. Baseline scopes are `user:org:read offline_access`; do not request profile, email, or metadata scopes.
4. Clerk displays consent and its organization selector. The callback rejects wrong/missing state, wrong path, non-loopback requests, duplicate callbacks, OAuth errors, oversized input, and timeouts. It returns only a generic success/failure page and shuts down immediately.
5. Exchange the single-use code directly at the discovered token endpoint with the exact redirect URI and verifier, without a client secret. Validate token response type, issuer, signature, `typ`, `exp`, `sub`, scopes, and non-empty `org_id`; probe Olive `GET /api/v1/organization`; only then persist credentials.

No authorization code, verifier, state, access token, refresh token, callback query, or full authorization URL is logged or emitted in telemetry. `--json` reports non-secret context, organization, expiry, scopes, and storage backend only.

## Organization selection

`user:org:read` makes Clerk's consent screen the authoritative organization selector and places the chosen `org_id` in the access token. The CLI does not accept an arbitrary organization ID flag and Olive never trusts a client-supplied organization header. The context records the returned organization ID plus display name from Olive only as non-authoritative UX metadata.

One OAuth context represents one grant for one selected organization. To use another organization, run login again and save a different context. `auth use-context` switches contexts locally; it does not mutate a Clerk/Petal active organization. Refresh must preserve the grant's organization and scopes. A changed or missing `org_id` on refresh fails closed and requires login again.

## Refresh, revocation, and storage

- Refresh when the access token is within five minutes of expiry, and once after an Olive 401 that plausibly indicates expiry. Serialize refresh per context and atomically replace the complete token set if Clerk rotates the refresh token. Never retry `invalid_grant`; remove unusable local OAuth credentials and require login. Network/server failures do not erase a still-usable token.
- `auth logout` for OAuth first calls Clerk's discovered revocation endpoint with the refresh token, then always removes local credentials. A revocation network failure produces a warning/non-zero result explaining that the local credential was removed but the remote grant may remain. `--local-only` is an explicit escape hatch. `logout --all` applies this per OAuth context and retains its existing confirmation requirement.
- Store the OAuth token set in the native OS credential vault (macOS Keychain, Windows Credential Manager, Linux Secret Service), keyed by a random opaque reference. The YAML file stores only that reference, credential kind, API URL, issuer/client ID, organization metadata, scopes, and expiry. It must never contain an OAuth refresh or access token.
- Unlike API keys, OAuth refresh tokens have **no plaintext fallback** because Clerk refresh tokens do not expire. If no usable credential vault exists (common in minimal containers or remote Linux), OAuth login fails with an actionable message; API-key contexts remain available. Vault operations receive bounded timeouts, values are never placed on command lines or in environment variables, and deletion is best-effort on config cleanup.
- Access tokens may remain only in memory between commands; storing the token set in the vault is allowed when needed for startup/atomic refresh, but no token is copied into YAML. File and directory permissions remain `0600`/`0700` and atomic-write rules remain unchanged.

## Headless and non-interactive behavior

The supported OAuth flow requires a browser that can reach the same machine's loopback listener. `--no-browser` may print the authorization URL for cases where the browser runs on that same desktop, but it does not claim to support SSH, containers, CI, or a browser on another host. The command times out and cleans up without persisting partial state.

There is no copy/paste callback, session-cookie import, refresh-token flag, or invented device flow. Until Clerk advertises and documents RFC 8628 support, truly headless use must use existing organization API keys via `OLLYGARDEN_API_KEY`, stdin/token-file login, or API-key contexts. OAuth commands fail before opening a browser when stdin is non-interactive unless the user explicitly requested browser login; they never silently fall back to reading a user token.

## Coexistence and migration

Config schema v2 adds a discriminated credential reference while preserving v1 API-key data:

```yaml
version: 2
current-context: acme-user
contexts:
  prod-automation:
    credential-kind: api-key
    api-url: https://api.ollygarden.cloud
    api-key: og_sk_xxxxxx_<32 hex>
  acme-user:
    credential-kind: oauth
    api-url: https://api.ollygarden.cloud
    oauth-credential-ref: random-opaque-vault-key
    oauth-issuer: https://clerk.ollygarden.app
    oauth-client-id: client_...
    organization-id: org_...
```

Missing `credential-kind` means `api-key`, so v1 files migrate in memory and are rewritten only on the next mutation. No automatic API-key-to-OAuth conversion occurs. Older CLI versions will reject schema v2 rather than misread it.

Resolution remains `OLLYGARDEN_API_KEY` > explicitly selected context > current context. Thus existing CI and scripts are unchanged. `OLLYGARDEN_API_KEY` always wins even when the selected context is OAuth. There is no OAuth access/refresh-token environment variable. The shared client resolves either API key or a fresh OAuth access token into the same `Authorization: Bearer` header.

## Approved route matrix

The matrix is intentionally narrower than Olive's current user-only surface. “CLI-approved” means a follow-up may implement it after Olive enforces the listed scope; it is not authorization to bypass current policy.

| Olive route | Capability | CLI decision | Required OAuth scope / safeguard |
|---|---|---|---|
| `GET /api/v1/services[/*]` | Read services; `favorite=true` filter and favorite fields | CLI-approved | `user:org:read` |
| `PATCH /api/v1/services/{id}/favorite` | Set own per-user favorite | CLI-approved | `user:org:read`; only authenticated `sub` |
| Existing Olive and Rose GET routes not listed below | Organization-scoped read parity | CLI-approved under existing command scope | `user:org:read`; normal server authorization |
| `GET/PUT /api/v1/rose/repositories/{id}/settings` | Repository review/fix instructions | CLI-approved | custom `rose:settings:read` / `rose:settings:write`; validate bounded input; show diff/target |
| `GET/PUT /api/v1/rose/settings/organization` | Organization review/fix instructions | CLI-approved | custom `rose:settings:read` / `rose:settings:write`; explicit org in output |
| `GET/PUT /api/v1/rose/integrations` | List/upsert observability integration | CLI-approved | custom `rose:integrations:read` / `rose:integrations:write`; secrets only via stdin/file; never echo response secrets |
| `DELETE /api/v1/rose/integrations/{id}` | Remove integration | CLI-approved | `rose:integrations:write`; TTY confirmation or `--confirm` when non-TTY |
| `POST /api/v1/rose/onboarding/session`, `GET /onboarding/status`, `POST /onboarding/repos` | GitHub App installation/onboarding | **Dashboard-only** | Browser redirect and installation ownership UX belong in Petal; no CLI proxy flow |
| `GET /api/v1/rose/admin/installations` | Cross-installation administration | **Dashboard-only** | Elevated support/admin data is not needed for normal CLI users |

Webhook commands remain available to OAuth only with existing `webhooks:write`; this design does not broaden them. Existing API-key-compatible Rose execution, finding, and repository activation routes remain unchanged and do not require OAuth.

## Threat model and controls

| Threat | Control / residual risk |
|---|---|
| Authorization-code interception or malicious local callback | PKCE S256, exact redirect, high-entropy state, loopback-only random port, one request, short timeout. A same-user local process can race the port; state and verifier prevent useful exchange. |
| Login CSRF / wrong-account grant | Independent state validation plus Clerk consent and visible organization selector. |
| Token theft from disk, backups, process listings, logs, or crash reports | Refresh/token set only in OS vault; no token flags/env/YAML; redact headers and OAuth parameters. A compromised same-user account or unlocked vault remains out of scope. |
| Stolen refresh token with no natural expiry | Remote revocation on logout, local vault deletion, least scopes, and user-visible revocation failure. Clerk account/vault compromise remains high impact. |
| Revoked JWT still accepted | Clerk JWTs remain valid until their one-day expiry; document residual risk. Prefer short-lived access tokens if Clerk makes lifetime configurable. Do not switch Olive to opaque tokens without a separate availability/performance design because Olive currently verifies JWTs locally. |
| Scope or confused-deputy escalation | Olive derives `sub`/`org_id` only from verified tokens and enforces route/method scopes deny-by-default. Rose receives only Olive-signed, short-lived assertions. |
| Cross-organization action | One org per grant/context; probe and display target org; reject refresh identity changes; no org override header/flag. |
| Malicious discovery/config endpoint | Pin expected issuer and HTTPS same-origin endpoints; fixed shipped client ID; no dynamic client registration. Custom API URLs need separately configured trusted issuer/client metadata, never discovery supplied by Olive. |
| Headless workaround leaks credentials | No cookie/token import or copy/paste protocol; direct users to API keys until a standards-based Clerk device flow exists. |
| Concurrent refresh corrupts credentials | Per-context process lock, atomic vault replacement, short refresh skew, and one controlled 401 refresh retry. |

## Delivery gates and follow-ups

1. [E-3434](https://linear.app/ollygarden/issue/E-3434/oliveclerk-enable-the-approved-cli-oauth-pkce-and-scope-contract) — Olive/Clerk contract: register the public PKCE client, configure exact loopback redirects and consent, create custom scopes, extend Olive's method/route policy for only approved routes, and test JWT claims and denial cases.
2. [E-3435](https://linear.app/ollygarden/issue/E-3435/cli-implement-browser-oauthpkce-credential-lifecycle) — CLI auth foundation: schema migration, vault abstraction, discovery, browser/loopback PKCE login, refresh serialization, status/logout/revocation, redaction, and cross-platform tests.
3. [E-3436](https://linear.app/ollygarden/issue/E-3436/cli-add-user-favorite-filters-and-mutation) — CLI favorite filter/mutation commands.
4. [E-3437](https://linear.app/ollygarden/issue/E-3437/cli-add-oauth-scoped-rose-settings-commands) — CLI Rose settings commands.
5. [E-3438](https://linear.app/ollygarden/issue/E-3438/cli-add-oauth-scoped-rose-observability-integration-commands) — CLI Rose integration commands with secret-input and destructive-operation safeguards.

Each implementation PR must re-check Clerk metadata and the live Olive OpenAPI contract, include negative authorization tests, and preserve API-key behavior. Production enablement should canary OAuth 401/403 and refresh failures without recording user, organization, or token values. Merge, release, and production configuration are outside this design task.
