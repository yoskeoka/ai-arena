#!/usr/bin/env bash
set -euo pipefail

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT

cat >"$temp_dir/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
output_file=
call_file=${MOCK_CALL_FILE:?}
while [[ $# -gt 0 ]]; do
  case "$1" in
    --output) output_file=$2; shift 2 ;;
    *) shift ;;
  esac
done
call_count=$(<"$call_file")
call_count=$((call_count + 1))
printf '%s' "$call_count" >"$call_file"
case "$call_count" in
  1)
    exit 7
    ;;
  2)
    printf '{"api":"OK","worker":"NOT_READY"}\n' >"$output_file"
    printf '503'
    ;;
  3)
    printf '{not-json}\n' >"$output_file"
    printf '200'
    ;;
  4)
    printf '{"api":"OK","worker":"OK"}\n' >"$output_file"
    printf '200'
    ;;
  *)
    printf '{"api":"OK","worker":"NOT_READY"}\n' >"$output_file"
    printf '503'
    ;;
esac
EOF
chmod +x "$temp_dir/curl"

call_file="$temp_dir/calls"
printf '0' >"$call_file"
CURL_BIN="$temp_dir/curl" SLEEP_BIN=true MOCK_CALL_FILE="$call_file" \
  REMOTE_HEALTH_MAX_ATTEMPTS=4 REMOTE_HEALTH_INTERVAL_SECONDS=1 \
  "$repo_root/tools/dev/wait-for-remote-health.sh" https://backend.example

if [[ $(<"$call_file") -ne 4 ]]; then
  echo "health helper did not retry transport, HTTP, and malformed responses" >&2
  exit 1
fi

printf '0' >"$call_file"
if CURL_BIN="$temp_dir/curl" SLEEP_BIN=true MOCK_CALL_FILE="$call_file" \
  REMOTE_HEALTH_MAX_ATTEMPTS=1 REMOTE_HEALTH_INTERVAL_SECONDS=1 \
  "$repo_root/tools/dev/wait-for-remote-health.sh" https://backend.example; then
  echo "health helper accepted a failed response" >&2
  exit 1
fi

summary_file="$temp_dir/summary"
output_file="$temp_dir/output"
printf '0' >"$call_file"
if GITHUB_OUTPUT="$output_file" GITHUB_STEP_SUMMARY="$summary_file" CURL_BIN="$temp_dir/curl" \
  SLEEP_BIN=true MOCK_CALL_FILE="$call_file" REMOTE_HEALTH_MAX_ATTEMPTS=1 \
  REMOTE_HEALTH_INTERVAL_SECONDS=1 "$repo_root/tools/dev/wait-for-remote-health.sh" https://backend.example; then
  echo "health helper accepted the timeout response" >&2
  exit 1
fi
if ! grep -q 'last_http_status=transport-error' "$output_file" || ! grep -q 'last_api=empty-or-malformed' "$output_file" || ! grep -q 'last_worker=empty-or-malformed' "$output_file"; then
  echo "health helper did not export the last observation" >&2
  exit 1
fi
if ! grep -q 'Last worker component: `empty-or-malformed`' "$summary_file"; then
  echo "health helper did not write the last worker observation to the summary" >&2
  exit 1
fi
