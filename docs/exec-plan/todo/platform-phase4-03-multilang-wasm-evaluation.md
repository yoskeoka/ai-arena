# platform-phase4-03-multilang-wasm-evaluation
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Go 以外の言語でビルドした WASM module を `janken` 上で評価し、Phase 4 時点で platform がどこまでを「動作確認済み」と扱うかを明文化する。多言語 WASM の存在自体は project-plan の要件に沿うが、Phase 5 を止めないために、まず Go 基準経路を固定した後に評価 lane を分離して扱う。

親 plan:

- `platform-phase4-wasm-runtime.md` (`docs/exec-plan/todo/` または `docs/exec-plan/done/`)

depends on:

- `platform-phase4-01-wasm-runtime-contract.md`
- `platform-phase4-02-go-wasm-janken-verification.md`

## Scope

- 非 Go toolchain の WASM module を `janken` で流すための評価手順の整備
- candidate language ごとの build/runtime 差分、制約、失敗例の記録
- Phase 4 時点での「supported」「experiment-only」「unsupported」の区分整理
- 必要なら最小 1 言語分の reference sample を追加

この plan では以下は扱わない。

- 多数言語の全面サポート保証
- language-specific SDK の整備
- dungeon game での多言語 bot の本格運用
- online submission pipeline

## Spec Changes

### `docs/specs/ai-runtime.md`

- 多言語 WASM の扱いを support policy として追記する
- 「runtime contract は共通だが、toolchain ごとの動作確認状況は別で管理する」方針を明記する
- supported / experiment-only / unsupported の区分基準を定義する

### `docs/specs/janken-game.md`

- 多言語評価で利用する `janken` verification path を reference として追記する

### Optional new appendix or issue note

- 言語ごとの観測差分が長くなる場合は appendix または `docs/issues/` に切り出し、恒久 spec には support boundary だけ残す

## Expected Code Changes

### evaluation fixtures

- 少なくとも 1 つの非 Go WASM sample、またはそれに準ずる評価 artifact を追加する
- 言語固有の build helper が必要なら最小限追加する

### verification helpers

- `janken` で multi-language sample を流す helper または targeted test を追加する
- 成功ケースだけでなく、WASI 非互換や unsupported capability の失敗を再現できるようにする

### documentation outputs

- support matrix か評価結果の要約を repo 内に残す
- 環境前提が重い場合は、必須 gate ではなく optional verification として位置づける

## Verification

- 選定した非 Go toolchain で sample module を build できる、または build blocker を再現可能な形で記録できる
- `arena-runner` が非 Go WASM module を `janken` で実行できる、または contract 上の blocker が明確になる
- support 区分が spec と verification artifact で対応づく
- Go 基準経路の contract を崩さない

## Sub-tasks

- [ ] Decide the support classification policy for non-Go WASM toolchains
- [ ] Select the first candidate language(s) for evaluation
- [ ] Add the minimum evaluation sample or fixture
- [ ] Add a targeted `janken` verification path for non-Go WASM
- [ ] Record success cases, failure modes, and environment assumptions
- [ ] Reflect the support boundary back into `ai-runtime.md`

## Parallelism

- support policy drafting and candidate-language investigation can proceed in parallel
- sample build work and verification helper work can proceed in parallel after the first candidate is chosen

## Risks and Mitigations

- 多言語 support を早く約束しすぎると、toolchain 差分の運用コストを抱え込む
  - mitigation: Phase 4 では support policy を区分化し、confirmed path と experiment path を分ける
- CI や開発環境に重い toolchain 依存を入れると repo 運用が不安定になる
  - mitigation: 初回は optional verification 前提にし、常設 gate 化は明示的判断に分ける
- 失敗原因が runtime contract 由来か toolchain 由来か曖昧になる
  - mitigation: Go 基準経路との比較で差分を説明し、contract 側の不備と toolchain 特有の問題を切り分ける

## Design Decisions

- 多言語 WASM は Go 基準経路の後続評価 lane として扱う
- Phase 4 の成果は「全面サポート宣言」ではなく「support boundary の明文化」とする
- 長期に残すべきものは support policy と contract だけに絞り、詳細な試行ログは appendix または issue note に逃がしてよい
