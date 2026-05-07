# platform-phase3-02-5-registry-store-boundary
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`platform-phase3-02-game-registry.md` で導入する `GameDescriptor` ベースの registry を、将来 DB backend や self-service registration へ拡張しやすい構成へ整理する。lookup key と runtime build/replay 入口は維持したまま、永続化される registered game metadata と、process 内の build function 解決を分離し、in-memory 実装から DB-backed 実装へ滑らかに移行できる interface を固定する。

親 plan:

- `platform-phase3-common-interface-contract.md` (`docs/exec-plan/todo/` または `docs/exec-plan/done/`)

depends on:

- `platform-phase3-02-game-registry.md`

## Scope

- registry を store / resolver / composed runtime registry に分割する設計
- DB backend へ置き換え可能な persisted descriptor record の定義
- in-memory store adapter を残したまま interface を固定する移行
- runner / replay/debug から見える lookup contract を維持したまま内部構成を差し替える準備

この plan では以下は扱わない。

- 実 DB schema や migration
- 管理 API / UI / game registration self-service flow
- network 越しの game package 配布や plugin sandbox
- trusted external game backend そのものの transport 実装

## Spec Changes

### `docs/specs/platform-game-registry.md`

- registry contract を「runtime で直接 `GameDescriptor` を保持する層」から、「store された descriptor record を runtime resolver で解決する層」へ整理する
- persisted な registered game の最小 record を定義する
  - `game_id`
  - `game_version_major`
  - `build_mode`
  - `builder_id` または同等の resolver key
  - ruleset/build 制約を表す metadata
- canonical な一意キーは `game_id + game_version_major` の composite key とし、`registry_key` という論理名を残す場合も derived / denormalized field として扱うことを明記する
- runtime 側の `GameDescriptor` は DB に保存する shape ではなく、resolver が構築する process-local object であることを明記する
- registry lookup の責務を 3 段に分けて明記する
  - `RegistryStore`: key から persisted record を読む
  - `DescriptorResolver`: record を runtime `GameDescriptor` へ解決する
  - `Registry`: store + resolver を束ねて runner/replay に lookup を提供する

### `docs/specs/platform.md`

- runner / replay は registry から runtime `GameDescriptor` を取得するが、永続化 backend の種類は意識しないことを明記する
- compatibility 判定と build 実行の流れを、`lookup persisted record -> resolve runtime descriptor -> build -> catalog compatibility` の順に整理する

## Expected Code Changes

- `internal/platform/registry/`
  - `RegistryStore` interface
  - `DescriptorRecord` または同等の persisted shape
  - `DescriptorResolver` interface
  - store + resolver を束ねる `Registry`
- in-memory 実装
  - 既存の hard-coded registration を in-memory store adapter と in-process resolver registration に分割する
  - default registry は `InMemoryStore + StaticResolver` の compose に置き換える
- `cmd/arena-runner/main.go`
  - lookup 呼び出しは維持しつつ、registry 初期化が compose 前提になるよう整理する
- `internal/platform/replay/`
  - history replay も同じ resolved descriptor 経路を使い続けることを確認する
- `internal/games/echo/`, `internal/games/janken/`
  - game package 直登録ではなく、resolver が参照する builder ID / registration entry を公開する

## Interface Direction

この plan で固定したい最小境界は以下。

- `RegistryStore.Lookup(ctx, key) -> (DescriptorRecord, error)`
- `DescriptorResolver.Resolve(ctx, record) -> (GameDescriptor, error)`
- `Registry.Lookup(ctx, key) -> (GameDescriptor, error)`

`DescriptorRecord` は DB に保存できる plain data に限定し、function pointer や closure を含めない。
`GameDescriptor` は runtime でのみ存在する object とし、fresh run / snapshot resume / history replay の build 入口を持つ。
初期の resolver が pure transform に見える場合でも、将来 DB / remote metadata / cancellation / deadline を扱う拡張で public contract を変えないため、`ctx` と `error` を最初から含める。

## Verification

- `go test ./...` が通る
- in-memory store adapter 経由でも `echo-count` / `janken` lookup が成立する
- unknown key / unknown builder ID / incompatible build metadata が一貫した error になる
- runner と replay/debug が registry backend の実装詳細に依存していないことを test で確認する
- plain data の `DescriptorRecord` だけを見れば DB 保存対象が分かり、`GameDescriptor` だけを見れば runtime build/replay 責務が分かる状態になる

## Sub-tasks

- [ ] Define persisted `DescriptorRecord` vs runtime `GameDescriptor` roles in spec
- [ ] Define `RegistryStore`, `DescriptorResolver`, and composed `Registry` responsibilities
- [ ] Refactor the default in-memory registry into store + resolver components
- [ ] Preserve the existing lookup key (`game_id + game_version major`) through the refactor
- [ ] Keep fresh run / snapshot resume / history replay on the same resolved descriptor path
- [ ] Add tests for unknown registry key, unknown builder ID, and successful in-memory resolution
- [ ] Confirm runner and replay/debug do not depend on the concrete backend type

## Parallelism

- spec 更新と interface skeleton 作成は並行で進められる
- in-memory store adapter への分割と test 追加は、interface 固定後に並行で進められる

## Risks and Mitigations

- persisted shape と runtime shape を曖昧にしたまま分割すると、結局 DB へ function を保存できない問題が残る
  - mitigation: `DescriptorRecord` は plain data に限定し、runtime build function は resolver 側へ寄せる
- resolver key を game package の import path や Go symbol に寄せすぎると、将来の repo 分割や versioning で不安定になる
  - mitigation: `builder_id` は platform が安定運用できる明示的 identifier として定義する
- backend 抽象化を急ぎすぎて現行 in-memory registry の可読性を落とす
  - mitigation: first step では `InMemoryStore + StaticResolver` を default とし、runtime behavior は変えずに責務だけ分ける
- game plugin/self-service registration まで同時に広げると scope が膨らむ
  - mitigation: 今回は DB-backed lookup の差し替え可能性だけを扱い、配布・認証・承認フローは別 plan に送る

## Design Decisions

- registry lookup key は引き続き `game_id + game_version major` を使う
- `ruleset_version` は persisted record に制約情報として載せ得るが、lookup key には含めない
- DB に保存するのは plain data の `DescriptorRecord` までとし、runtime build/replay function は保存しない
- runtime build/replay function の解決は `DescriptorResolver` が担う
- default 実装は in-memory backend を残すが、構成は `InMemoryStore + StaticResolver + Registry` に寄せる
- runner / replay/debug から見える public contract は `Registry.Lookup -> GameDescriptor` のまま維持する
- 実 DB backend、registration API、plugin 配布はこの plan の後続に分離する
