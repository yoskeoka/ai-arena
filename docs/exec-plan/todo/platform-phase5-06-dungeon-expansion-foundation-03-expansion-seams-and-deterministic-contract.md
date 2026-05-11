# platform-phase5-06-dungeon-expansion-foundation-03-expansion-seams-and-deterministic-contract
**Execution**: Use `/execute-task` to implement this plan.

Addresses:

- `docs/issues/dungeon-post-phase5-refactor-before-feature-expansion.md`

## Objective

Rogue / NetHack 系の拡張へ入る前提として、dungeon の expansion seam と deterministic contract を
subsystem 前提で固定する。`actors` `items` `inventory` `effects` `combat` `visibility/fov`
のような将来機能が、hidden information や replay contract を壊さず追加できる条件を spec と
test shape に落とし込む。

関連 issue:

- `docs/issues/dungeon-post-phase5-refactor-before-feature-expansion.md`

depends on:

- `docs/exec-plan/todo/platform-phase5-06-dungeon-expansion-foundation-01-domain-layout-and-state-split.md`
- `docs/exec-plan/todo/platform-phase5-06-dungeon-expansion-foundation-02-turn-engine-pipeline.md`

## Scope

- subsystem ごとの拡張差し込み位置を固定する
- deterministic rule を subsystem 単位で明文化する
- scenario catalog / deterministic regression / replay verification の役割分担を整理する
- feature expansion 開始前に必要な issue close 条件を定義する
- この plan の execution 完了時に `docs/issues/dungeon-post-phase5-refactor-before-feature-expansion.md` を `docs/issues/done/` へ移す

この plan では以下は扱わない。

- monsters / inventory / combat の本実装
- multi-floor dungeon の実装
- online service / persistence への拡張

## Spec Changes

### `docs/specs/dungeon-game.md`

- 将来 subsystem の責務を整理する
  - actors
  - items / inventory
  - tile or status effects
  - combat
  - visibility / FOV
- hidden information を保ったまま subsystem を増やす条件を補足する
- deterministic rule を subsystem 境界で明文化する
  - 単一 RNG stream
  - fixed phase order
  - stable iteration order
  - replay / resume source of truth

### `docs/specs/platform.md`

- dungeon の deterministic regression が何を比較し、何を比較しないかを subsystem 拡張前提で補足する

### `docs/specs/ai-runtime.md`

- reference AI が将来 subsystem を跨いでも `visible_state` 契約の範囲だけで判断する前提を補足する

### `docs/design-decisions/adr.md`

- dungeon feature expansion は subsystem seam を先に固定してから入る方針を記録する

## Expected Code Changes

### extension seams

- `games/dungeon` 配下で将来 subsystem が入る package/module 位置を整理する
- skeleton or narrow interface レベルで次の拡張点を用意する
  - actors
  - effects
  - inventory/items
  - combat hooks

### deterministic support

- deterministic helper / iteration helper / RNG ownership を subsystem 指向の形へ寄せる
- scenario catalog と deterministic result regression の test entrypoint を整理する

### issue closure support

- この issue を閉じるための完了条件を plan / spec / tests 上で明確化する
- execution PR では issue file を `docs/issues/done/` へ移す

## Verification

- `go test ./games/dungeon/... ./e2e/...`
- scenario catalog, deterministic result regression, replay/resume verification の 3 層が役割分担どおりに通る
- future subsystem skeleton が hidden information を bypass しない
- extension seam の追加で `internal` 依存や runner 直結の逆流が起きていない
- 実装完了時に `docs/issues/dungeon-post-phase5-refactor-before-feature-expansion.md` が `docs/issues/done/` へ移動している

## Sub-tasks

- [ ] 将来 subsystem の差し込み位置を整理する
- [ ] deterministic rule を subsystem 単位で spec 化する
- [ ] scenario catalog / regression / replay verification の役割を再整理する
- [ ] narrow interface or skeleton を追加する
- [ ] issue close 条件を定義する
- [ ] execution 完了時に issue file を `docs/issues/done/` へ移す

## Parallelism

- deterministic contract の spec 更新と AI/runtime 側の補足は並行で進められる
- subsystem skeleton と verification entrypoint 整理は extension seam が固まれば並行で進められる

## Risks and Mitigations

- 先取りしすぎて実装されない抽象化だけが残る
  - mitigation: skeleton は extension point と ownership の明示に留め、ゲームルール本体は後続 plan に送る
- deterministic rule が generic すぎて実装時の判断に使えない
  - mitigation: RNG ownership, iteration order, replay source-of-truth を具体的な dungeon subsystem に結び付けて書く
- reference AI の観測境界が拡張で曖昧になる
  - mitigation: hidden information と visible contract の境界を ai-runtime と dungeon spec の両方に残す

## Design Decisions

- Rogue / NetHack 系の拡張を見据え、subsystem seam を先に固定する
- deterministic contract は turn loop 単位ではなく subsystem 単位でも管理する
- scenario catalog と deterministic result regression は今後も両方残し、役割を明確に分ける
