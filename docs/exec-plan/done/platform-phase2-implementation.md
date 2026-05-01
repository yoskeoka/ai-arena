# platform-phase2-implementation

## Objective

元の巨大な `platform-phase2-implementation` plan を、実行可能で review しやすい 4 つの child plan に分割し、元 plan に含まれていた内容が適切に移動しきっていることを確認する。

この親 plan は、分割前の大目標を保持するための記録であり、もう `/execute-task` の対象ではない。

## Completion Criteria

- 元 plan の実装内容が以下の child plan に過不足なく移されている
  - `platform-phase2-01-foundation.md`
  - `platform-phase2-02-fixture-e2e.md`
  - `platform-phase2-03-replay-debug.md`
  - `platform-phase2-04-janken-integration.md`
- child plan 間の依存と順序が明示されている
- 元 plan にだけ残って child plan へ移動されていない spec / code / verification 要件がない
- 分割完了後、この親 plan が実行対象ではなく記録対象であることが明示されている

## Planning Context

- `docs/project-plan.md` は platform をゲーム非依存基盤として位置付け、JSON-RPC 2.0、stdin/stdout、同時行動/順番制、stderr capture を要求している
- `docs/design-decisions/adr.md` では以下が既に決まっている
  - AI 通信は stdin/stdout + JSON-RPC 2.0
  - Phase 2 の最初の実装検証は local subprocess 実行を使う
  - Phase 2 の主実証ゲームは `janken`
  - `echo-count` は `janken` の代替ではなく、platform 単体検証用 fixture として扱う
- `docs/design-decisions/core-beliefs.md` に従い、spec を先に更新し、検証可能な単位で plan を切る

## Child Plans

### 1. `platform-phase2-01-foundation.md`

- protocol / catalog / runtime / session / match / record の foundation 実装
- unit test 中心で閉じる

### 2. `platform-phase2-02-fixture-e2e.md`

- `echo-count` fixture game
- fixture AI 群
- `arena-runner` happy-path / failure-path の CLI + e2e

depends on:

- `platform-phase2-01-foundation.md`

### 3. `platform-phase2-03-replay-debug.md`

- `start-from-snapshot`
- `resume-from-history-and-continue`

depends on:

- `platform-phase2-02-fixture-e2e.md`

### 4. `platform-phase2-04-janken-integration.md`

- `janken` を platform fixture 後の richer integration として実装する

depends on:

- `platform-phase2-02-fixture-e2e.md`

## Migration Checklist

- [x] foundation scope を `platform-phase2-01-foundation.md` へ移した
- [x] fixture / runner / e2e scope を `platform-phase2-02-fixture-e2e.md` へ移した
- [x] snapshot / history replay scope を `platform-phase2-03-replay-debug.md` へ移した
- [x] `janken` richer integration scope を `platform-phase2-04-janken-integration.md` へ移した
- [x] ordered child plan naming を導入して実行順を明示した
- [x] 親 plan 自体は実行対象ではなく、分割完了記録であることを明示した

## Migration Audit

分割前 parent plan から移した重要項目:

- `game_id` / `game_version` / `ruleset_version` 契約
- protocol failure と game validation failure の責務分離
- `echo-count` fixture appendix と failure-mode coverage
- `start-from-snapshot` / `resume-from-history-and-continue`
- `janken` を fixture 後の richer integration として扱う順序

この parent plan にだけ残すもの:

- 分割の目的
- child plan 間の順序
- 分割漏れがないことの確認結果

この parent plan から除去したもの:

- 実装タスクそのもの
- `/execute-task` 前提の execution 指示
- child plan に移した spec / code / verification 詳細

## Resolved Decisions

- 既存の巨大 plan をそのまま実行 plan として使わない
- Phase 2 実装は verification 境界ベースで 4 plan に分割する
- 分割完了後の parent plan は `todo/` に残さず `done/` へ移す
