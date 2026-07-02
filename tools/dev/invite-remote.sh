#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
postgres_dsn=${1:-${ARENA_SERVICE_POSTGRES_DSN:-${INVITE_REMOTE_POSTGRES_DSN:-}}}
frontend_origin=${2:-${INVITE_REMOTE_FRONTEND_ORIGIN:-${LOCAL_AUTH_FRONTEND_ORIGIN:-}}}

if [ -z "$postgres_dsn" ]; then
  echo "remote Postgres DSN is required" >&2
  echo "usage: $0 <postgres-dsn> <frontend-origin>" >&2
  exit 1
fi
if [ -z "$frontend_origin" ]; then
  echo "frontend origin is required" >&2
  echo "usage: $0 <postgres-dsn> <frontend-origin>" >&2
  exit 1
fi

frontend_origin=${frontend_origin%/}

cd "$repo_root"
make render-build >/dev/null

invite_json=$(./app signup-invite-create --postgres-dsn "$postgres_dsn" --role operator)
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
