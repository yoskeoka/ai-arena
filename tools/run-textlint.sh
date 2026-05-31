#!/bin/bash
set -euo pipefail

if [ "${1:-}" = "--all" ]; then
    shift
    mapfile -t files < <(git ls-files 'docs/**/*.md' 'docs/*.md')
else
    if [ "$#" -eq 0 ]; then
        echo "usage: $0 --all | <target.md>..." >&2
        exit 1
    fi
    files=("$@")
fi

if [ "${#files[@]}" -eq 0 ]; then
    echo "No Markdown files found."
    exit 0
fi

pnpm exec textlint --rulesdir ./tools/textlint-rules --format json "${files[@]}"
