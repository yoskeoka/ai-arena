#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
artifact_dir=${OPERATOR_UI_ARTIFACT_DIR:-"$repo_root/operator-ui/test-results"}
port=${OPERATOR_UI_AUTH_MOCK_PORT:-10001}
postgres_dsn=${ARENA_SERVICE_POSTGRES_DSN:-${AI_ARENA_PG_TEST_DSN:-postgres://arena:arena@127.0.0.1:55432/arena_service?sslmode=disable}}
log_to_file=${OPERATOR_UI_LOG_TO_FILE:-}

if [ -z "$log_to_file" ]; then
  if [ -n "${OPERATOR_UI_TEST_SCENARIO:-}" ]; then
    log_to_file=1
  else
    log_to_file=0
  fi
fi

case "$artifact_dir" in
  /*) ;;
  *) artifact_dir="$repo_root/operator-ui/${artifact_dir#./}" ;;
esac

log_path="$artifact_dir/github-oauth-test-double.log"
mkdir -p "$artifact_dir"
if [ "$log_to_file" = "1" ]; then
  exec >"$log_path" 2>&1
fi

cd "$repo_root"

export GOPATH="${GOPATH:-/tmp/ai-arena-operator-ui-go}"
export GOMODCACHE="${GOMODCACHE:-$GOPATH/pkg/mod}"
export GOCACHE="${GOCACHE:-/tmp/ai-arena-operator-ui-go-build}"
mkdir -p "$GOPATH" "$GOMODCACHE" "$GOCACHE"

exec go run ./cmd/github-oauth-test-double --listen-addr "127.0.0.1:$port" --postgres-dsn "$postgres_dsn"
