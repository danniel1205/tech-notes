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
    # --autostash: local edits anywhere in the repo are stashed and re-applied around the pull.
    git pull --rebase --autostash origin "$BRANCH"
    echo "Up to date. Latest sessions:"
    ls -1 sessions/ | tail -n 3
    ;;
  end)
    MSG="${2:-session: $MACHINE $(date +%F)}"
    git add -- "$DIR"
    if git diff --cached --quiet -- "$DIR"; then
      echo "Nothing to commit."
    else
      # --no-verify: skip the repo's pre-commit hook (it regenerates README.md TOC,
      # which is unrelated to sync commits and leaves README.md dirty).
      git commit --no-verify -m "$MSG" -- "$DIR"
    fi
    git pull --rebase --autostash origin "$BRANCH"
    git push origin "$BRANCH"
    echo "[$MACHINE] Pushed to '$BRANCH'. The other machine can now run: ./sync.sh start"
    ;;
  *)
    echo "Usage: $0 start | end \"commit message\"" >&2
    exit 1
    ;;
esac
