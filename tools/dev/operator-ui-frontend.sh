#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
artifact_dir=${OPERATOR_UI_ARTIFACT_DIR:-"$repo_root/operator-ui/test-results"}
port=${OPERATOR_UI_FRONTEND_PORT:-4173}
host=${OPERATOR_UI_FRONTEND_HOST:-}
log_to_file=${OPERATOR_UI_LOG_TO_FILE:-}

if [ -z "$log_to_file" ]; then
  if [ -n "${OPERATOR_UI_TEST_SCENARIO:-}" ]; then
    log_to_file=1
  else
    log_to_file=0
  fi
fi

if [ -z "$host" ]; then
  if [ -n "${OPERATOR_UI_TEST_SCENARIO:-}" ]; then
    host=127.0.0.1
  else
    host=localhost
  fi
fi

case "$artifact_dir" in
  /*) ;;
  *) artifact_dir="$repo_root/operator-ui/${artifact_dir#./}" ;;
esac

log_path="$artifact_dir/frontend.log"

mkdir -p "$artifact_dir"
if [ "$log_to_file" = "1" ]; then
  exec >"$log_path" 2>&1
fi

operator_ui_dir="$repo_root/operator-ui"
store_dir=${PNPM_STORE_DIR:-/tmp/pnpm-store-ai-arena}
xdg_data_home=${XDG_DATA_HOME:-/tmp/.xdg-data-ai-arena}
pnpm_home=${PNPM_HOME:-/tmp/.pnpm-home-ai-arena}

mkdir -p "$store_dir" "$xdg_data_home" "$pnpm_home"
export XDG_DATA_HOME="$xdg_data_home"
export PNPM_HOME="$pnpm_home"

"$repo_root/tools/dev/ensure-pnpm-install.sh" "$operator_ui_dir"

cd "$operator_ui_dir"
pnpm_config_store_dir="$store_dir" pnpm exec vite --host "$host" --port "$port" --strictPort
