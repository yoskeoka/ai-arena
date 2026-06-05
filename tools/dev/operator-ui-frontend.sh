#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
artifact_dir=${OPERATOR_UI_ARTIFACT_DIR:-"$repo_root/operator-ui/test-results"}
port=${OPERATOR_UI_FRONTEND_PORT:-4173}

case "$artifact_dir" in
  /*) ;;
  *) artifact_dir="$repo_root/operator-ui/${artifact_dir#./}" ;;
esac

log_path="$artifact_dir/frontend.log"

mkdir -p "$artifact_dir"
exec >"$log_path" 2>&1

cd "$repo_root/operator-ui"
pnpm_config_store_dir=/tmp/pnpm-store-ai-arena pnpm exec vite --host 127.0.0.1 --port "$port" --strictPort
