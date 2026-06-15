#!/bin/bash
set -euo pipefail

repo_root=$(cd "$(dirname -- "$0")/.." && pwd)
xdg_data_home=${XDG_DATA_HOME:-/tmp/.xdg-data-ai-arena}
pnpm_home=${PNPM_HOME:-/tmp/.pnpm-home-ai-arena}
store_dir=${PNPM_STORE_DIR:-/tmp/pnpm-store-ai-arena}

mkdir -p "$store_dir" "$xdg_data_home" "$pnpm_home"
export XDG_DATA_HOME="$xdg_data_home"
export PNPM_HOME="$pnpm_home"

"$repo_root/tools/dev/ensure-pnpm-install.sh" "$repo_root"

if [ "${1:-}" = "--all" ]; then
    shift
    mapfile -t files < <(git ls-files 'docs/**/*.md' 'docs/*.md')
else
    if [ "$#" -eq 0 ]; then
        echo "usage: $0 --all | <target.md>..." >&2
        exit 1
    fi
    files=("$@")
fi

if [ "${#files[@]}" -eq 0 ]; then
    echo "No Markdown files found."
    exit 0
fi

PNPM_STORE_DIR="$store_dir" \
  pnpm_config_store_dir="$store_dir" \
  pnpm exec textlint --rulesdir ./tools/textlint-rules --format json "${files[@]}"
