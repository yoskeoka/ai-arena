# runtime-stream-capture-flake

## Summary

`make test` 実行中に `internal/platform/runtime` の
`TestStartStreamsAndCapturesStderr` が一度だけ
`incoming err: read |0: file already closed` で失敗した。
同 test を単体で再実行すると通過し、直後の `make test` 再実行でも再現しなかったため、
現時点では flaky failure とみなす。

## Context

- observed on: `2026-05-24`
- command: `make test`
- failing package: `./internal/platform/runtime`
- failing test: `TestStartStreamsAndCapturesStderr`

## Impact

- 現行 quality gate の全件通過を不安定化させる可能性がある
- local-subprocess adapter の stdout close と incoming channel 終端の競合がある場合、
  今後の runner / session 周辺変更のたびにノイズになる

## Next Steps

- `internal/platform/runtime/local_subprocess.go` の stdout reader 終端順序と
  `adapter.incoming` close 条件を見直す
- `TestStartStreamsAndCapturesStderr` が boot response を待つ前に EOF race を踏まないよう、
  helper process と reader 側の同期条件を確認する
- 再現条件が掴めるなら stress loop を追加し、fix 前に failure mode を固定する
