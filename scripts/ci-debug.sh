#!/usr/bin/env sh
set -eu

run_id="${1:-${RUN_ID:-}}"

if ! command -v gh >/dev/null 2>&1; then
  echo "gh is required. Install GitHub CLI and run gh auth login first." >&2
  exit 1
fi

if [ -z "$run_id" ]; then
  run_id="$(gh run list --limit 1 --json databaseId --jq '.[0].databaseId')"
fi

if [ -z "$run_id" ] || [ "$run_id" = "null" ]; then
  echo "No GitHub Actions run found." >&2
  exit 1
fi

log_file="${CI_DEBUG_LOG:-/tmp/ai-chat-ci-${run_id}.log}"

echo "Run: $run_id"
echo
echo "Jobs:"
gh run view "$run_id" --json jobs --jq '.jobs[] | [.name,.conclusion,.databaseId] | @tsv'

echo
echo "Saving failed-step log to: $log_file"
gh run view "$run_id" --log-failed > "$log_file"

echo
echo "Likely failure lines:"
if ! grep -n -E -- '--- FAIL|FAIL:|expected|unexpected|Error:|error:|Process completed|panic|timed out|not found' "$log_file"; then
  echo "No common failure markers found. Open the saved log directly:"
  echo "$log_file"
fi

echo
echo "Tip: inspect more context with:"
echo "  grep -n -C 5 -E -- '--- FAIL|Error:|Process completed' $log_file"
