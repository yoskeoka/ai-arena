#!/usr/bin/env bash
set -euo pipefail

# Builds the built-in games through the same WASI bundle boundary used by an
# official submission. The output ZIPs are fixtures for upload/registry/worker
# integration tests; they are not checked into the repository.

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
output_dir="${1:-$repo_root/.local/builtin-game-bundles}"
work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT

command -v zip >/dev/null || { echo "zip is required" >&2; exit 1; }
mkdir -p "$output_dir"

pack_game() {
  local name="$1" package="$2" game_id="$3" version="$4" ruleset="$5" args="$6"
  local bundle_dir="$work_dir/$name"
  mkdir -p "$bundle_dir"
  (cd "$repo_root" && GOOS=wasip1 GOARCH=wasm go build -o "$bundle_dir/module.wasm" "$package")
  cat >"$bundle_dir/manifest.json" <<EOF
{
  "schema_version": "arena-bundle/v1",
  "artifact_kind": "game",
  "game_id": "$game_id",
  "game_version": "$version",
  "game_master_protocol_version": "v1",
  "rulesets": [{"ruleset_version": "$ruleset", "player_count": 2, "max_active_bots_per_owner": 1}],
  "runtime": {"kind": "wasm-wasi", "module": "module.wasm", "args": $args}
}
EOF
  (cd "$bundle_dir" && zip -q -X "$output_dir/$name.arena-bundle.zip" manifest.json module.wasm)
}

pack_game "echo-count" "./cmd/echo-count-gamemaster" "echo-count" "2.0.0" "phase2-simultaneous-3turn" '["--game-version", "2.0.0", "--ruleset", "phase2-simultaneous-3turn"]'
pack_game "janken" "./cmd/janken-gamemaster" "janken" "2.1.0" "regular" '[]'

echo "wrote game bundles to $output_dir"
