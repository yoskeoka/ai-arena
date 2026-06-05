#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
artifact_dir=${OPERATOR_UI_ARTIFACT_DIR:-"$repo_root/operator-ui/test-results"}
mode=${OPERATOR_UI_BACKEND_MODE:-file-backed}
port=${OPERATOR_UI_BACKEND_PORT:-10000}

case "$artifact_dir" in
  /*) ;;
  *) artifact_dir="$repo_root/operator-ui/${artifact_dir#./}" ;;
esac

log_path="$artifact_dir/backend.log"

mkdir -p "$artifact_dir"
exec >"$log_path" 2>&1

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
  real-local)
    export POSTGRES_MANAGED="${POSTGRES_MANAGED:-compose}"
    if [ "$POSTGRES_MANAGED" = "compose" ]; then
      make postgres-down
    fi
    make postgres-up
    export AI_ARENA_PG_TEST_DSN="${AI_ARENA_PG_TEST_DSN:-postgres://arena:arena@127.0.0.1:55432/arena_service?sslmode=disable}"
    export AI_ARENA_PG_ATLAS_DEV_DSN="${AI_ARENA_PG_ATLAS_DEV_DSN:-postgres://arena:arena@127.0.0.1:55432/postgres?sslmode=disable}"
    export ARENA_SERVICE_POSTGRES_DSN="${ARENA_SERVICE_POSTGRES_DSN:-$AI_ARENA_PG_TEST_DSN}"
    export ARENA_SERVICE_PRESET_CONFIG="${ARENA_SERVICE_PRESET_CONFIG:-./config/platform-service/presets.operator-ui-postgres.json}"
    export SEAWEED_MANAGED="${SEAWEED_MANAGED:-compose}"
    make postgres-schema-apply
    if make seaweed-bootstrap; then
      export ARENA_SERVICE_ARTIFACT_BACKEND="${ARENA_SERVICE_ARTIFACT_BACKEND:-r2}"
      export ARENA_SERVICE_ARTIFACT_R2_BUCKET="${ARENA_SERVICE_ARTIFACT_R2_BUCKET:-ai-arena-local}"
      export ARENA_SERVICE_ARTIFACT_R2_S3_ENDPOINT="${ARENA_SERVICE_ARTIFACT_R2_S3_ENDPOINT:-http://127.0.0.1:8333}"
      export ARENA_SERVICE_ARTIFACT_R2_ACCESS_KEY_ID="${ARENA_SERVICE_ARTIFACT_R2_ACCESS_KEY_ID:-admin}"
      export ARENA_SERVICE_ARTIFACT_R2_SECRET_ACCESS_KEY="${ARENA_SERVICE_ARTIFACT_R2_SECRET_ACCESS_KEY:-secret}"
    else
      echo "seaweed bootstrap failed; falling back to file-backed artifacts"
      unset ARENA_SERVICE_ARTIFACT_BACKEND
      unset ARENA_SERVICE_ARTIFACT_R2_BUCKET
      unset ARENA_SERVICE_ARTIFACT_R2_S3_ENDPOINT
      unset ARENA_SERVICE_ARTIFACT_R2_ACCESS_KEY_ID
      unset ARENA_SERVICE_ARTIFACT_R2_SECRET_ACCESS_KEY
      rm -rf "$repo_root/tmp/operator-ui-browser-postgres"
    fi
    ;;
  *)
    echo "unsupported OPERATOR_UI_BACKEND_MODE: $mode" >&2
    exit 1
    ;;
esac

make render-build
PORT="$port" make render-start
