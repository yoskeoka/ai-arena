#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <backend-url>" >&2
  exit 2
fi

backend_url=${1%/}
max_attempts=${REMOTE_HEALTH_MAX_ATTEMPTS:-42}
interval_seconds=${REMOTE_HEALTH_INTERVAL_SECONDS:-10}
request_timeout_seconds=${REMOTE_HEALTH_REQUEST_TIMEOUT_SECONDS:-15}
curl_bin=${CURL_BIN:-curl}
sleep_bin=${SLEEP_BIN:-sleep}
last_status=unavailable
last_api=unavailable
last_worker=unavailable

if [[ ! $max_attempts =~ ^[1-9][0-9]*$ ]]; then
  echo "REMOTE_HEALTH_MAX_ATTEMPTS must be a positive integer" >&2
  exit 2
fi
if [[ ! $interval_seconds =~ ^[1-9][0-9]*$ ]]; then
  echo "REMOTE_HEALTH_INTERVAL_SECONDS must be a positive integer" >&2
  exit 2
fi
if [[ ! $request_timeout_seconds =~ ^[1-9][0-9]*$ ]]; then
  echo "REMOTE_HEALTH_REQUEST_TIMEOUT_SECONDS must be a positive integer" >&2
  exit 2
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required to validate the health response" >&2
  exit 2
fi

response_file=$(mktemp)
trap 'rm -f "$response_file"' EXIT

write_observation() {
  if [[ -n ${GITHUB_OUTPUT:-} ]]; then
    {
      echo "last_http_status=$last_status"
      echo "last_api=$last_api"
      echo "last_worker=$last_worker"
    } >>"$GITHUB_OUTPUT"
  fi
  if [[ -n ${GITHUB_STEP_SUMMARY:-} ]]; then
    {
      echo "### Staging worker readiness"
      echo
      echo "- Last HTTP status: \`$last_status\`"
      echo "- Last API component: \`$last_api\`"
      echo "- Last worker component: \`$last_worker\`"
    } >>"$GITHUB_STEP_SUMMARY"
  fi
}

for ((attempt = 1; attempt <= max_attempts; attempt += 1)); do
  : >"$response_file"
  if ! status=$($curl_bin --silent --show-error --output "$response_file" --write-out '%{http_code}' \
    --max-time "$request_timeout_seconds" "$backend_url/healthz"); then
    status=transport-error
  fi
  last_status=$status

  components=$(jq -er 'if type == "object" and (keys == ["api", "worker"]) and (.api | type == "string") and (.worker | type == "string") then [.api, .worker] | @tsv else empty end' "$response_file" 2>/dev/null || true)
  if [[ -n $components ]]; then
    IFS=$'\t' read -r last_api last_worker <<<"$components"
  else
    last_api=empty-or-malformed
    last_worker=empty-or-malformed
  fi

  if [[ $status == 200 && $last_api == OK && $last_worker == OK ]]; then
    echo "staging backend worker readiness converged on attempt $attempt"
    write_observation
    exit 0
  fi
  echo "worker readiness attempt $attempt/$max_attempts: status=$last_status api=$last_api worker=$last_worker" >&2
  if ((attempt < max_attempts)); then
    $sleep_bin "$interval_seconds"
  fi
done

echo "staging backend worker readiness did not converge within $max_attempts attempts" >&2
write_observation
exit 1
