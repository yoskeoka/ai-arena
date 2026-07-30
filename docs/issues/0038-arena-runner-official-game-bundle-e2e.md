# arena-runner-official-game-bundle-e2e

## Summary

`arena-bundle/v1` の game/AI ZIP を、外部ゲームリポジトリがローカルで
unpack し、WASM/WASI の game master と 2 つの AI を実行して完走を確認する
公開 E2E 経路がない。

現行の `arena-runner` は内蔵 registry の game を選択するか、dev-only の
native `local-subprocess` game-master manifest overlay を受け取る。したがって
official game bundle の WASI 起動を外部ゲームリポジトリの CI から直接検証
できない。

## Why This Matters

- game/AI artifact の upstream validator は archive、manifest、WASM import、
  digest を検証できるが、2-player match の実行互換性までは証明しない
- 外部ゲームの release CI は、GitHub Release に載せる exact bytes が game
  master と AI の両方として動作することを確認できない
- operator/service staging だけに依存すると、artifact author が失敗を早期に
  再現するための狭いローカル入口を持てない

## Desired Direction

後続 plan で、公開的かつ deterministic な artifact-backed runner E2E を
定義する。少なくとも次を満たすことを検討する。

- game bundle と AI bundle を validator 済みの exact ZIP bytes から
  materialize する
- game master と複数 AI を host filesystem / network capability なしの WASI
  runtime で実行する
- fresh run に加え、公開 snapshot / replay lifecycle を検証できる
- game repository が platform internal package を import せず、tag または
  commit で固定した公開 CLI/API だけを利用できる

この作業は release artifact の schema/admission contract を変更するものでは
なく、その contract を実行まで検証する consumer-facing test surface を追加する
ものである。

## Evidence

- `cmd/arena-runner` の `--game-master-manifest` は dev-only
  `local-subprocess` overlay として定義されている
- artifact admission/worker は digest-identified WASI bundle を扱えるため、
  platform service 側の実行基盤は別途存在する
- `reversi-ai-arena` の release artifact 実装では、validator と digest の
  検証は可能だが、official game bundle を runner へ渡す公開入力がない
