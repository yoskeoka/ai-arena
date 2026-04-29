# platform-phase2-02-fixture-e2e
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`platform-phase2-01-foundation.md` の上に `echo-count` fixture と `arena-runner` の black-box verification を載せ、Phase 2 platform 単体の happy-path / failure-path を CLI と e2e で閉じる。

親 plan:

- `platform-phase2-implementation.md`。このファイルは `docs/exec-plan/todo/` または `docs/exec-plan/done/` に存在しうる

depends on:

- `platform-phase2-01-foundation.md`

## Scope

- `echo-count` fixture game
- fixture AI 群
- `arena-runner` からの match 起動
- simultaneous / sequential の happy-path e2e
- timeout / invalid action / protocol violation / init failure / shutdown failure の failure-path e2e

この plan では以下は扱わない。

- snapshot/history replay entrypoint
- `janken` 実装

## Spec Changes

### `docs/specs/platform.md`

- fixture appendix を追加する
- `echo-count` の rule、payload、score/placement、mode 別進行を書く
- `accepted` / `no_action` と failure reason の example record を追加する
- failure-mode AI を使った expected record 例を追加する
- `echo-count` の `game_id` / `game_version` と AI metadata 一致条件を明記する

### `docs/specs/janken-game.md`

- `janken` が fixture 完了後の richer integration であることを補足する
- `echo-count` と責務が重ならないことを明記する

## Expected Code Changes

- `internal/games/echo/`
- `testdata/ai/echo/`
- `cmd/arena-runner/` の CLI surface
- `e2e/`

追加する fixture AI は少なくとも以下を含む。

- `echo-ai`
- `timeout-ai`
- `invalid-action-ai`
- `bad-json-ai`
- `late-response-ai`
- `init-timeout-ai`
- `exit-after-init-ai`
- `hung-after-game-over-ai`

## Verification

完了は CLI 実行と black-box e2e で判定する。最低限、以下を機械的に確認できること。

- `arena-runner` から `echo-count` simultaneous match を 2 echo AI で起動できる
- `arena-runner` から `echo-count` sequential match を 2 echo AI で起動できる
- final score / placement / snapshot / stderr capture が期待どおり
- `game_id` 一致かつ `game_version major` 一致ケースだけ起動を許可する
- timeout / malformed / mismatched-id / illegal-action / late-response を別 reason で記録する
- init failure と shutdown failure を lifecycle event として残す

## Sub-tasks

- [ ] Update `docs/specs/platform.md` fixture appendix
- [ ] Update `docs/specs/janken-game.md` to clarify follow-up responsibility
- [ ] Implement `echo-count` fixture game for simultaneous and sequential modes
- [ ] Implement fixture AI programs
- [ ] Add happy-path CLI and e2e coverage
- [ ] Add failure-path CLI and e2e coverage
- [ ] Capture representative runner commands and expected record assertions

## Parallelism

- fixture game 実装と一部 fixture AI 実装は並行できる
- happy-path e2e と failure-path e2e は fixture AI が揃った後に分担できる

## Risks and Mitigations

- transcript 全文一致に寄ると e2e が壊れやすい
  - mitigation: turn order / counters / score / snapshot / failure reason の意味的 assertion を中心にする
- `echo-count` が肥大化すると `janken` の責務を食い始める
  - mitigation: fixture は deterministic platform verification に必要な最小要素へ留める
