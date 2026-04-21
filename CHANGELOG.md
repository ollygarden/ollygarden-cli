# Changelog

All notable changes to the ollygarden CLI are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

GoReleaser also appends commit-based release notes to each GitHub Release. This
file records the curated, human-readable summary.

## [Unreleased]

## [0.1.0] - 2026-04-21

Initial public release.

### Added
- `organization` command: show org tier, features, and score.
- `services` commands: `list`, `get`, `grouped`, `search`, `versions`, `insights`.
- `insights` commands: `list`, `get`, `summary`.
- `analytics services` command: per-service analytics.
- `webhooks` commands: `list`, `get`, `create`, `update`, `delete`, `test`, plus `deliveries list` and `deliveries get`.
- `version` command with JSON output, including git commit, build date, and Go runtime.
- Human and `--json` output modes, `--quiet` mode, `NO_COLOR` support.
- `install.sh` for macOS/Linux with checksum verification.
- Cross-platform builds for darwin/linux/windows × amd64/arm64 via GoReleaser.

[Unreleased]: https://github.com/ollygarden/ollygarden-cli/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/ollygarden/ollygarden-cli/releases/tag/v0.1.0
