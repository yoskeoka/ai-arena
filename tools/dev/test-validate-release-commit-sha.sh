#!/usr/bin/env bash
set -euo pipefail

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
valid_sha=$(git -C "$repo_root" rev-parse HEAD)

if [[ $(cd "$repo_root" && ./tools/dev/validate-release-commit-sha.sh "$valid_sha") != "$valid_sha" ]]; then
  echo "SHA helper did not accept the current full SHA" >&2
  exit 1
fi

for invalid_sha in "${valid_sha:0:7}" "${valid_sha^^}" "0123456789abcdef0123456789abcdef01234567"; do
  if (cd "$repo_root" && ./tools/dev/validate-release-commit-sha.sh "$invalid_sha"); then
    echo "SHA helper accepted an invalid or unreachable SHA: $invalid_sha" >&2
    exit 1
  fi
done
