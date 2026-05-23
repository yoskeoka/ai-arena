# external-gamemaster-manifest-registration-03-resume-replay-follow-up
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`0052` で追加する dev-only external game master manifest overlay を、fresh run だけでなく
resume / replay / history build まで拡張するための follow-up を定義する。
この plan の到達点は、manifest 由来 game master を debug / replay flow に載せるために必要な
contract と verification を整理し、fresh-run first の初回実装から安全に分離することに置く。

## Context

- `0052` は fresh run only を明示スコープとする
- 既存 `arena-runner` には `--record-input` / `--snapshot-input` / `--history-input` / `--target-turn` があり、
  built-in descriptor 前提の replay/debug 導線を持っている
- manifest overlay でも replay/debug を同じ artifact contract で扱えると、consumer repo の自己完結検証が厚くなる
- ただし fresh-run と同時に入れると descriptor contract と debug artifact contract の両方が膨らむ

## Scope

- manifest overlay path に対する resume / replay / history build contract を定義する
- manifest に追加で必要な replay/debug metadata surface を整理する
- runner debug flags と manifest overlay の接続ルールを定義する
- external fixture を用いた replay/resume verification を整理する

この plan では以下を扱わない。

- `0052` の fresh-run overlay 基盤そのもの
- official runtime kind / admission policy の決定
- online replay service や persisted registration backend

## Spec Changes

### `docs/specs/platform-game-registry.md`

- temporary descriptor overlay でも snapshot resume / history replay build 入口を扱えるよう、
  runtime descriptor contract 上の要件を補足する

### `docs/specs/game-master.md`

- external game master manifest が replay/debug で何を満たす必要があるかを追記する
- 初回は optional だった replay/debug entry を、この follow-up でどこまで mandatory にするか整理する

### `docs/specs/platform.md`

- runner debug flags と manifest overlay path の組み合わせ rules を追記する
- `record.json` / `snapshot.json` / `history.json` を使う既存 debug entrypoint と矛盾しないことを明記する

## Expected Code Changes

- manifest overlay path で replay/debug descriptor を構築する runner integration
- replay/resume fixture assets
- history build / snapshot resume / metadata mismatch coverage

## Sub-tasks

- [ ] manifest overlay に必要な replay/debug contract を定義する
- [ ] debug flags と manifest overlay の precedence / validation を定義する
- [ ] snapshot resume path を追加する
- [ ] history replay build path を追加する
- [ ] external fixture による replay/resume verification を追加する

## Parallelism

- contract 定義と replay fixture 設計は並行で進められる
- snapshot resume と history replay は descriptor shape 固定後に並行で進められる

## Dependencies

- depends on: `0052-external-gamemaster-manifest-registration-01-dev-runner-overlay.md`
- informed by: `0053-external-gamemaster-manifest-registration-02-official-runtime-admission.md`

## Risks and Mitigations

- fresh-run overlay と replay/debug overlay の要件が混ざって初回実装が過大化する
  - mitigation: `0052` 完了前提の follow-up として明確に分離する
- manifest に replay/debug 専用情報を詰め込みすぎて fresh-run の最小 contract が見えなくなる
  - mitigation: fresh-run 必須項目と replay/debug 拡張項目を spec 上で分離する
- `record` / `snapshot` / `history` の source-of-truth 関係が曖昧になる
  - mitigation: 既存 artifact contract を維持し、debug entrypoint ごとの正本を明記する

## Design Decisions

- fresh run only を first milestone とし、replay/debug は別 follow-up に分離する
- manifest overlay の replay/debug 拡張でも既存 `arena-runner` artifact contract を崩さない
