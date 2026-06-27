#!/bin/sh
set -eu

label=${1:?usage: run-quiet-command.sh <label> <command> [args...]}
shift

if [ "$#" -eq 0 ]; then
  echo "run-quiet-command.sh: missing command for $label" >&2
  exit 1
fi

if [ "${VERBOSE:-0}" = "1" ]; then
  exec "$@"
fi

log_path=$(mktemp "/tmp/$(printf '%s' "$label" | tr ' /:' '---').XXXXXX.log")

if "$@" >"$log_path" 2>&1; then
  rm -f "$log_path"
  printf '%s passed\n' "$label"
  exit 0
fi

cat "$log_path" >&2
rm -f "$log_path"
printf '%s failed\n' "$label" >&2
exit 1
