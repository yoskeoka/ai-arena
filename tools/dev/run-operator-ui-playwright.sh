#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
operator_ui_dir="$repo_root/operator-ui"
store_dir=${PNPM_STORE_DIR:-/tmp/pnpm-store-ai-arena}
xdg_data_home=${XDG_DATA_HOME:-/tmp/.xdg-data-ai-arena}
pnpm_home=${PNPM_HOME:-/tmp/.pnpm-home-ai-arena}

mkdir -p "$store_dir" "$xdg_data_home" "$pnpm_home"
export XDG_DATA_HOME="$xdg_data_home"
export PNPM_HOME="$pnpm_home"

"$repo_root/tools/dev/ensure-pnpm-install.sh" "$operator_ui_dir"
"$repo_root/tools/dev/ensure-operator-ui-playwright-browser.sh"

cd "$operator_ui_dir"
export pnpm_config_store_dir="$store_dir"
exec pnpm exec playwright test "$@"
