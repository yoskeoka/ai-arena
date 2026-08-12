#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
release_base="https://github.com/yoskeoka/reversi-ai-arena/releases/download/v0.1.0"
game_bundle="reversi-game-v0.1.0.arena.zip"
ai_bundle="reversi-rust-reference-ai-v0.1.0.arena.zip"
expected_game_digest="98bd46609016dc763bcbfff747c6705c7f1608a86d164d77b38db208d0d1c0df"
expected_ai_digest="15ec6133f92002f68f291b39df69ac9beeb177debc93dd1d9c725c77db042326"
work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT

command -v curl >/dev/null || { echo "curl is required" >&2; exit 1; }
command -v sha256sum >/dev/null || { echo "sha256sum is required" >&2; exit 1; }

for asset in SHA256SUMS "$game_bundle" "$ai_bundle"; do
  curl --fail --location --silent --show-error "$release_base/$asset" --output "$work_dir/$asset"
done

(cd "$work_dir" && sha256sum --check SHA256SUMS)

test "$(sha256sum "$work_dir/$game_bundle" | awk '{print $1}')" = "$expected_game_digest"
test "$(sha256sum "$work_dir/$ai_bundle" | awk '{print $1}')" = "$expected_ai_digest"

output_dir="$work_dir/output"
(cd "$repo_root" && go run ./cmd/arena-runner \
  --game-master-bundle "$work_dir/$game_bundle" \
  --player-bundle "black=$work_dir/$ai_bundle" \
  --player-bundle "white=$work_dir/$ai_bundle" \
  --match-id reversi-release-v0.1.0 \
  --output-dir "$output_dir" \
  --log-output none)

test -f "$output_dir/reversi-release-v0.1.0/result-summary.json"
echo "verified Reversi v0.1.0 release artifact match"
