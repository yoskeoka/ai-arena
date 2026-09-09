#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <backend-url> <expected-full-commit-sha>" >&2
  exit 2
fi

backend_url=${1%/}
expected_sha=$2
max_attempts=${REMOTE_VERSION_MAX_ATTEMPTS:-80}
interval_seconds=${REMOTE_VERSION_INTERVAL_SECONDS:-15}
request_timeout_seconds=${REMOTE_VERSION_REQUEST_TIMEOUT_SECONDS:-15}
curl_bin=${CURL_BIN:-curl}
sleep_bin=${SLEEP_BIN:-sleep}
last_status=unavailable
last_version=unavailable

if [[ ! $expected_sha =~ ^[0-9a-f]{40}$ ]]; then
  echo "expected full commit SHA must be 40 lowercase hexadecimal characters" >&2
  exit 2
fi
if [[ ! $max_attempts =~ ^[1-9][0-9]*$ ]]; then
  echo "REMOTE_VERSION_MAX_ATTEMPTS must be a positive integer" >&2
  exit 2
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required to validate the version response" >&2
  exit 2
fi

response_file=$(mktemp)
trap 'rm -f "$response_file"' EXIT

write_observation() {
  if [[ -n ${GITHUB_OUTPUT:-} ]]; then
    {
      echo "last_http_status=$last_status"
      echo "last_version_sha=$last_version"
    } >>"$GITHUB_OUTPUT"
  fi
  if [[ -n ${GITHUB_STEP_SUMMARY:-} ]]; then
    {
      echo "### Staging version convergence"
      echo
      echo "- Expected SHA: \`$expected_sha\`"
      echo "- Last HTTP status: \`$last_status\`"
      echo "- Last version SHA: \`$last_version\`"
    } >>"$GITHUB_STEP_SUMMARY"
  fi
}

for ((attempt = 1; attempt <= max_attempts; attempt += 1)); do
  : >"$response_file"
  if ! status=$($curl_bin --silent --show-error --output "$response_file" --write-out '%{http_code}' \
    --max-time "$request_timeout_seconds" "$backend_url/version"); then
    status=transport-error
  fi
  last_status=$status
  version=$(jq -er 'if type == "object" and (keys == ["version_sha"]) and (.version_sha | type == "string" and length > 0) then .version_sha else empty end' "$response_file" 2>/dev/null || true)
  if [[ -n $version ]]; then
    last_version=$version
  else
    last_version=empty-or-malformed
  fi

  if [[ $status == 200 && $version == "$expected_sha" ]]; then
    echo "staging backend version converged on attempt $attempt: $expected_sha"
    write_observation
    exit 0
  fi
  echo "version convergence attempt $attempt/$max_attempts: status=$last_status version=$last_version" >&2
  if ((attempt < max_attempts)); then
    $sleep_bin "$interval_seconds"
  fi
done

echo "staging backend did not converge to expected version within $max_attempts attempts" >&2
write_observation
exit 1
