# platform-phase4-01-wasm-runtime-contract
**Execution**: Use `/execute-task` to implement this plan.

## Objective

WASM/WASI を AI player の正式 runtime kind として platform contract に組み込み、local subprocess と並ぶ実行経路として `arena-runner` から選択・起動・監査できる状態にする。Phase 5 のダンジョンゲーム実装が「AI runtime はまだ暫定」という前提に依存しないよう、まず runtime 契約と adapter 境界を固定する。

親 plan:

- `platform-phase4-wasm-runtime.md` (`docs/exec-plan/todo/` または `docs/exec-plan/done/`)

depends on:

- `platform-phase3-02-game-registry.md`
- `platform-phase3-03-game-master-runtime-boundary.md`

## Scope

- AI sidecar manifest における WASM runtime kind の追加
- WASM module の起動契約、標準 stream 契約、sandbox 制約、resource limit の spec 化
- wazero を使う WASM runtime adapter の導入
- `arena-runner` / catalog / registry から runtime kind を選ぶ入口の統一
- subprocess runtime と WASM runtime を並列に維持できる adapter 境界の固定

この plan では以下は扱わない。

- `janken` での Go 製 WASM sample AI の end-to-end 実証
- 多言語 WASM の互換性評価
- online 提出 API や永続化 backend への接続
- game master 側の WASM 化

## Spec Changes

### `docs/specs/platform.md`

- Phase 2a の「runtime は local subprocess を使う」という暫定記述を改め、AI runtime は `local-subprocess` と `wasm-wasi` の少なくとも 2 種を持つことを明記する
- `runtime adapter` の責務を、process/WASM の差分吸収、標準 stream 接続、stderr capture、timeout/shutdown 監査に整理する
- runner が sidecar manifest と CLI 指定から runtime kind を解決し、registry/catalog と矛盾しないことを明記する
- resource limit の責務を platform 側に寄せる
  - deadline は session/request 側
  - memory 上限や host capability 制限は runtime 側

### `docs/specs/platform-common-contract.md`

- transport 実装が subprocess か WASM かに依らず維持すべき不変条件を補足する
  - `stdout` は JSON-RPC response 専用
  - `stderr` は AI の自由ログであり platform が capture する
  - transport 継続不能は `runtime-stopped` として記録する
- WASM runtime 導入後も共通メソッド契約や failure 分類を変更しないことを明記する

### New `docs/specs/ai-runtime.md`

- AI player runtime の公式仕様を新設する
- 少なくとも以下を定義する
  - runtime manifest schema
  - `local-subprocess` / `wasm-wasi` の runtime kind
  - WASM module path と metadata の対応
  - 標準 input / output / error の扱い
  - 許可する WASI capability の最小集合
  - deadline、memory limit、shutdown の責務境界
  - sandbox 前提
    - network なし
    - repo/workspace への暗黙 access なし
  - 監査対象
    - runtime 起動失敗
    - malformed response
    - timeout
    - forced shutdown

## Expected Code Changes

### `internal/platform/runtime/`

- runtime 実装を subprocess 前提の単一 adapter から、共通 interface + `subprocess` / `wasm` 実装へ整理する
- wazero を用いた `wasm-wasi` adapter を追加する
- `stdin` / `stdout` / `stderr`、deadline、shutdown、error 分類の surface を既存 session が再利用できる形に揃える

### `internal/platform/session/`

- transport が process でも WASM でも同じ request/response loop を使えることを確認し、必要なら runtime error の shape を整理する

### `internal/platform/catalog/` and runner loading path

- AI manifest から runtime kind と entrypoint を解決する
- `arena-runner` が player ごとに適切な runtime adapter を選べるようにする
- registry/catalog の metadata compatibility 判定と runtime 解決の境界を整理する

### `cmd/arena-runner/`

- player manifest に `wasm-wasi` が指定された場合の起動経路を追加する
- runtime 起動失敗と manifest 不正を一貫した error として返す

## Verification

- `go test ./...`
- targeted test で以下を確認する
  - `wasm-wasi` manifest を解決して adapter を起動できる
  - `stdout` JSON-RPC / `stderr` log capture が subprocess と同じ contract で動く
  - deadline 超過が `invalid-timeout` または `runtime-stopped` に一貫して写像される
  - malformed response と起動失敗が spec 通り分類される
  - shutdown 時に module 実行が停止し、stderr summary が取得できる
- `arena-runner` が runtime kind ごとに正しい adapter を選ぶ unit/integration test を追加する

## Sub-tasks

- [ ] Define the official AI runtime contract in spec
- [ ] Define the WASM runtime manifest shape and runtime kind selection rules
- [ ] Refactor runtime adapter boundaries so subprocess and WASM share one session-facing interface
- [ ] Implement the wazero-based WASM/WASI adapter
- [ ] Update runner/catalog loading paths to resolve runtime kind from player metadata
- [ ] Add failure-classification coverage for startup, timeout, malformed output, and shutdown
- [ ] Verify subprocess and WASM paths preserve the same session contract

## Parallelism

- spec drafting and runtime package boundary audit can proceed in parallel
- manifest parsing and wazero adapter implementation can proceed in parallel after the runtime contract is fixed
- failure-classification tests can proceed in parallel with runner loading updates once adapter shape is stable

## Risks and Mitigations

- runtime adapter を WASM 導入のために作り直し過ぎると、既存 subprocess path を壊しやすい
  - mitigation: session-facing interface は維持し、実装差分を adapter 配下へ閉じ込める
- WASI capability を広く許しすぎると、project-plan の sandbox 前提が崩れる
  - mitigation: `ai-runtime.md` で deny-by-default の capability 方針を先に固定する
- registry/catalog と runtime 解決責務が混ざる
  - mitigation: metadata compatibility は registry/catalog、runtime 起動は player manifest/runner で分離する

## Design Decisions

- Phase 4 の first step は「Go 製 sample を動かすこと」ではなく「WASM runtime を正式 contract にすること」とする
- AI runtime の正本 spec を `platform.md` の一節ではなく `docs/specs/ai-runtime.md` に分離する
- runtime kind は sidecar manifest で明示する
- subprocess path は削除せず、Phase 4 の間は比較対象かつ fallback として維持する
