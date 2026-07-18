# Contributing

The OllyGarden CLI is publicly distributed and maintained by OllyGarden
employees with contributions from approved collaborators.

## Before opening a pull request

1. Start from the current default branch and keep the change focused.
2. Read [AGENTS.md](AGENTS.md). For command-surface changes, also read
   [specs/CLI.md](specs/CLI.md) and
   [specs/CLI_GUIDELINES.md](specs/CLI_GUIDELINES.md).
3. Use Conventional Commits, for example `feat(services): add tag filtering`.
4. Add focused tests for behavior changes. Commands must cover human and JSON
   output, validation, relevant API errors, and quiet mode.
5. Run:

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
