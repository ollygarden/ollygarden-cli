# Ralph — Autonomous Task Loop

Runs Claude agents in a `prioritize → plan → implement` loop to work through a task list.

## Quick Start

```bash
# 1. Add tasks to ralph/tasks.md
# 2. Run the loop
bash ralph/run.sh

# Or run in background
nohup bash ralph/run.sh > ralph/output.log 2>&1 &
tail -f ralph/output.log
```

## Control

```bash
echo stop > ralph/control     # stop after current agent finishes
echo pause > ralph/control    # pause (write 'resume' to continue)
echo resume > ralph/control   # resume from pause
Ctrl+C                        # stop immediately
```

## Task Format

Edit `ralph/tasks.md`:

```markdown
# Tasks
- [ ] **task-1**: Add rate limiting middleware
- [ ] **task-2**: Implement webhook retry logic
- [x] **task-3**: Already done (skipped)
```

Ralph picks the first unchecked `[ ]` task each cycle.

## How It Works

```
prioritize → plan → implement → prioritize → ...
```

1. **Prioritize**: Picks next `[ ]` task from `tasks.md`, sets it as `current_task`
2. **Plan**: Explores codebase, writes plan to `ralph/plans/<task-id>.md`
3. **Implement**: Executes plan, runs tests, commits, marks task `[x]`, logs to `progress.txt`

State is tracked in `ralph/state.yaml`. On failure, retries up to 3 times before exiting.

## Files

| File | Purpose |
|------|---------|
| `run.sh` | Main loop script |
| `state.yaml` | Current mode, task, iteration, blocked state |
| `tasks.md` | Task backlog (you edit this) |
| `progress.txt` | Append-only completion log |
| `plans/` | Per-task implementation plans (generated) |
| `prompts/` | Agent prompts (one per mode) |

## Dependencies

- `claude` CLI
- `yq` (YAML processor)
- `git`

## Reset State

```bash
# Reset to initial state (keeps tasks.md intact)
cat > ralph/state.yaml << 'EOF'
mode: prioritize
current_task: null
iteration: 0
blocked: false
retry_count: 0
EOF
rm -f ralph/control
```
