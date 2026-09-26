#!/usr/bin/env bash
# Sync the AI Infra Zero to Hero shared memory between machines.
#   ./sync.sh start          -> pull latest changes
#   ./sync.sh end "message"  -> commit THIS FOLDER ONLY and push
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

REPO_ROOT="$(git rev-parse --show-toplevel)"
REL_DIR="$(realpath --relative-to="$REPO_ROOT" "$DIR" 2>/dev/null || python3 -c 'import os,sys; print(os.path.relpath(sys.argv[1], sys.argv[2]))' "$DIR" "$REPO_ROOT")"
BRANCH="$(git rev-parse --abbrev-ref HEAD)"

if [[ "$(uname)" == "Darwin" ]]; then MACHINE="mac"; else MACHINE="cloudtop"; fi

case "${1:-}" in
  start)
    echo "[$MACHINE] Pulling latest on branch '$BRANCH'..."
    if ! git diff --quiet -- "$DIR" || ! git diff --cached --quiet -- "$DIR"; then
      echo "Uncommitted local changes in $REL_DIR. Stashing them before pull."
      git stash push -m "sync.sh auto-stash $(date +%F-%T)" -- "$DIR"
      git pull --rebase origin "$BRANCH"
      git stash pop
    else
      git pull --rebase origin "$BRANCH"
    fi
    echo "Up to date. Latest sessions:"
    ls -1 sessions/ | tail -n 3
    ;;
  end)
    MSG="${2:-session: $MACHINE $(date +%F)}"
    git add -- "$DIR"
    if git diff --cached --quiet; then
      echo "Nothing to commit."
    else
      git commit -m "$MSG" -- "$DIR"
    fi
    git pull --rebase origin "$BRANCH"
    git push origin "$BRANCH"
    echo "[$MACHINE] Pushed to '$BRANCH'. The other machine can now run: ./sync.sh start"
    ;;
  *)
    echo "Usage: $0 start | end \"commit message\"" >&2
    exit 1
    ;;
esac
