#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)

echo "local-dummy-fixture is a legacy alias for local invite creation; use tools/dev/local-invite-url.sh instead." >&2
exec "$repo_root/tools/dev/local-invite-url.sh"
