# Releasing

Tags matching `v*` trigger `.github/workflows/release.yml`. GoReleaser builds
Linux, macOS, and Windows archives for amd64 and arm64, publishes them to the
GitHub release, and publishes `checksums.txt` with SHA-256 digests.

The same GoReleaser run generates the `ollygarden` Homebrew cask from those
archives. The cask version, artifact URLs, and checksums therefore stay aligned
with the GitHub release. CI creates a release snapshot and runs
`.github/scripts/verify-release-artifacts.sh` to enforce that contract.

## Homebrew tap

The cask is published to `ollygarden/homebrew-tap` when the CLI repository has
a `HOMEBREW_TAP_GITHUB_TOKEN` Actions secret. The token must have contents write
access to that repository; the workflow's default `GITHUB_TOKEN` cannot write
across repositories. Without the secret, GoReleaser still generates and tests
the cask but skips tap upload, so GitHub releases remain available.

After the tap and secret are provisioned, every stable tag updates
`Casks/ollygarden.rb`. Prerelease tags generate the cask without replacing the
stable tap entry. Verify a published cask with:

```bash
brew update
brew info --cask ollygarden/tap/ollygarden
brew install --cask ollygarden/tap/ollygarden
ollygarden version
```
