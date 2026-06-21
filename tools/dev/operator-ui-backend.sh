#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
artifact_dir=${OPERATOR_UI_ARTIFACT_DIR:-"$repo_root/operator-ui/test-results"}
mode=${OPERATOR_UI_BACKEND_MODE:-file-backed}
port=${OPERATOR_UI_BACKEND_PORT:-10000}
frontend_port=${OPERATOR_UI_FRONTEND_PORT:-4173}
frontend_host=${OPERATOR_UI_FRONTEND_HOST:-127.0.0.1}
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

log_path="$artifact_dir/backend.log"

mkdir -p "$artifact_dir"
if [ "$log_to_file" = "1" ]; then
  exec >"$log_path" 2>&1
fi

cd "$repo_root"

echo "operator-ui backend mode: $mode"
echo "artifact dir: $artifact_dir"

export GOPATH="${GOPATH:-/tmp/ai-arena-operator-ui-go}"
export GOMODCACHE="${GOMODCACHE:-$GOPATH/pkg/mod}"
export GOCACHE="${GOCACHE:-/tmp/ai-arena-operator-ui-go-build}"
mkdir -p "$GOPATH" "$GOMODCACHE" "$GOCACHE"

case "$mode" in
  file-backed)
    rm -rf "$repo_root/tmp/operator-ui-browser-file-backed"
    export ARENA_SERVICE_PRESET_CONFIG="${ARENA_SERVICE_PRESET_CONFIG:-./config/platform-service/presets.operator-ui-file-backed.json}"
    ;;
  auth-mock)
    if [ "${OPERATOR_UI_RESET_POSTGRES:-0}" = "1" ]; then
      make postgres-down
      make postgres-up
    fi
    rm -rf "$repo_root/tmp/operator-ui-browser-file-backed"
    export AI_ARENA_PG_TEST_DSN="${AI_ARENA_PG_TEST_DSN:-postgres://arena:arena@127.0.0.1:55432/arena_service?sslmode=disable}"
    export AI_ARENA_PG_ATLAS_DEV_DSN="${AI_ARENA_PG_ATLAS_DEV_DSN:-postgres://arena:arena@127.0.0.1:55432/postgres?sslmode=disable}"
    export ARENA_SERVICE_POSTGRES_DSN="${ARENA_SERVICE_POSTGRES_DSN:-$AI_ARENA_PG_TEST_DSN}"
    export ARENA_SERVICE_PRESET_CONFIG="${ARENA_SERVICE_PRESET_CONFIG:-./config/platform-service/presets.operator-ui-file-backed.json}"
    export ARENA_GITHUB_OAUTH_CLIENT_ID="${ARENA_GITHUB_OAUTH_CLIENT_ID:-playwright-client-id}"
    export ARENA_GITHUB_OAUTH_CLIENT_SECRET="${ARENA_GITHUB_OAUTH_CLIENT_SECRET:-playwright-client-secret}"
    export ARENA_AUTH_GITHUB_TEST_DOUBLE="${ARENA_AUTH_GITHUB_TEST_DOUBLE:-1}"
    export ARENA_AUTH_GITHUB_PROVIDER_AUTH_URL="${ARENA_AUTH_GITHUB_PROVIDER_AUTH_URL:-http://localhost:$port/_internal/test-auth/github/authorize}"
    export ARENA_AUTH_GITHUB_PROVIDER_TOKEN_URL="${ARENA_AUTH_GITHUB_PROVIDER_TOKEN_URL:-http://localhost:$port/_internal/test-auth/github/token}"
    export ARENA_AUTH_GITHUB_PROVIDER_USER_URL="${ARENA_AUTH_GITHUB_PROVIDER_USER_URL:-http://localhost:$port/_internal/test-auth/github/user}"
    export ARENA_AUTH_ALLOWED_RETURN_ORIGINS="${ARENA_AUTH_ALLOWED_RETURN_ORIGINS:-http://${frontend_host}:${frontend_port},http://127.0.0.1:${frontend_port},http://localhost:${frontend_port},http://127.0.0.1:5173,http://localhost:5173}"
    make postgres-schema-apply
    ;;
  postgres)
    export AI_ARENA_PG_TEST_DSN="${AI_ARENA_PG_TEST_DSN:-postgres://arena:arena@127.0.0.1:5432/arena_service?sslmode=disable}"
    export AI_ARENA_PG_ATLAS_DEV_DSN="${AI_ARENA_PG_ATLAS_DEV_DSN:-postgres://arena:arena@127.0.0.1:5432/postgres?sslmode=disable}"
    export ARENA_SERVICE_POSTGRES_DSN="${ARENA_SERVICE_POSTGRES_DSN:-$AI_ARENA_PG_TEST_DSN}"
    export ARENA_SERVICE_PRESET_CONFIG="${ARENA_SERVICE_PRESET_CONFIG:-./config/platform-service/presets.operator-ui-postgres.json}"
    export ARENA_SERVICE_ARTIFACT_BACKEND="${ARENA_SERVICE_ARTIFACT_BACKEND:-r2}"
    export ARENA_SERVICE_ARTIFACT_R2_BUCKET="${ARENA_SERVICE_ARTIFACT_R2_BUCKET:-ai-arena-local}"
    export ARENA_SERVICE_ARTIFACT_R2_S3_ENDPOINT="${ARENA_SERVICE_ARTIFACT_R2_S3_ENDPOINT:-http://127.0.0.1:8333}"
    export ARENA_SERVICE_ARTIFACT_R2_ACCESS_KEY_ID="${ARENA_SERVICE_ARTIFACT_R2_ACCESS_KEY_ID:-admin}"
    export ARENA_SERVICE_ARTIFACT_R2_SECRET_ACCESS_KEY="${ARENA_SERVICE_ARTIFACT_R2_SECRET_ACCESS_KEY:-secret}"
    export SEAWEED_MANAGED="${SEAWEED_MANAGED:-compose}"
    if [ "$SEAWEED_MANAGED" = "compose" ]; then
      rm -rf "$repo_root/.local/seaweed"
    fi
    make postgres-schema-apply
    make seaweed-bootstrap
    ;;
  local|real-local)
    export AI_ARENA_PG_TEST_DSN="${AI_ARENA_PG_TEST_DSN:-postgres://arena:arena@127.0.0.1:55432/arena_service?sslmode=disable}"
    export AI_ARENA_PG_ATLAS_DEV_DSN="${AI_ARENA_PG_ATLAS_DEV_DSN:-postgres://arena:arena@127.0.0.1:55432/postgres?sslmode=disable}"
    export ARENA_SERVICE_POSTGRES_DSN="${ARENA_SERVICE_POSTGRES_DSN:-$AI_ARENA_PG_TEST_DSN}"
    export ARENA_SERVICE_PRESET_CONFIG="${ARENA_SERVICE_PRESET_CONFIG:-./config/platform-service/presets.operator-ui-postgres.json}"
    export ARENA_SERVICE_ARTIFACT_BACKEND="${ARENA_SERVICE_ARTIFACT_BACKEND:-r2}"
    export ARENA_SERVICE_ARTIFACT_R2_BUCKET="${ARENA_SERVICE_ARTIFACT_R2_BUCKET:-ai-arena-local}"
    export ARENA_SERVICE_ARTIFACT_R2_S3_ENDPOINT="${ARENA_SERVICE_ARTIFACT_R2_S3_ENDPOINT:-http://127.0.0.1:8333}"
    export ARENA_SERVICE_ARTIFACT_R2_ACCESS_KEY_ID="${ARENA_SERVICE_ARTIFACT_R2_ACCESS_KEY_ID:-admin}"
    export ARENA_SERVICE_ARTIFACT_R2_SECRET_ACCESS_KEY="${ARENA_SERVICE_ARTIFACT_R2_SECRET_ACCESS_KEY:-secret}"
    ;;
  *)
    echo "unsupported OPERATOR_UI_BACKEND_MODE: $mode" >&2
    exit 1
    ;;
esac

make render-build
if [ -z "${OPERATOR_UI_TEST_SCENARIO:-}" ] && command -v direnv >/dev/null 2>&1; then
  PORT="$port" direnv exec "$repo_root" make render-start
else
  PORT="$port" make render-start
fi
