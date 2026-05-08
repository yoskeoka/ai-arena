# wasm-e2e-coverage-and-ci-cost
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`ai-arena` の WASM verification を、default Go quality gate と dedicated WASM lane に整理し直す。`make test` / `make lint` は通常 PR で軽く保ちつつ、Go-WASM と Rust-WASM の `janken` verification は CI 上で継続実行できる状態にする。

この plan は以下を成立させる。

- `make test` を fast default gate に戻し、WASM build + runner e2e を every PR の常設コストから外す
- Go-WASM verification を CI から失わず、WASM 関連変更では dedicated lane で自動実行する
- Rust-WASM evaluation path も CI 上で再現可能にしつつ、通常の Go gate へは混ぜない
- `docs/specs/go-quality-gates.md` と実際の test / workflow 配置の境界を一致させる

Addresses:

- `docs/issues/wasm-e2e-coverage-and-ci-cost.md`

## Context

- `docs/project-plan.md` は最終提出形式として WASM/WASI を要求している一方、Phase 9 では official support をまず Go に寄せ、Rust は先行 evaluation lane として扱う
- `docs/design-decisions/core-beliefs.md` は correctness over speed を求めているため、CI 境界の整理も spec を先に揃える
- `docs/design-decisions/adr.md` の Phase 2 実行戦略は、基礎 contract と後続の WASM 統合を切り分けて進める方針を採っている
- `docs/specs/go-quality-gates.md` は WASM verification を dedicated helper / targeted test として扱う方針だが、現状の `e2e/arena_runner_test.go` では Go-WASM 3 本が `make test` に常設で含まれている
- 現状観測では Go-WASM 3 本だけで約 17 秒、`make test` 全体で約 37 秒かかっており、WASM path が default gate の大きい割合を占めている

## Scope

- default Go gate と dedicated WASM gate の責務分離
- Go-WASM / Rust-WASM verification test の実行境界見直し
- WASM 専用 GitHub Actions workflow の追加または再編
- `docs/specs/go-quality-gates.md` の CI 契約更新
- relevant path 変更時だけ WASM lane を自動実行する trigger 設計

この plan では以下は扱わない。

- WASM runtime contract 自体の再設計
- Go / Rust sample AI の仕様変更
- 他言語 WASM support の追加
- branch protection 設定そのものの変更

## Design Decision

この plan では追加 ADR は作らない。

- `make test` / `make lint` は引き続き default required gate とし、通常 PR の fast feedback を優先する
- Go-WASM と Rust-WASM は dedicated workflow へ分離し、WASM 関連 path に限定して CI で流す
- Go-WASM は supported path の継続検証として dedicated lane の主対象にし、Rust-WASM は experiment/evaluation lane として同じ workflow か隣接 workflow に分離して扱う
- Rust-WASM を「CI で回すこと」と「default gate に入れること」は分け、通常 Go gate へは混ぜない

これは WASM support policy の変更ではなく、既存の support boundary に合わせて CI 配置を整理する運用判断として扱う。

## Spec Changes

### `docs/specs/go-quality-gates.md`

以下を明記する。

- default quality gate は `make test` / `make lint` のままとし、WASM `janken` verification は dedicated CI lane で扱う
- Go-WASM verification は supported path の targeted automated check として dedicated workflow で継続実行する
- Rust-WASM verification は experiment/evaluation lane として CI で再現可能にするが、default Go gate には混ぜない
- WASM lane の自動実行対象は runtime / runner / testdata / workflow など relevant path に限定してよい

### Optional spec touch if wording drift remains

- `docs/specs/ai-runtime.md` または `docs/specs/janken-game.md` に CI lane の扱いが必要以上に固定されている場合は、support boundary にだけ表現を揃える

## Expected Code Changes

### `e2e/arena_runner_test.go` または隣接 test 配置

- Go-WASM 常設 3 本を default `go test ./...` から外し、dedicated WASM lane だけで走る形へ移す
- Rust-WASM evaluation test と Go-WASM test の実行条件を揃え、default gate と dedicated lane の境界をコード上でも明確にする
- 実行切り替えは build tag、environment variable guard、または dedicated test command のいずれかで実装するが、最終的に `make test` では WASM lane が走らないことを保証する

### `Makefile`

- default gate 用 target と WASM verification 用 target を分ける
- Go-WASM verification を呼ぶ dedicated target を追加または整理する
- Rust-WASM verification を CI から呼べる dedicated target を追加または整理する
- 既存の visible manual helper (`run-janken-go-wasm`, `run-janken-rust-wasm-eval`) は必要ならそのまま残し、CI 用 target とは責務を分ける

### `.github/workflows/*.yml`

- WASM 専用 workflow を追加し、Go-WASM job と Rust-WASM job を分離して実行できるようにする
- trigger は `pull_request` / `push` の relevant path filter を基本にし、必要なら `workflow_dispatch` を併設する
- Go-WASM / Rust-WASM 各 job の toolchain setup と cache 方針を分離し、通常の `go-ci.yml` へ Rust toolchain を持ち込まない
- workflow の `uses:` 更新は `pinact` で行う

## Sub-tasks

- [ ] Update `docs/specs/go-quality-gates.md` so default gate and dedicated WASM lanes are explicitly separated
- [ ] Decide the concrete test-selection mechanism that keeps WASM tests out of plain `make test`
- [ ] Move Go-WASM verification off the default `go test ./...` path while preserving automated CI coverage
- [ ] Add or refine dedicated Make targets for Go-WASM CI verification
- [ ] Add or refine dedicated Make targets for Rust-WASM CI verification
- [ ] Add a WASM-focused GitHub Actions workflow with relevant path filters
- [ ] Keep `go-ci.yml` focused on the fast default gate and free from Rust toolchain setup
- [ ] Verify the issue can be closed by showing that default gate no longer runs WASM e2e, while dedicated CI still runs both Go-WASM and Rust-WASM lanes

## Parallelism

- [parallel] Spec wording and workflow YAML drafting can proceed independently once the lane split is fixed
- [parallel] Go-WASM target extraction and Rust-WASM target extraction can proceed independently
- Workflow verification depends on the final Make target names and test-selection mechanism

## Risks and Mitigations

- test selection を ad-hoc な env guard だけで増やすと、どの lane が何を保証するのか分かりにくくなる
  - mitigation: `make test` と dedicated WASM targets の責務を spec と Makefile の両方で明示する
- Rust toolchain setup を通常 Go CI に混ぜると、every PR の runtime と cache が重くなる
  - mitigation: Rust-WASM は独立 workflow/job に分離し、relevant path 変更時だけ自動実行する
- Go-WASM を default gate から外す際に coverage が実質的に弱くなる恐れがある
  - mitigation: Go-WASM lane 自体は CI 常設とし、runtime/runner/testdata/workflow 変更では自動で走るようにする
- path filter が狭すぎると WASM 関連変更を取りこぼす
  - mitigation: runtime、runner、`testdata/ai/janken`、`e2e/`、`Makefile`、workflow file など lane に効く path を明示的に列挙する

## Verification

The execution PR is complete when the following are true.

- Local `make test` no longer runs the Go-WASM or Rust-WASM `janken` verification path
- Dedicated local/CI targets exist for both Go-WASM and Rust-WASM verification
- The WASM workflow runs automatically for relevant-path PRs and pushes
- The normal `go-ci.yml` path no longer installs Rust toolchain or pays WASM build/e2e cost
- `docs/specs/go-quality-gates.md` matches the final test/workflow boundary
