# arena-runner-e2e-fixture-prebuild-and-init-deadline
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`arena-runner` の e2e で、fixture bot の起動コストに引きずられた `init failed for p1: invalid-timeout` を解消し、
local subprocess と WASM verification の両方で安定した black-box verification を維持する。

この plan は以下を成立させる。

- `init` request と `turn` request の deadline を別々に扱い、初回起動コストのばらつきで match 全体が不必要に `failed` へ落ちないようにする
- e2e fixture bot の `go run` 依存を減らし、必要な bot だけを lazy prebuild して test process 内で再利用する
- 既存の `go-ci` / `wasm-verification` の lane 分離を崩さず、workflow ごとに必要な fixture だけ build する
- `docs/specs/` と実装の両方で、「競技上重要なのは turn deadline であり、init は一度きりでより大きい起動コストを許容できる」境界を一致させる

Addresses:

- `docs/issues/done/arena-runner-e2e-init-regression.md`
- `docs/issues/done/arena-runner-e2e-timeout-flake.md`

## Context

- `docs/project-plan.md` は AI runtime を長期プロセスとして扱い、競技上の制約は主に turn 応答時間に置いている
- `docs/design-decisions/adr.md` の Phase 2 実行戦略は、WASM 統合前の local subprocess verification を開発用の比較経路として維持する方針を採っている
- `docs/specs/platform.md` では `init` response を protocol-ready ACK として扱っており、現行 contract では別の boot/readiness handshake は定義していない
- `docs/specs/ai-runtime.md` は request deadline を session/request 側の責務としているが、現行実装では `init` と `turn` が同じ短い deadline に寄っている
- `docs/specs/go-quality-gates.md` は default gate と dedicated WASM lane をすでに分離しているため、fixture prebuild も同じ lane boundary に従うべきである
- `docs/issues/done/arena-runner-e2e-init-regression.md` では `timeout` / `bad-json` / `mismatched-id` のような turn/failure-path test まで `p1` の init timeout で崩れており、AI ロジックではなく fixture 起動形態が不安定要因になっている

## Scope

- `arena-runner` / match/session lifecycle における init deadline の整理
- e2e fixture build helper と prepared entry path の導入
- local subprocess fixture の `go run` 依存を lazy prebuild に置き換えるための test harness 更新
- Go-WASM / Rust-WASM lane と整合する fixture build ownership の整理
- 関連 spec と verification command の更新

この plan では以下は扱わない。

- AI runtime protocol に新しい boot/readiness method を追加すること
- online match service や提出 API の timeout policy
- `supported` / `experiment-only` の language support policy 自体の変更
- fixture bot の game logic や `janken` / `dungeon` ルール変更

## Design Decision

この plan では追加 ADR は作らない。

- protocol contract は現状維持とし、`init` 自体を readiness ACK として使い続ける
- `init` deadline は `turn` deadline より大きく持てるようにし、起動コスト差を吸収する
- e2e では「全 fixture を一括 prebuild」ではなく、「各 test が必要な fixture 集合を宣言し、helper が lazy build + shared reuse する」方式を採る
- 各 test は AI player 名だけを宣言し、runtime kind や build toolchain には関知しない。`local-subprocess` / `wasm-wasi` の差分は fixture helper が吸収する
- WASM lane も同じ helper 設計に寄せるが、build 対象の選択は各 lane/test が宣言する fixture set に委ねる

boot handshake を protocol に追加する案は、runtime/session/spec 全体の contract 拡張が必要になり、今回の flake 解消より変更範囲が大きいので採らない。

## Spec Changes

### `docs/specs/platform.md`

以下を明記する。

- `init` request は protocol-ready ACK を確認する request であり、turn deadline と別の締切を持ってよい
- 競技上の主な応答制約は turn processing に置き、init は AI runtime の初回起動コストを吸収できる大きめの上限を許容してよい
- init timeout は turn timeout と同じ `invalid-timeout` reason を使ってよいが、event log 上は init phase failure であることが読めること

### `docs/specs/ai-runtime.md`

以下を明記する。

- request deadline は request kind ごとに異なってよく、`init` / `turn` / `game_over` を同一値に固定しない
- `local-subprocess` verification では source + reproducible build step を正本として扱い、test/CI helper が事前 build artifact を作って sidecar manifest と組み合わせてよい
- runtime contract としての readiness は追加 boot protocol ではなく `init` request/response で確認する
- e2e helper の caller は AI player 名だけを指定し、runtime kind ごとの build 手順や artifact path 解決は helper 側の責務として隠蔽してよい

### `docs/specs/go-quality-gates.md`

以下を明記する。

- e2e fixture build helper は lazy prebuild + shared reuse を前提にしてよい
- default `make test` は Go subprocess fixture だけを対象にし、WASM fixture build は必要な dedicated lane からだけ起動する
- Go-WASM / Rust-WASM lane はそれぞれ必要な fixture set だけを build する

## Expected Code Changes

### `internal/platform/match/` and adjacent runner/session wiring

- `initDeadline` を `turn` 系 deadline から明確に分離する
- `init` phase timeout が turn verification の expectation を不必要に壊さないよう、timeout semantics と event recording を整理する
- 必要なら config/constant 名を見直し、何の deadline かが code から読めるようにする

### `e2e/`

- fixture 宣言 helper を追加する
- 各 test は `assumeFixtures(...)` 相当の helper に runner へ渡したい AI player 名だけを明示する。helper は player 名ごとの prepared entry path を返し、caller は既存どおり `player_id=<entry>` 形式の `--player` arg を組み立てる
- helper は package-global cache と `sync.Once` を使い、同じ test process 内で build 結果を再利用する
- helper は manifest から runtime kind を解決し、Go subprocess fixture と Go-WASM / Rust-WASM fixture の build ownership を分け、未要求の artifact は build しない

### `testdata/ai/**`

- 既存 `.arena.json` を常に `go run` 前提のまま使うのではなく、test helper が temp sidecar manifest を生成して built artifact を指せるようにする
- source-of-truth は引き続き source code + checked-in manifest shape に置き、binary artifact は commit しない

### `Makefile` and workflows if command surface drift appears

- 既存 `make test`, `make test-wasm-go`, `make test-wasm-rust` の lane boundary を崩さず、必要なら helper 前提の補助 target/notes を更新する
- workflow ごとの fixture build scope が分かるよう、test selection と env guard の責務を整理する

## Sub-tasks

- [ ] Update `docs/specs/platform.md` to separate init deadline semantics from turn deadline semantics
- [ ] Update `docs/specs/ai-runtime.md` to describe request-kind-specific deadlines and test/CI prebuild usage
- [ ] Update `docs/specs/go-quality-gates.md` so lazy fixture prebuild responsibility matches the existing CI lane split
- [ ] Introduce e2e fixture descriptor/helper APIs for declared fixture dependencies
- [ ] Implement shared lazy prebuild caching for Go subprocess fixture bots inside the e2e package
- [ ] Switch affected `arena_runner_test.go` cases from checked-in `go run` entries to prepared fixture entry paths
- [ ] Extend the same helper model to Go-WASM and Rust-WASM lanes without forcing unrelated builds
- [ ] Adjust init deadline constants/wiring and verify that init timeout remains explicit while normal startup variance no longer causes match-wide false failures

## Parallelism

- [parallel] Spec wording updates can proceed independently from the concrete e2e helper implementation once the policy is fixed
- [parallel] Go subprocess fixture helper and WASM fixture helper can be implemented independently if they share only a thin descriptor interface
- init deadline wiring depends on the chosen spec wording for request-kind-specific deadlines
- final verification depends on both helper migration and init deadline separation landing together

## Risks and Mitigations

- init deadline を大きくしすぎるだけで fixture prebuild を入れないと、e2e runtime は改善しても default gate が無駄に遅いまま残る
  - mitigation: deadline separation と lazy prebuild を同一 plan で扱い、どちらか片方だけで終わらせない
- helper が hidden magic になると、どの test がどの fixture/language toolchain に依存するか読みにくくなる
  - mitigation: 各 test で必要 fixture を明示宣言させ、helper 側は build/caching だけを担う
- suite 全体一括 prebuild は CI lane 分離を壊し、Rust toolchain や不要 WASM build を通常 lane に持ち込む恐れがある
  - mitigation: lazy build + fixture declaration で lane ごとに必要な artifact だけ作る
- temp manifest/path の扱いを誤ると sidecar metadata compatibility assertion が弱くなる
  - mitigation: checked-in manifest schema を元に temp sidecar を生成し、game metadata compatibility check は既存と同じ code path を通す
- boot handshake を後追いで混ぜると spec と runtime contract が膨らむ
  - mitigation: 今回は protocol 拡張を避け、`init` を readiness ACK とする既存 contract のまま解く

## Verification

The execution PR is complete when the following are true.

- normal local startup variance では `TestArenaRunnerHappyPaths` と既存 failure-path tests が init timeout へ崩れにくくなる
- `init-timeout` case のような本来の init failure test は引き続き explicit に `failed` を検証できる
- e2e package 内で同一 fixture を複数 test が使っても、毎回 `go run` ではなく shared prebuilt artifact を再利用する
- default `make test` は Rust-WASM build を起動せず、Go-WASM build も必要な dedicated lane 以外では走らない
- `make test-wasm-go` と `make test-wasm-rust` はそれぞれ必要な fixture set だけを build して verification を完了できる
- `docs/specs/platform.md`, `docs/specs/ai-runtime.md`, `docs/specs/go-quality-gates.md` が最終実装境界と一致する
