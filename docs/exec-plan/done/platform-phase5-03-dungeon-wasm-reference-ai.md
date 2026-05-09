# platform-phase5-03-dungeon-wasm-reference-ai
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Go subprocess bot で先に検証した判断ロジックを流用し、WASM/WASI で動く dungeon reference AI を成立させる。固定マップや seed 付きマップで継続的にゲームクリアまたはスコア獲得ができる公式 baseline bot を用意し、Phase 5 の成果を正式 runtime 経路へ接続する。

depends on:

- `platform-phase5-01-dungeon-fixed-map-mvp.md`
- `platform-phase5-02-dungeon-seeded-generation.md`
- `platform-phase5-02-5-dungeon-balance-tuning.md`
- `platform-phase4-02-go-wasm-janken-verification.md`

## Scope

- subprocess bot と WASM bot が共有する判断ロジック package の固定
- Go から WASM/WASI へ build する dungeon reference AI の追加
- `arena-runner` で dungeon game と WASM bot を組み合わせた verification path の追加
- 「高度な最適戦略ではないが、継続的にクリアまたは加点できる」baseline の成立
- balance 調整後 ruleset で treasure routing と goal routing の両方を扱える baseline の成立

この plan では以下は扱わない。

- 多言語 dungeon bot support
- 高度な探索最適化や対人メタ戦略
- モンスターや戦闘前提の combat bot

## Spec Changes

### `docs/specs/ai-runtime.md`

- dungeon reference AI の Go-to-WASM build / manifest 例を追加する
- Phase 5 での official reference bot の位置づけを補足する

### `docs/specs/dungeon-game.md`

- reference AI が前提にしてよい観測情報と、禁止される hidden information を明記する
- local subprocess bot と WASM bot が同じ判断ロジックを共有する前提を、開発フローとして補足する
- 視界半径が変わっても同じ判断層が扱える input shape を保つ
- balance 調整後の turn / score 条件でも baseline bot が成立する前提を補足する

## Expected Code Changes

### shared bot logic

- `games/dungeon/botlogic` または同等 package を追加する
- subprocess bot と WASM bot の entrypoint から同じ判断ロジックを呼ぶ

### WASM bot entrypoint

- `cmd/dungeon-bot-wasm/` を追加する
- Go-to-WASM build target と sidecar manifest を追加する

### verification helpers

- `make run-dungeon-go-wasm` のような helper を追加する
- local subprocess bot と WASM bot で同じ seed / 同じ可視入力に対して整合する行動列を確認できるようにする

## Verification

- `go test ./...`
- Go 版 dungeon WASM bot を build できる
- `arena-runner` で dungeon game + WASM bot の match を完走できる
- 同じ入力列に対して subprocess bot と WASM bot が同じ判断を返す
- baseline bot が固定マップまたは seed 付きマップで継続的にクリアまたは加点できる

## Sub-tasks

- [ ] shared bot logic package を定義する
- [ ] local subprocess bot から shared logic を参照するよう整理する
- [ ] WASM bot entrypoint と manifest を追加する
- [ ] Go-to-WASM build helper を追加する
- [ ] dungeon verification path を `arena-runner` に追加する
- [ ] [parallel] manual helper を追加する
- [ ] [parallel] shared logic の parity test を追加する

## Parallelism

- WASM entrypoint 追加と parity test 追加は、shared bot logic の API が固まれば並行で進められる
- build helper 追加と manual helper 追加は並行で進められる

## Risks and Mitigations

- subprocess bot と WASM bot が別実装に分岐すると、比較基準として使えない
  - mitigation: 判断ロジックは shared package に一本化し、entrypoint だけ分ける
- baseline bot の要求を上げすぎると、Phase 5 が bot 最適化で止まる
  - mitigation: 「継続的にクリアまたは加点できる」水準をゴールに置き、高度な最適化は後続に送る
- WASM 化を後付けにすると、Go subprocess 前提の API が shared logic に漏れやすい
  - mitigation: shared logic は純粋な domain input/output だけを受け取り、IO は entrypoint 側に閉じ込める

## Design Decisions

- Phase 5 の official reference bot は Go 実装を source of truth とする
- 開発中の高速 feedback は subprocess bot、正式 verification は WASM bot で行う
- shared bot logic は transport 非依存・runtime 非依存の pure decision layer とする
- baseline bot の成功条件は「毎回最適」ではなく「継続的に完走または加点できる」とする
- baseline bot は最短ゴール専用 bot ではなく、treasure-heavy な勝ち筋にも最低限反応できることを目標にする
