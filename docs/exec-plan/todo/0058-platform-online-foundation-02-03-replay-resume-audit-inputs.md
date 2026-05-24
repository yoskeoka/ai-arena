# platform-online-foundation-02-03-replay-resume-audit-inputs
**Execution**: Use `/execute-task` to implement this plan.

## Objective

persisted match artifact を replay / resume / audit の source-of-truth input として再利用できる形に整理する。
最初のゴールは、service 側に保存された match metadata と `record.json` / `snapshot.json` / `history.json`
の関係を固定し、どの artifact と locator を使えば再実行・監査できるかを operator から辿れるようにすることに置く。

## Context

- `platform.md` と既存 runner/replay 実装は、`record.json` を primary source-of-truth としつつ `snapshot.json` / `history.json` を保持する前提をすでに持っている
- `0050` は terminal artifact を保存するが、service 側 persisted state から replay/resume/audit 入力をどう解決するかは未定義である
- `0054-external-gamemaster-manifest-registration-03-resume-replay-follow-up.md` でも replay/debug 入力 contract が必要になるため、platform 側の persisted-match input contract を先に整えておく価値がある
- `0046-platform-online-foundation-02-persistence-and-read-model.md` 親 scope のうち、source-of-truth artifact と rebuild path をこの child plan で受け持つ

## Scope

- replay / resume / audit が依存する persisted input contract を定義する
- service persisted state から `record` / `snapshot` / `history` / exported/public snapshot locator を解決する
- target-turn replay や snapshot resume に必要な metadata join を整理する
- persisted artifact の整合性を検証する verification path を追加する

この plan では以下を扱わない。

- external game master manifest overlay 自体の replay/debug 対応
- spectator 向け event stream / public API
- retry / rerun policy
- ranking 再計算

## Spec Changes

### `docs/specs/platform.md`

- service persisted state から replay/debug entrypoint へ渡す input contract を追記する
- `record.json` / `snapshot.json` / `history.json` / `exported-snapshot.json` の source-of-truth / derived 関係を service read path からも明記する

### `docs/specs/platform-service-skeleton.md`

- audit / replay / resume のために service が保持すべき locator と metadata を追記する

### 必要なら更新

- `docs/specs/platform-game-registry.md`
  - persisted match metadata から registry build/replay 入口へ渡す前提を補足する

## Expected Code Changes

- persisted match から replay/resume/audit input を解決する service helper
- artifact consistency / metadata join verification
- replay/resume/audit input resolution の integration test

## Sub-tasks

- [ ] replay / resume / audit が要求する persisted input を棚卸しする
- [ ] service persisted state と artifact source-of-truth の対応表を spec に落とす
- [ ] persisted match から replay/debug input を解決する helper を実装する
- [ ] `record` / `snapshot` / `history` の整合性 verification を追加する
- [ ] `0054` など後続 plan が再利用できる dependency surface を明文化する

## Parallelism

- input 棚卸しと spec 追記は並行できる
- `0056` 完了後は integration test と resolution helper を分担できる

## Dependencies

- depends on: `0056-platform-online-foundation-02-01-durable-store-and-write-model.md`
- depends on: parent/base item `0046-platform-online-foundation-02-persistence-and-read-model.md` (to be retired to `docs/exec-plan/done/` after split)
- informed by: `0054-external-gamemaster-manifest-registration-03-resume-replay-follow-up.md`

## Risks and Mitigations

- service read path が runner/game-specific replay logic まで持ち始めると責務が崩れる
  - mitigation: service 側は persisted input の解決と整合性確認に留め、game 固有 replay build は既存 registry/runner 側へ残す
- `record` / `snapshot` / `history` のどれが正本かが operator surface 上で曖昧になる
  - mitigation: primary source-of-truth は `record.json`、`snapshot.json` / `history.json` は replay/resume 補助 artifact という整理を明文化する
- external overlay follow-up と scope が混ざる
  - mitigation: この plan は platform 側 persisted input contract に限定し、manifest overlay 固有拡張は `0054` に残す

## Design Decisions

- replay / resume / audit で再利用するのは persisted match input の解決責務までとし、実行エンジンの責務は既存 runner 側へ残す
- service persisted state は artifact locator と metadata join を提供し、artifact 本文の正本性は file-backed layout に残す
- external overlay follow-up があっても、platform 側の persisted-match input contract を先に固定する
