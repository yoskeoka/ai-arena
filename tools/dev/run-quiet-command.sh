#!/bin/sh
set -eu

label=${1:?usage: run-quiet-command.sh <label> <command> [args...]}
shift

if [ "$#" -eq 0 ]; then
  echo "run-quiet-command.sh: missing command for $label" >&2
  exit 1
fi

if [ "${VERBOSE:-0}" = "1" ] || [ -n "${CI:-}" ]; then
  exec "$@"
fi

log_path=$(mktemp "/tmp/$(printf '%s' "$label" | tr ' /:' '---').XXXXXX")

if "$@" >"$log_path" 2>&1; then
  printf '%s passed\n' "$label"
  printf 'log: %s\n' "$log_path"
  exit 0
fi

printf '%s failed\n' "$label" >&2
printf 'log: %s\n' "$log_path" >&2
printf "hint: inspect targeted lines first, e.g. grep -niE 'error|fail' %s\n" "$log_path" >&2
exit 1
