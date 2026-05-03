#!/usr/bin/env python3

import json
import re
import subprocess
import sys
from pathlib import Path


RELEVANT_STOP_PREFIXES = (
    ".codex/",
    ".github/workflows/",
    "tools/codex-hook",
)
RELEVANT_STOP_FILES = {
    "Makefile",
    "go.mod",
    "go.sum",
    "docs/specs/go-quality-gates.md",
    "docs/specs/codex-hooks.md",
}


def repo_root() -> Path:
    return Path(__file__).resolve().parent.parent


def run(cmd: list[str]) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        cmd,
        cwd=repo_root(),
        text=True,
        capture_output=True,
    )


def is_go_edit(payload: dict) -> bool:
    if payload.get("tool_name") != "apply_patch":
        return False
    command = str(payload.get("tool_input", {}).get("command", ""))
    return bool(re.search(r"\.go(\s|$)", command))


def changed_paths() -> list[str]:
    proc = run(["git", "status", "--porcelain"])
    if proc.returncode != 0:
        return []
    paths: list[str] = []
    for line in proc.stdout.splitlines():
        if not line:
            continue
        path = line[3:]
        if " -> " in path:
            path = path.split(" -> ", 1)[1]
        paths.append(path)
    return paths


def should_run_stop() -> bool:
    for path in changed_paths():
        if path.endswith(".go"):
            return True
        if path in RELEVANT_STOP_FILES:
            return True
        if path.startswith(RELEVANT_STOP_PREFIXES):
            return True
    return False


def emit_stop_block(reason: str, logs: list[str]) -> int:
    if logs:
        sys.stderr.write("\n\n".join(logs))
        if not logs[-1].endswith("\n"):
            sys.stderr.write("\n")
    json.dump({"decision": "block", "reason": reason}, sys.stdout)
    return 0


def post_tool_use(payload: dict) -> int:
    if not is_go_edit(payload):
        return 0
    proc = run(["make", "fmt"])
    if proc.returncode == 0:
        return 0
    sys.stderr.write("ai-arena PostToolUse hook: `make fmt` failed.\n")
    if proc.stdout:
        sys.stderr.write(proc.stdout)
    if proc.stderr:
        sys.stderr.write(proc.stderr)
    return 2


def stop(payload: dict) -> int:
    if payload.get("stop_hook_active"):
        return 0
    if not should_run_stop():
        return 0

    failures: list[str] = []
    for cmd in (["make", "lint"], ["make", "test"]):
        proc = run(cmd)
        if proc.returncode != 0:
            rendered = [f"$ {' '.join(cmd)}"]
            if proc.stdout:
                rendered.append(proc.stdout.rstrip())
            if proc.stderr:
                rendered.append(proc.stderr.rstrip())
            failures.append("\n".join(part for part in rendered if part))

    if not failures:
        return 0

    return emit_stop_block(
        "ai-arena stop hook found failing `make lint` or `make test`. Fix the quality gates before ending the turn.",
        failures,
    )


def main() -> int:
    mode = sys.argv[1]
    payload = json.load(sys.stdin)
    if mode == "post-tool-use":
        return post_tool_use(payload)
    if mode == "stop":
        return stop(payload)
    raise ValueError(f"unknown mode: {mode}")


if __name__ == "__main__":
    raise SystemExit(main())
