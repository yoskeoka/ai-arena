# platform-phase5-06-dungeon-expansion-foundation-02-turn-engine-pipeline
**Execution**: Use `/execute-task` to implement this plan.

## Objective

dungeon の turn progression を `Match.Apply` 中心の一枚岩処理から、phase を持つ turn engine へ整理する。
movement / interaction / actor or tile effect / scoring / visibility refresh を独立に進化できる形へ切り替え、
次の feature expansion が `Apply` への条件分岐追加で崩れない基盤を作る。

関連 issue:

- `docs/issues/dungeon-post-phase5-refactor-before-feature-expansion.md`

depends on:

- `docs/exec-plan/todo/platform-phase5-06-dungeon-expansion-foundation-01-domain-layout-and-state-split.md`

## Scope

- turn engine の phase 境界を定義する
- `Apply` を orchestration へ寄せる
- movement / chest interaction / goal resolution / score recompute / discovery refresh を分離する
- 将来の actor interaction / item use / effect tick を差し込める phase slot を作る

この plan では以下は扱わない。

- monsters / combat / inventory の具体ルール実装
- FOV アルゴリズム変更
- reference AI の挙動改善

## Spec Changes

### `docs/specs/dungeon-game.md`

- 1 turn の解決順序を責務単位で明文化する
  - action normalization
  - movement resolution
  - interaction resolution
  - terminal / score update
  - visibility refresh
- この phase 順が deterministic contract の一部であることを明記する
- 将来の tile effect / actor effect がどの phase へ入るかの差し込み位置を補足する

### `docs/specs/game-master.md`

- dungeon game master が turn request を集約した後、phase 固定順で game progression を進める前提を補足する

### `docs/design-decisions/adr.md`

- dungeon turn progression を phase engine として扱う方針を記録する

## Expected Code Changes

### engine pipeline

- `games/dungeon` 配下で turn engine 相当の module を追加する
- `Apply` の処理を phase ごとの narrow function へ分ける
- phase 間で受け渡す intermediate state を定義する

### current mechanic migration

- 現在ある `move` / `wait` / chest split / goal bonus / discovery refresh を新 pipeline 上へ移す
- 現行 ruleset の挙動差分がないことを targeted scenario で確認する

### tests

- phase ごとの unit test を追加する
- 既存 scenario catalog を pipeline の safety net として流用する

## Verification

- `go test ./games/dungeon/... ./e2e/...`
- 既存 targeted scenario catalog が全て通る
- fixed deterministic result regression が維持される
- `Apply` 相当の public behavior が既存 ruleset で変わらない

## Sub-tasks

- [ ] 現行 `Apply` の処理を phase ごとに分解する
- [ ] turn engine の phase interface を定義する
- [ ] movement / interaction / scoring / visibility refresh を個別 function へ移す
- [ ] 将来 phase の空き slot を定義する
- [ ] targeted scenario と deterministic regression を追従させる

## Parallelism

- phase 順序の spec 化と ADR 更新は並行で進められる
- scenario catalog の期待値追従は phase interface が固まれば並行で進められる

## Risks and Mitigations

- phase を増やしただけでデータ所有者が曖昧になり、逆に複雑化する
  - mitigation: 01 で切った state ownership に従って各 phase の read/write 範囲を限定する
- pipeline 抽象化が早すぎて現行 mechanic でも読みにくくなる
  - mitigation: まずは現在ある mechanic をそのまま phase 化し、将来 hook は slot だけに留める
- chest / goal / score の順序差で deterministic result が崩れる
  - mitigation: existing scenario catalog と deterministic regression を phase 移行の acceptance gate に使う

## Design Decisions

- dungeon turn progression は phase engine として扱う
- `Apply` は public API と orchestration に寄せる
- current mechanic の parity を保ったまま engine 化する

