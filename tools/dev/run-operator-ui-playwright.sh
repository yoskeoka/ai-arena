#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
operator_ui_dir="$repo_root/operator-ui"
lane=${OPERATOR_UI_LANE:-}
store_dir=${PNPM_STORE_DIR:-/tmp/pnpm-store-ai-arena}
xdg_data_home=${XDG_DATA_HOME:-/tmp/.xdg-data-ai-arena}
pnpm_home=${PNPM_HOME:-/tmp/.pnpm-home-ai-arena}

set_default() {
  name=$1
  value=$2
  eval "current=\${$name:-}"
  if [ -z "$current" ]; then
    export "$name=$value"
  fi
}

resolve_operator_ui_path() {
  path=$1
  case "$path" in
    /*) printf '%s\n' "$path" ;;
    ./*) printf '%s/%s\n' "$operator_ui_dir" "${path#./}" ;;
    *) printf '%s/%s\n' "$operator_ui_dir" "$path" ;;
  esac
}

if [ -n "$lane" ]; then
  case "$lane" in
    fixture-local)
      set_default OPERATOR_UI_TEST_SCENARIO local
      set_default OPERATOR_UI_PLAYWRIGHT_REPORTER dot
      ;;
    real-local)
      set_default OPERATOR_UI_TEST_SCENARIO real-local
      set_default OPERATOR_UI_BACKEND_MODE real-local
      set_default OPERATOR_UI_EXPECT_DELEGATED_DOWNLOAD auto
      set_default OPERATOR_UI_CAPTURE_ARTIFACTS 1
      set_default OPERATOR_UI_RESET_POSTGRES 1
      set_default OPERATOR_UI_ARTIFACT_DIR ./test-results/real-local
      set_default OPERATOR_UI_REPORT_DIR ./playwright-report/real-local
      set_default OPERATOR_UI_PLAYWRIGHT_REPORTER dot
      ;;
    auth-local)
      set_default OPERATOR_UI_TEST_SCENARIO real-local
      set_default OPERATOR_UI_BACKEND_MODE auth-mock
      set_default OPERATOR_UI_TEST_AUTH 1
      set_default OPERATOR_UI_FRONTEND_HOST localhost
      set_default OPERATOR_UI_RESET_POSTGRES 1
      set_default OPERATOR_UI_EXPECT_DELEGATED_DOWNLOAD 0
      set_default OPERATOR_UI_CAPTURE_ARTIFACTS 1
      set_default OPERATOR_UI_ARTIFACT_DIR ./test-results/auth-local
      set_default OPERATOR_UI_REPORT_DIR ./playwright-report/auth-local
      set_default OPERATOR_UI_PLAYWRIGHT_REPORTER dot
      ;;
    ci-auth)
      set_default OPERATOR_UI_TEST_SCENARIO ci
      set_default OPERATOR_UI_BACKEND_MODE auth-mock
      set_default OPERATOR_UI_TEST_AUTH 1
      set_default OPERATOR_UI_FRONTEND_HOST localhost
      set_default OPERATOR_UI_EXPECT_DELEGATED_DOWNLOAD 0
      set_default OPERATOR_UI_PLAYWRIGHT_REPORTER dot
      ;;
    ci-file-backed)
      set_default OPERATOR_UI_TEST_SCENARIO ci
      set_default OPERATOR_UI_BACKEND_MODE file-backed
      set_default OPERATOR_UI_EXPECT_DELEGATED_DOWNLOAD 0
      set_default OPERATOR_UI_PLAYWRIGHT_REPORTER dot
      ;;
    ci-postgres)
      set_default OPERATOR_UI_TEST_SCENARIO ci
      set_default OPERATOR_UI_BACKEND_MODE postgres
      set_default OPERATOR_UI_EXPECT_DELEGATED_DOWNLOAD 1
      set_default OPERATOR_UI_PLAYWRIGHT_REPORTER dot
      ;;
    remote)
      set_default OPERATOR_UI_TEST_SCENARIO remote
      set_default OPERATOR_UI_EXPECT_DELEGATED_DOWNLOAD auto
      set_default OPERATOR_UI_CAPTURE_ARTIFACTS 1
      set_default OPERATOR_UI_ARTIFACT_DIR ./test-results/remote
      set_default OPERATOR_UI_REPORT_DIR ./playwright-report/remote
      set_default OPERATOR_UI_PLAYWRIGHT_REPORTER dot
      ;;
    *)
      echo "unsupported OPERATOR_UI_LANE: $lane" >&2
      exit 1
      ;;
  esac
fi

if [ "${VERBOSE:-0}" = "1" ] && [ "${OPERATOR_UI_PLAYWRIGHT_REPORTER:-dot}" = "dot" ]; then
  export OPERATOR_UI_PLAYWRIGHT_REPORTER=list
fi

artifact_dir=$(resolve_operator_ui_path "${OPERATOR_UI_ARTIFACT_DIR:-./test-results}")
report_dir=$(resolve_operator_ui_path "${OPERATOR_UI_REPORT_DIR:-./playwright-report}")
mkdir -p "$store_dir" "$xdg_data_home" "$pnpm_home"
mkdir -p "$artifact_dir" "$report_dir"
playwright_log="$artifact_dir/playwright.exec.log"
export XDG_DATA_HOME="$xdg_data_home"
export PNPM_HOME="$pnpm_home"

"$repo_root/tools/dev/ensure-pnpm-install.sh" "$operator_ui_dir"
"$repo_root/tools/dev/ensure-operator-ui-playwright-browser.sh"

cd "$operator_ui_dir"
export pnpm_config_store_dir="$store_dir"

if [ "${VERBOSE:-0}" = "1" ]; then
  exec pnpm exec playwright test "$@"
fi

: >"$playwright_log"
if pnpm exec playwright test "$@" >"$playwright_log" 2>&1; then
  printf 'operator-ui %s passed\n' "${lane:-playwright}"
  printf 'log: %s\n' "$playwright_log"
  printf 'artifacts: %s\n' "$artifact_dir"
  printf 'report: %s\n' "$report_dir"
  exit 0
fi

printf 'operator-ui %s failed\n' "${lane:-playwright}" >&2
printf 'log: %s\n' "$playwright_log" >&2
printf "hint: inspect targeted lines first, e.g. grep -niE 'error|fail' %s\n" "$playwright_log" >&2
printf 'artifacts: %s\n' "$artifact_dir" >&2
printf 'report: %s\n' "$report_dir" >&2
exit 1
