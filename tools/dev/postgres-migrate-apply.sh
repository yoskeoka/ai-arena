#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
dsn=${AI_ARENA_PG_MIGRATION_DSN:-}
atlas_image=${ATLAS_IMAGE:-arigaio/atlas:0.30.0}
migrations_url=${POSTGRES_MIGRATIONS_URL:-file://internal/platform/service/postgres/migrations}
baseline_version=${POSTGRES_MIGRATION_BASELINE_VERSION:-}

if [ -z "$dsn" ]; then
  echo "AI_ARENA_PG_MIGRATION_DSN is required" >&2
  exit 1
fi

run_psql() {
  query=$1
  attempt=0
  while [ "$attempt" -lt 10 ]; do
    if command -v psql >/dev/null 2>&1; then
      if result=$(psql "$dsn" -Atqc "$query" 2>/dev/null); then
        printf '%s\n' "$result"
        return
      fi
    elif result=$(docker run --rm --network host postgres:17-alpine \
      psql "$dsn" -Atqc "$query" 2>/dev/null); then
      printf '%s\n' "$result"
      return
    fi

    attempt=$((attempt + 1))
    sleep 1
  done

  echo "failed to query migration target via psql after $attempt attempts" >&2
  exit 1
}

run_atlas_apply() {
  docker run --rm --network host \
    -v "$repo_root:/work" \
    -w /work \
    -v /var/run/docker.sock:/var/run/docker.sock \
    "$atlas_image" \
    migrate apply "$@" --url "$dsn" --dir "$migrations_url"
}

revisions_exists=$(run_psql "SELECT CASE WHEN to_regclass('atlas_schema_revisions.atlas_schema_revisions') IS NULL THEN 0 ELSE 1 END;")
user_table_count=$(run_psql "SELECT count(*) FROM pg_catalog.pg_tables WHERE schemaname NOT IN ('pg_catalog', 'information_schema') AND NOT (schemaname = 'atlas_schema_revisions' AND tablename = 'atlas_schema_revisions');")

if [ "$revisions_exists" = "1" ]; then
  run_atlas_apply
  exit 0
fi

if [ "$user_table_count" = "0" ]; then
  run_atlas_apply --allow-dirty
  exit 0
fi

if [ -n "$baseline_version" ]; then
  run_atlas_apply --baseline "$baseline_version"
  exit 0
fi

echo "database has user tables but no Atlas revision history; set POSTGRES_MIGRATION_BASELINE_VERSION or run make postgres-migrate-baseline VERSION=<latest_version> first" >&2
exit 1
