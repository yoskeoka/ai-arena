#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
artifact_dir=${OPERATOR_UI_ARTIFACT_DIR:-"$repo_root/operator-ui/test-results"}
mode=${OPERATOR_UI_BACKEND_MODE:-file-backed}
port=${OPERATOR_UI_BACKEND_PORT:-10000}
frontend_port=${OPERATOR_UI_FRONTEND_PORT:-4173}
frontend_host=${OPERATOR_UI_FRONTEND_HOST:-127.0.0.1}
auth_mock_port=${OPERATOR_UI_AUTH_MOCK_PORT:-10001}
oidc_mock_port=${OPERATOR_UI_OIDC_MOCK_PORT:-10002}
log_to_file=${OPERATOR_UI_LOG_TO_FILE:-}
reset_postgres=${OPERATOR_UI_RESET_POSTGRES:-0}

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

go_path="${GOPATH:-/tmp/ai-arena-operator-ui-go}"
go_path="${go_path%%:*}"
export GOPATH="$go_path"
export GOMODCACHE="${GOMODCACHE:-$GOPATH/pkg/mod}"
export GOCACHE="${GOCACHE:-/tmp/ai-arena-operator-ui-go-build}"
mkdir -p "$GOPATH" "$GOMODCACHE" "$GOCACHE"

if [ -n "${OPERATOR_UI_TEST_SCENARIO:-}" ]; then
  game_bundle_dir="${OPERATOR_UI_GAME_BUNDLE_DIR:-$repo_root/.local/operator-ui-game-bundles}"
  "$repo_root/tools/dev/package-builtin-game-bundles.sh" "$game_bundle_dir"
  export OPERATOR_UI_BUNDLE_FIXTURE_DIR="${OPERATOR_UI_BUNDLE_FIXTURE_DIR:-$game_bundle_dir}"
fi

if [ "${OPERATOR_UI_TEST_AUTH:-0}" = "1" ]; then
  export ARENA_GITHUB_OAUTH_CLIENT_ID="${ARENA_GITHUB_OAUTH_CLIENT_ID:-playwright-client-id}"
  export ARENA_GITHUB_OAUTH_CLIENT_SECRET="${ARENA_GITHUB_OAUTH_CLIENT_SECRET:-playwright-client-secret}"
  export ARENA_AUTH_GITHUB_PROVIDER_OAUTH_BASE_URL="${ARENA_AUTH_GITHUB_PROVIDER_OAUTH_BASE_URL:-http://127.0.0.1:${auth_mock_port}}"
  export ARENA_AUTH_GITHUB_PROVIDER_API_BASE_URL="${ARENA_AUTH_GITHUB_PROVIDER_API_BASE_URL:-http://127.0.0.1:${auth_mock_port}}"
  export ARENA_AUTH_ALLOWED_RETURN_ORIGINS="${ARENA_AUTH_ALLOWED_RETURN_ORIGINS:-http://${frontend_host}:${frontend_port},http://127.0.0.1:${frontend_port},http://localhost:${frontend_port},http://127.0.0.1:5173,http://localhost:5173}"
fi

echo "operator-ui backend mode: $mode"
echo "artifact dir: $artifact_dir"

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
    export PORT="${PORT:-$port}"
    export ARENA_GITHUB_OAUTH_CLIENT_ID="${ARENA_GITHUB_OAUTH_CLIENT_ID:-playwright-client-id}"
    export ARENA_GITHUB_OAUTH_CLIENT_SECRET="${ARENA_GITHUB_OAUTH_CLIENT_SECRET:-playwright-client-secret}"
    export ARENA_AUTH_GITHUB_PROVIDER_OAUTH_BASE_URL="${ARENA_AUTH_GITHUB_PROVIDER_OAUTH_BASE_URL:-http://127.0.0.1:${auth_mock_port}}"
    export ARENA_AUTH_GITHUB_PROVIDER_API_BASE_URL="${ARENA_AUTH_GITHUB_PROVIDER_API_BASE_URL:-http://127.0.0.1:${auth_mock_port}}"
    export ARENA_AUTH_ALLOWED_RETURN_ORIGINS="${ARENA_AUTH_ALLOWED_RETURN_ORIGINS:-http://${frontend_host}:${frontend_port},http://127.0.0.1:${frontend_port},http://localhost:${frontend_port},http://127.0.0.1:5173,http://localhost:5173}"
    make postgres-schema-apply
    ;;
  oidc-mock)
    if [ "${OPERATOR_UI_RESET_POSTGRES:-0}" = "1" ]; then
      make postgres-down
      make postgres-up
    fi
    export AI_ARENA_PG_TEST_DSN="${AI_ARENA_PG_TEST_DSN:-postgres://arena:arena@127.0.0.1:55432/arena_service?sslmode=disable}"
    export AI_ARENA_PG_ATLAS_DEV_DSN="${AI_ARENA_PG_ATLAS_DEV_DSN:-postgres://arena:arena@127.0.0.1:55432/postgres?sslmode=disable}"
    export ARENA_SERVICE_POSTGRES_DSN="${ARENA_SERVICE_POSTGRES_DSN:-$AI_ARENA_PG_TEST_DSN}"
    export ARENA_SERVICE_PRESET_CONFIG="${ARENA_SERVICE_PRESET_CONFIG:-./config/platform-service/presets.operator-ui-file-backed.json}"
    export PORT="${PORT:-$port}"
    export ARENA_GITHUB_OAUTH_CLIENT_ID="${ARENA_GITHUB_OAUTH_CLIENT_ID:-playwright-client-id}"
    export ARENA_GITHUB_OAUTH_CLIENT_SECRET="${ARENA_GITHUB_OAUTH_CLIENT_SECRET:-playwright-client-secret}"
    export ARENA_AUTH_GITHUB_PROVIDER_OAUTH_BASE_URL="${ARENA_AUTH_GITHUB_PROVIDER_OAUTH_BASE_URL:-http://127.0.0.1:${auth_mock_port}}"
    export ARENA_AUTH_GITHUB_PROVIDER_API_BASE_URL="${ARENA_AUTH_GITHUB_PROVIDER_API_BASE_URL:-http://127.0.0.1:${auth_mock_port}}"
    export ARENA_AUTH_ALLOWED_RETURN_ORIGINS="${ARENA_AUTH_ALLOWED_RETURN_ORIGINS:-http://${frontend_host}:${frontend_port},http://127.0.0.1:${frontend_port},http://localhost:${frontend_port}}"
    make postgres-schema-apply
    go run ./cmd/local-oidc-test-provider --listen-addr "127.0.0.1:${oidc_mock_port}" --issuer "http://127.0.0.1:${oidc_mock_port}" --postgres-dsn "$ARENA_SERVICE_POSTGRES_DSN" >"$artifact_dir/local-oidc-test-provider.log" 2>&1 &
    oidc_pid=$!
    trap 'kill "$oidc_pid" 2>/dev/null || true' EXIT INT TERM
    until curl -fsS "http://127.0.0.1:${oidc_mock_port}/.well-known/openid-configuration" >/dev/null; do sleep 1; done
    # Native loopback registrations omit the port so the provider accepts the runtime callback port.
    oidc_registration=$(curl -fsS -X POST "http://127.0.0.1:${oidc_mock_port}/register" -H 'Content-Type: application/json' --data "{\"application_type\":\"native\",\"client_name\":\"AI Arena local OIDC\",\"redirect_uris\":[\"http://127.0.0.1/auth/local-oidc/callback\"],\"grant_types\":[\"authorization_code\"],\"response_types\":[\"code\"],\"scope\":\"openid profile email\",\"token_endpoint_auth_method\":\"client_secret_post\"}")
    export ARENA_AUTH_LOCAL_OIDC_ISSUER="http://127.0.0.1:${oidc_mock_port}"
    export ARENA_AUTH_LOCAL_OIDC_ENABLED=1
    export ARENA_AUTH_LOCAL_OIDC_CLIENT_ID=$(printf '%s' "$oidc_registration" | sed -n 's/.*"client_id":"\([^"]*\)".*/\1/p')
    export ARENA_AUTH_LOCAL_OIDC_CLIENT_SECRET=$(printf '%s' "$oidc_registration" | sed -n 's/.*"client_secret":"\([^"]*\)".*/\1/p')
    test -n "$ARENA_AUTH_LOCAL_OIDC_CLIENT_ID" && test -n "$ARENA_AUTH_LOCAL_OIDC_CLIENT_SECRET"
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
    if [ "$reset_postgres" = "1" ]; then
      make postgres-down
    fi
    make postgres-up
    export AI_ARENA_PG_TEST_DSN="${AI_ARENA_PG_TEST_DSN:-postgres://arena:arena@127.0.0.1:55432/arena_service?sslmode=disable}"
    export AI_ARENA_PG_ATLAS_DEV_DSN="${AI_ARENA_PG_ATLAS_DEV_DSN:-postgres://arena:arena@127.0.0.1:55432/postgres?sslmode=disable}"
    export ARENA_SERVICE_POSTGRES_DSN="${ARENA_SERVICE_POSTGRES_DSN:-$AI_ARENA_PG_TEST_DSN}"
    export ARENA_SERVICE_PRESET_CONFIG="${ARENA_SERVICE_PRESET_CONFIG:-./config/platform-service/presets.operator-ui-postgres.json}"
    export ARENA_SERVICE_ARTIFACT_BACKEND="${ARENA_SERVICE_ARTIFACT_BACKEND:-r2}"
    export ARENA_SERVICE_ARTIFACT_R2_BUCKET="${ARENA_SERVICE_ARTIFACT_R2_BUCKET:-ai-arena-local}"
    export ARENA_SERVICE_ARTIFACT_R2_S3_ENDPOINT="${ARENA_SERVICE_ARTIFACT_R2_S3_ENDPOINT:-http://127.0.0.1:8333}"
    export ARENA_SERVICE_ARTIFACT_R2_ACCESS_KEY_ID="${ARENA_SERVICE_ARTIFACT_R2_ACCESS_KEY_ID:-admin}"
    export ARENA_SERVICE_ARTIFACT_R2_SECRET_ACCESS_KEY="${ARENA_SERVICE_ARTIFACT_R2_SECRET_ACCESS_KEY:-secret}"
    make postgres-schema-apply
    go run ./cmd/local-oidc-test-provider --listen-addr "127.0.0.1:${oidc_mock_port}" --issuer "http://127.0.0.1:${oidc_mock_port}" --postgres-dsn "$ARENA_SERVICE_POSTGRES_DSN" >"$artifact_dir/local-oidc-test-provider.log" 2>&1 &
    oidc_pid=$!
    trap 'kill "$oidc_pid" 2>/dev/null || true' EXIT INT TERM
    until curl -fsS "http://127.0.0.1:${oidc_mock_port}/.well-known/openid-configuration" >/dev/null; do sleep 1; done
    # Native loopback registrations omit the port so the provider accepts the runtime callback port.
    oidc_registration=$(curl -fsS -X POST "http://127.0.0.1:${oidc_mock_port}/register" -H 'Content-Type: application/json' --data "{\"application_type\":\"native\",\"client_name\":\"AI Arena local OIDC\",\"redirect_uris\":[\"http://127.0.0.1/auth/local-oidc/callback\"],\"grant_types\":[\"authorization_code\"],\"response_types\":[\"code\"],\"scope\":\"openid profile email\",\"token_endpoint_auth_method\":\"client_secret_post\"}")
    export ARENA_AUTH_LOCAL_OIDC_ISSUER="http://127.0.0.1:${oidc_mock_port}"
    export ARENA_AUTH_LOCAL_OIDC_ENABLED=1
    export ARENA_AUTH_LOCAL_OIDC_CLIENT_ID=$(printf '%s' "$oidc_registration" | sed -n 's/.*"client_id":"\([^"]*\)".*/\1/p')
    export ARENA_AUTH_LOCAL_OIDC_CLIENT_SECRET=$(printf '%s' "$oidc_registration" | sed -n 's/.*"client_secret":"\([^"]*\)".*/\1/p')
    test -n "$ARENA_AUTH_LOCAL_OIDC_CLIENT_ID" && test -n "$ARENA_AUTH_LOCAL_OIDC_CLIENT_SECRET"
    if ! make seaweed-up || ! make seaweed-bootstrap; then
      export ARENA_SERVICE_ARTIFACT_BACKEND=file
    fi
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
