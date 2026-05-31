#!/bin/bash
set -euo pipefail

base_ref="${1:-origin/main}"

contains_japanese_text() {
    local path="$1"
    python3 - "$path" <<'PY'
import sys
from pathlib import Path

path = Path(sys.argv[1])
content = path.read_text(encoding="utf-8", errors="ignore")
for char in content:
    codepoint = ord(char)
    if (
        0x3040 <= codepoint <= 0x30FF
        or 0x31F0 <= codepoint <= 0x31FF
        or 0x3400 <= codepoint <= 0x4DBF
        or 0x4E00 <= codepoint <= 0x9FFF
        or 0xF900 <= codepoint <= 0xFAFF
        or 0xFF65 <= codepoint <= 0xFF9F
    ):
        raise SystemExit(0)
raise SystemExit(1)
PY
}

while IFS= read -r path; do
    [ -n "$path" ] || continue
    [ -f "$path" ] || continue

    case "$path" in
        docs/*.md)
            contains_japanese_text "$path" && printf '%s\n' "$path"
            ;;
    esac
done < <(git diff --name-only --diff-filter=AMR "${base_ref}...HEAD")
