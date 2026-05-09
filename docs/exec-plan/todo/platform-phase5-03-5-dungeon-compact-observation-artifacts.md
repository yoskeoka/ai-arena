# platform-phase5-03-5-dungeon-compact-observation-artifacts
**Execution**: Use `/execute-task` to implement this plan.

## Objective

dungeon の local verification と AI Agent 実装時の観察導線を compact artifact
中心へ切り替える。human と agent の両方が、長い `event_log` や full JSON を
既定で開かなくても match 結果を確認できる状態を作る。

depends on:

- `docs/exec-plan/done/platform-phase5-03-dungeon-wasm-reference-ai.md`
- `docs/issues/dungeon-agent-observation-should-default-to-compact-artifacts.md`

## Scope

- `arena-runner` の derived artifact として `result-summary.json` を追加する
- dungeon manual helper に quiet/summary-first 導線を追加する
- `ai-arena/AGENTS.md` に artifact 読取順と log trimming rule を追加する
- compact artifact を前提にした e2e / unit test を追加する

この plan では以下は扱わない。

- source-of-truth としての `record.json` / `history.json` の削除や縮小
- 全 game 共通の artifact redesign
- replay/debug 用詳細 artifact の互換性変更

## Spec Changes

### `docs/specs/platform.md`

- `result-summary.json` を derived artifact として追加する
- `structured-log.ndjson` / `stdout` / summary artifact の使い分けを明記する
- `event_log` は source-of-truth/debug 向けであり、通常の確認導線の既定ではないことを補足する

### `docs/specs/dungeon-game.md`

- local verification でまず見る artifact と、per-turn log を読む条件を追記する
- compact result に含める dungeon 固有 field の責務を定義する

### `AGENTS.md`

- dungeon/debug artifact の既定読取順を追加する
- `record.json.event_log` と `history.json` は因果追跡時だけ読むことを明記する

## Expected Code Changes

### runner artifact generation

- `cmd/arena-runner` で `result-summary.json` を standard artifact として出力する
- summary は placement, score breakdown, selected public-state fields, artifact path 参照を持つ

### helper / CLI ergonomics

- `--log-output none`、または同等の quiet mode を追加する
- `make run-dungeon-local-quiet` など compact 導線の helper target を追加する

### verification

- compact artifact の shape を検証する test を追加する
- quiet mode でも standard artifact が残ることを確認する

## Verification

- `go test ./cmd/arena-runner ./e2e/...`
- `make run-dungeon-local-quiet` で full event stream を読まずに結果確認できる
- `result-summary.json` と `exported-snapshot.json` だけで順位と主要結果を確認できる

## Sub-tasks

- [ ] summary artifact の schema を specs に追加する
- [ ] runner に `result-summary.json` 出力を実装する
- [ ] quiet/summary-first helper を追加する
- [ ] `AGENTS.md` に observation rule を追加する
- [ ] compact artifact を前提にした test を追加する

## Risks and Mitigations

- summary に情報を入れすぎると full snapshot の縮小コピーになる
  - mitigation: selected public fields に絞り、詳細は既存 artifact へ退避する
- quiet mode が debug 能力を落とす
  - mitigation: full log path は残し、既定導線だけ compact 側へ寄せる
- dungeon 固有設計を早く一般化しすぎる
  - mitigation: まず dungeon で固め、他 game への横展開は別 plan で判断する

## Design Decisions

- compact observation は source-of-truth 置換ではなく derived artifact 追加として実現する
- agent/human の既定導線は `result-summary.json` -> `exported-snapshot.json` / `snapshot.json` -> full logs とする
- quiet helper は debug を捨てず、既定の読み始めを軽くするために導入する
