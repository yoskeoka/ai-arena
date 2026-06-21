#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
postgres_dsn=${ARENA_SERVICE_POSTGRES_DSN:-${AI_ARENA_PG_TEST_DSN:-postgres://arena:arena@127.0.0.1:55432/arena_service?sslmode=disable}}
frontend_origin=${LOCAL_AUTH_FRONTEND_ORIGIN:-http://localhost:5173}

cd "$repo_root"
make render-build >/dev/null

echo "creating local invite URL: operator signup invite"

create_invite() {
  if command -v direnv >/dev/null 2>&1; then
    direnv exec "$repo_root" ./app signup-invite-create --postgres-dsn "$postgres_dsn" --role operator
    return
  fi
  ./app signup-invite-create --postgres-dsn "$postgres_dsn" --role operator
}

invite_json=$(create_invite)
invite_token=$(printf '%s\n' "$invite_json" | sed -n 's/.*"invite_token":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)
role=$(printf '%s\n' "$invite_json" | sed -n 's/.*"role":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)
expires_at=$(printf '%s\n' "$invite_json" | sed -n 's/.*"expires_at":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)

if [ -z "$invite_token" ] || [ -z "$role" ] || [ -z "$expires_at" ]; then
  echo "failed to parse signup invite output" >&2
  exit 1
fi

invite_url="${frontend_origin}/login?invite_token=${invite_token}"

cat <<EOF
{
  "invite_token": "$invite_token",
  "role": "$role",
  "expires_at": "$expires_at",
  "invite_url": "$invite_url"
}
EOF
