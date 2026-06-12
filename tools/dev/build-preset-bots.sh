#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
output_dir="$repo_root/config/platform-service/preset-bots"
binary_path="$output_dir/echo-ai-2turn"
manifest_path="$binary_path.arena.json"
go_cmd=${GO:-go}

cd "$repo_root"
mkdir -p "$output_dir"

"$go_cmd" build -o "$binary_path" ./testdata/ai/echo/echo-ai

cat >"$manifest_path" <<'EOF'
{
  "ai_id": "echo-ai-2turn",
  "protocol": {
    "transport": "stdio-jsonrpc-ndjson",
    "game_id": "echo-count",
    "game_version": "2.0.0",
    "ruleset_version": "phase2-simultaneous-2turn"
  },
  "runtime": {
    "kind": "local-subprocess",
    "command": ["./config/platform-service/preset-bots/echo-ai-2turn"]
  }
}
EOF
