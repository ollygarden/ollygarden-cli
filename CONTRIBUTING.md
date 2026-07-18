# Contributing to the OllyGarden CLI

Thank you for your interest in contributing! Community contributions are
welcome.

## Community expectations

Participation is governed by OllyGarden's
[Code of Conduct](https://github.com/ollygarden/.github/blob/main/CODE_OF_CONDUCT.md).
Report suspected vulnerabilities privately through the repository's
[security policy](https://github.com/ollygarden/ollygarden-cli/security/policy),
not in a public issue. Project roles and decision making follow OllyGarden's
[governance policy](https://github.com/ollygarden/.github/blob/main/GOVERNANCE.md).

## Contributions from AI coding agents

Contributions authored with AI coding agents are welcome and held to the same
standards as any other change. A human contributor must review and take
responsibility for the result, be able to respond to feedback, and disclose
material agent involvement in the pull request description.

## Getting started

1. Search existing issues and pull requests for related work. For a large or
   potentially breaking change, open an issue before investing in an
   implementation.
2. Fork and clone the repository.
3. Create a focused branch from the current `main` branch.
4. Make and validate the change.
5. Open a focused pull request with a summary and test plan.

## Before opening a pull request

1. Read [AGENTS.md](AGENTS.md). For command-surface changes, also read
   [specs/CLI.md](specs/CLI.md) and
   [specs/CLI_GUIDELINES.md](specs/CLI_GUIDELINES.md).
2. Use [Conventional Commits](https://www.conventionalcommits.org/), for example
   `feat(services): add tag filtering`.
3. Add focused tests for behavior changes. Commands must cover human and JSON
   output, validation, relevant API errors, and quiet mode.
4. Run:

   ```bash
   gofmt -w <changed-go-files>
   go mod tidy
   git diff --exit-code -- go.mod go.sum
   go build ./...
   go vet ./...
   go test ./...
   go test -race ./...
   golangci-lint run
   ```

## Pull request expectations

- Explain user-visible behavior, output, exit-code, or compatibility changes.
- Update the CLI specifications and examples with command-surface changes.
- Include the exact validation commands run.
- Never commit API keys, saved credential files, customer data, or production
  responses. Tests must use synthetic values and local test servers.
- Resolve review threads and keep the branch current before requesting merge.

Maintainers review pull requests for correctness, scope, compatibility, and
maintainability. They may ask contributors to update a branch or split unrelated
changes. Pull requests are merged after required checks, review feedback, and
CLA requirements are satisfied.

## Contributor License Agreement

Before a first pull request can be merged, contributors must sign OllyGarden's
[Contributor License Agreement](https://github.com/ollygarden/.github/blob/main/CLA.md).
The CLA bot provides instructions on the pull request, and the signature applies
to future contributions across the organization.
