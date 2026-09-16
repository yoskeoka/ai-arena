#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

generated_paths=(
  typespec/generated/openapi/operator
  operator-ui/src/generated/operator-api
)

pnpm --dir typespec install --frozen-lockfile
pnpm --dir typespec run test:normalize-empty-client-context
pnpm --dir typespec run build

if ! git diff --exit-code -- "${generated_paths[@]}"; then
  echo "TypeSpec build left tracked generated-artifact drift." >&2
  exit 1
fi

untracked=$(git ls-files --others --exclude-standard -- "${generated_paths[@]}")
if [[ -n "$untracked" ]]; then
  echo "TypeSpec build left untracked generated artifacts:" >&2
  printf '%s\n' "$untracked" >&2
  exit 1
fi

pnpm --dir operator-ui install --frozen-lockfile
pnpm --dir operator-ui run build
