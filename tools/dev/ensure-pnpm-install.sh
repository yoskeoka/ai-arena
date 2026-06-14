#!/bin/sh
set -eu

target_dir=${1:?usage: ensure-pnpm-install.sh <target-dir>}
store_dir=${PNPM_STORE_DIR:-/tmp/pnpm-store-ai-arena}
xdg_data_home=${XDG_DATA_HOME:-/tmp/.xdg-data-ai-arena}
pnpm_home=${PNPM_HOME:-/tmp/.pnpm-home-ai-arena}

mkdir -p "$store_dir" "$xdg_data_home" "$pnpm_home"
export XDG_DATA_HOME="$xdg_data_home"
export PNPM_HOME="$pnpm_home"

if [ -d "$target_dir/node_modules/.pnpm" ]; then
  exit 0
fi

echo "bootstrapping pnpm dependencies in $target_dir"
cd "$target_dir"
pnpm_config_store_dir="$store_dir" pnpm install --frozen-lockfile
