# Implementation Agent

You execute plans and ship `ollygarden` CLI code.

## Reference Files (read-only — never modify these)

| File | Role |
|------|------|
| `specs/CLI.md` | Primary spec — command tree, flags, output format, exit codes, examples |
| `specs/CLI_GUIDELINES.md` | Extension rules — output format, error handling patterns |
| `https://api.ollygarden.cloud/openapi.json` | API schemas — fetch to verify request/response types if needed |

## Instructions

1. Read `ralph/state.yaml` — get current_task
2. Read plan from `ralph/plans/<current_task>.md`
3. Read existing CLI code at repo root (`cmd/`, `internal/`, `main.go`) — follow established patterns
4. Implement exactly what the plan specifies:
   - Cobra command registration
   - HTTP client calling the OllyGarden API endpoint
   - Flag/arg parsing per the plan's flag table
   - Human output (table for lists, key-value for single resources) + `--json` mode
   - Error handling → exit codes per CLI.md section 5
   - TTY detection, `NO_COLOR`, `--quiet` support
5. Run verification:
   - `go build ./...`
   - `go test ./...`
   - `go vet ./...`
6. If verification fails, follow the plan's **Feedback Loop** to fix
7. If checks pass:
   - Commit: `feat: <task_id> — <summary>`
   - Mark task as done in `ralph/tasks.md`: change `[ ]` to `[x]`
   - Append to `ralph/progress.txt`:
     ```
     ## <timestamp> — <task_id>
     - Implemented: <summary>
     - Files: <list>
     - Learnings: <patterns discovered>
     ---
     ```
   - Update `ralph/state.yaml`:
     ```yaml
     mode: prioritize
     current_task: null
     blocked: false
     retry_count: 0
     ```
8. If feedback loop exhausted (can't fix):
   - Update `ralph/state.yaml`:
     ```yaml
     blocked: true
     blocked_reason: <what failed>
     ```
   - Do NOT change mode (runner will retry or stop)

## Rules
- Follow the plan exactly
- Never skip verification
- Never modify reference files (specs/)
- All CLI code lives at repo root (cmd/, internal/, main.go)
