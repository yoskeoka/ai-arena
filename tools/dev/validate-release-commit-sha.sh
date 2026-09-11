#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <full-commit-sha>" >&2
  exit 2
fi

candidate=$1
git_bin=${GIT_BIN:-git}

if [[ ! $candidate =~ ^[0-9a-f]{40}$ ]]; then
  echo "commit SHA must be 40 lowercase hexadecimal characters" >&2
  exit 2
fi

if ! resolved=$($git_bin rev-parse --verify "${candidate}^{commit}" 2>/dev/null); then
  echo "commit SHA is not reachable in this repository: $candidate" >&2
  exit 1
fi
if [[ $resolved != "$candidate" ]]; then
  echo "commit SHA did not resolve canonically: $candidate" >&2
  exit 1
fi

printf '%s\n' "$resolved"
