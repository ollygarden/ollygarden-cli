#!/bin/bash
set -e

cd "$(dirname "$0")/.."

MAX_RETRIES=3

cleanup() {
  echo "[ralph] stopped (iteration $iteration)"
}
trap cleanup EXIT

check_control() {
  [ ! -f ralph/control ] && return
  local cmd=$(cat ralph/control)
  rm -f ralph/control

  case "$cmd" in
    stop)
      echo "[ralph] stop requested"
      exit 0
      ;;
    pause)
      echo "[ralph] paused — write 'resume' to ralph/control"
      while [ ! -f ralph/control ] || [ "$(cat ralph/control 2>/dev/null)" != "resume" ]; do
        sleep 5
      done
      rm -f ralph/control
      echo "[ralph] resumed"
      ;;
  esac
}

format_duration() {
  local s=$1
  if [ "$s" -lt 60 ]; then echo "${s}s"
  elif [ "$s" -lt 3600 ]; then echo "$((s/60))m $((s%60))s"
  else echo "$((s/3600))h $((s%3600/60))m"
  fi
}

echo "[ralph] starting"

while true; do
  check_control

  mode=$(yq -r '.mode' ralph/state.yaml)
  blocked=$(yq -r '.blocked' ralph/state.yaml)
  retries=$(yq -r '.retry_count // 0' ralph/state.yaml)
  task=$(yq -r '.current_task // "none"' ralph/state.yaml)
  iteration=$(yq -r '.iteration // 0' ralph/state.yaml)

  # Handle blocked state
  if [ "$blocked" = "true" ]; then
    if [ "$retries" -ge "$MAX_RETRIES" ]; then
      echo "[ralph] FAILED: max retries ($MAX_RETRIES) exceeded on task=$task mode=$mode"
      exit 1
    fi
    echo "[ralph] retry $((retries+1))/$MAX_RETRIES on task=$task"
    yq -i ".retry_count = $((retries+1))" ralph/state.yaml
    yq -i '.blocked = false' ralph/state.yaml
  fi

  echo "[ralph] === iteration=$iteration mode=$mode task=$task ==="

  start_time=$(date +%s)
  git_before=$(git rev-parse HEAD 2>/dev/null || echo "none")

  if cat "ralph/prompts/${mode}.md" | claude \
    --permission-mode acceptEdits \
    --model opus; then

    duration=$(format_duration $(($(date +%s) - start_time)))
    git_after=$(git rev-parse HEAD 2>/dev/null || echo "none")

    if [ "$git_before" != "$git_after" ]; then
      commit_msg=$(git log --format=%s -1 2>/dev/null || echo "")
      files_changed=$(git diff --name-only "$git_before" "$git_after" 2>/dev/null | wc -l | tr -d ' ')
      echo "[ralph] DONE mode=$mode duration=$duration commit=\"$commit_msg\" files=$files_changed"
    else
      echo "[ralph] DONE mode=$mode duration=$duration (no commits)"
    fi
  else
    exit_code=$?
    duration=$(format_duration $(($(date +%s) - start_time)))
    echo "[ralph] ERROR mode=$mode exit=$exit_code duration=$duration"
    yq -i '.blocked = true' ralph/state.yaml
  fi

  yq -i '.iteration = (.iteration // 0) + 1' ralph/state.yaml
  check_control
  sleep 2
done
