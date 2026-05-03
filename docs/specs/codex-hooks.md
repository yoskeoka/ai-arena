# Codex Hook Integration

`ai-arena` は Codex の project-scoped hook を使って、編集直後の formatter と turn 終了時の quality gate を自動実行する。

## Hook Files

- `.codex/config.toml`
  - `features.codex_hooks = true`
- `.codex/hooks.json`
  - `PostToolUse`
  - `Stop`
- `tools/codex-hook-post-tool-use.sh`
- `tools/codex-hook-stop.sh`
- `tools/codex-hook.py`

## PostToolUse Contract

- `PostToolUse` は `apply_patch` 系 edit の後にだけ起動する
- patch payload に `.go` file が含まれる場合だけ `make fmt` を実行する
- formatter hook は auto fix を目的とし、`make fmt` の失敗時は Codex へ failure reason を返す
- formatter 実装は `docs/specs/go-quality-gates.md` の `make fmt` 契約に従う

## Stop Contract

- `Stop` は turn 終了時に起動する
- ただし以下の種類の変更が working tree にある場合だけ重い quality gate を走らせる
  - `.go` files
  - `Makefile`, `go.mod`, `go.sum`
  - `.codex/**`
  - `.github/workflows/**`
  - `tools/codex-hook*`
  - `docs/specs/go-quality-gates.md`
  - `docs/specs/codex-hooks.md`
- 実行する quality gate は以下
  - `make lint`
  - `make test`
- 最初の stop pass で gate が失敗した場合、hook は Codex に continuation reason を返して修正を促す
- `stop_hook_active = true` の continuation turn では再度 block しない

## Workspace Dispatch Compatibility

`ai-arena` の hook script は repo root に direct に置かず、`tools/` 配下の stable path に置く。

これにより以下のどちらでも同じ実装を再利用できる。

- `ai-arena` repo で Codex を直接開始した session
- `vibe-coding-workspace` root の dispatcher が `ai-arena` 用 hook として委譲する session
