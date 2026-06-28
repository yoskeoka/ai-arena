#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
operator_ui_dir="$repo_root/operator-ui"
store_dir=${PNPM_STORE_DIR:-/tmp/pnpm-store-ai-arena}
xdg_data_home=${XDG_DATA_HOME:-/tmp/.xdg-data-ai-arena}
pnpm_home=${PNPM_HOME:-/tmp/.pnpm-home-ai-arena}
skip_bootstrap=${OPERATOR_UI_SKIP_BROWSER_BOOTSTRAP:-0}

mkdir -p "$store_dir" "$xdg_data_home" "$pnpm_home"
export XDG_DATA_HOME="$xdg_data_home"
export PNPM_HOME="$pnpm_home"

"$repo_root/tools/dev/ensure-pnpm-install.sh" "$operator_ui_dir"

if [ "$skip_bootstrap" = "1" ]; then
  echo "skipping Playwright browser bootstrap; expecting workflow-managed browser runtime"
  exit 0
fi

browser_path=$(
  cd "$operator_ui_dir"
  pnpm_config_store_dir="$store_dir" pnpm exec node -e 'const { chromium } = require("@playwright/test"); process.stdout.write(chromium.executablePath())'
)

if [ -n "$browser_path" ] && [ -x "$browser_path" ]; then
  exit 0
fi

echo "bootstrapping Playwright Chromium for operator-ui"
if ! (
  cd "$operator_ui_dir"
  pnpm_config_store_dir="$store_dir" pnpm exec playwright install chromium
); then
  echo "Playwright browser bootstrap failed. See docs/development/operator-ui-local-verification.md for manual debugging." >&2
  exit 1
fi
