# platform-phase2-implementation
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`docs/specs/platform.md` に対する Phase 2 実装を、完了条件が機械的に判定できる単位へ分割する。

この親 plan 自体はコード実装を直接持たず、以下を確定するためのものとする。

- 実装順序
- plan 間依存
- 各 plan の verification 境界
- `echo-count` fixture と `janken` richer integration の責務分離

## Planning Context

- `docs/project-plan.md` は platform をゲーム非依存基盤として位置付け、JSON-RPC 2.0、stdin/stdout、同時行動/順番制、stderr capture を要求している
- `docs/design-decisions/adr.md` では以下が既に決まっている
  - AI 通信は stdin/stdout + JSON-RPC 2.0
  - Phase 2 の最初の実装検証は local subprocess 実行を使う
  - Phase 2 の主実証ゲームは `janken`
  - `echo-count` は `janken` の代替ではなく、platform 単体検証用 fixture として扱う
- `docs/design-decisions/core-beliefs.md` に従い、spec を先に更新し、検証可能な単位で plan を切る

## Split Options

### Option A: protocol / runtime / game / replay を技術レイヤ単位で細分化する

利点:

- 実装責務は明快
- 並列化しやすい

欠点:

- 「この PR が終わったか」を CLI や e2e で判断しにくい
- session / match / fixture の境界で plan 間の結合が強くなりやすい

### Option B: verification 境界で分割する

利点:

- 各 plan の完了条件を `go test`, `CLI 実行`, `e2e` で閉じやすい
- review 時に「何が確認できれば終わりか」が明確
- 実装順序と受け入れ順序が揃う

欠点:

- plan ごとの技術スコープはやや広くなる
- foundation plan に複数 package が入る

## Recommendation

この親 plan では **Option B** を採用する。

理由:

- 今回のユーザー要件が「CLI 実行、unit test、e2e test などで完了を検証しやすい単位」にある
- platform core はレイヤ分割よりも「どの verification で閉じるか」で区切った方が execution と review の両方で扱いやすい
- `echo-count` fixture と replay/debug 機能は、CLI ベースの受け入れ条件を独立させた方が regression を見つけやすい

## Child Plans

### 1. `platform-phase2-01-foundation.md`

役割:

- spec 更新
- protocol / catalog / runtime / session / match / record の foundation 実装

完了条件:

- unit test 中心で閉じる
- `go test` により protocol / runtime / session / match / record の主要契約が検証される

### 2. `platform-phase2-02-fixture-e2e.md`

役割:

- `echo-count` fixture game
- fixture AI 群
- `arena-runner` happy-path / failure-path の CLI + e2e

depends on:

- `platform-phase2-01-foundation.md`

完了条件:

- CLI 実行と black-box e2e で閉じる
- `game_id` / `game_version major` validation を runner 外形で確認できる

### 3. `platform-phase2-03-replay-debug.md`

役割:

- `start-from-snapshot`
- `resume-from-history-and-continue`

depends on:

- `platform-phase2-02-fixture-e2e.md`

完了条件:

- snapshot / history file を入力した CLI 実行と e2e で閉じる

### 4. `platform-phase2-04-janken-integration.md`

役割:

- `janken` を platform fixture 後の richer integration として実装する

depends on:

- `platform-phase2-02-fixture-e2e.md`

完了条件:

- `janken` の CLI 実行と e2e で、`echo-count` では不足するゲーム固有責務を検証できる

## Shared Decisions

- `game_id` / `game_version` / `ruleset_version` を game metadata と AI metadata の共有契約とする
- protocol 互換性判定は `game_id` 一致かつ `game_version major` 一致で行う
- platform が game master に渡す turn 入力は `accepted` または `no_action` に正規化する
- `invalid-timeout`, `invalid-protocol-malformed`, `invalid-protocol-mismatched-id`, `invalid-illegal-action`, `invalid-protocol-late-response` は record 上で区別する
- `echo-count` は `docs/specs/platform.md` の fixture appendix として扱い、独立 spec は持たない
- `janken` は Phase 2 の主実証ゲームとして別 plan へ切り出す

## Sub-tasks

- [ ] Create `platform-phase2-01-foundation.md`
- [ ] Create `platform-phase2-02-fixture-e2e.md`
- [ ] Create `platform-phase2-03-replay-debug.md`
- [ ] Create `platform-phase2-04-janken-integration.md`
- [ ] Remove implementation detail from this parent plan and leave only split/ordering/verification guidance

## Parallelism

- `platform-phase2-04-janken-integration.md` は `platform-phase2-02-fixture-e2e.md` 完了後に別 execution stream として扱える
- `platform-phase2-03-replay-debug.md` は `foundation` 完了後すぐではなく、fixture/e2e の record 形式が固まった後に着手する

## Resolved Decisions

- 既存の巨大 plan をそのまま実行 plan として使わない
- Phase 2 実装は verification 境界ベースで 4 plan へ分割する
