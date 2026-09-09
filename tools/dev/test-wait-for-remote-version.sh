#!/usr/bin/env bash
set -euo pipefail

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT
expected_sha=0123456789abcdef0123456789abcdef01234567

cat >"$temp_dir/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
output_file=
while [[ $# -gt 0 ]]; do
  case "$1" in
    --output) output_file=$2; shift 2 ;;
    *) shift ;;
  esac
done
printf '{"version_sha":"%s"}\n' "${MOCK_VERSION_SHA}" >"$output_file"
printf '%s' "${MOCK_HTTP_STATUS:-200}"
EOF
chmod +x "$temp_dir/curl"

CURL_BIN="$temp_dir/curl" SLEEP_BIN=true REMOTE_VERSION_MAX_ATTEMPTS=1 MOCK_VERSION_SHA="$expected_sha" \
  "$repo_root/tools/dev/wait-for-remote-version.sh" https://backend.example "$expected_sha"

if CURL_BIN="$temp_dir/curl" SLEEP_BIN=true REMOTE_VERSION_MAX_ATTEMPTS=1 MOCK_VERSION_SHA=wrong \
  "$repo_root/tools/dev/wait-for-remote-version.sh" https://backend.example "$expected_sha"; then
  echo "version helper accepted a mismatched SHA" >&2
  exit 1
fi
