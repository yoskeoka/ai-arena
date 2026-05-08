# WASM e2e coverage と CI 実行時間の扱いを整理する

## Summary

Phase 4 時点の `e2e/arena_runner_test.go` には、WASM 関連の検証が少なくとも以下の 4 本ある。

- 常設:
  - `TestArenaRunnerJankenGoWASMMixedRuntimePath`
    - Go-WASM module をその場で build し、`janken` を mixed runtime で完走させる
  - `TestArenaRunnerJankenGoWASMMissingModuleFails`
    - Go-WASM manifest の module 欠落を起動前 failure として確認する
  - `TestBuildGoWASMReportsBuildFailure`
    - Go-WASM build helper の failure path を確認する
- opt-in:
  - `TestArenaRunnerJankenRustWASMEvaluationPath`
    - `AI_ARENA_EXPERIMENT_RUST_WASM=1` のときだけ Rust-WASM module を build して `janken` evaluation lane を流す

現行 CI (`.github/workflows/go-ci.yml`) は `make test` を常設で実行するため、Go-WASM 系 3 本は default gate に含まれる。
一方で Rust-WASM は environment variable guard があるため、現行の CI では常設実行されていない。

つまり、現時点で CI 実行時間に直接効いているのは主に Go-WASM 側であり、Rust-WASM はまだ常設コストに入っていない。
ただし、Rust-WASM を将来常設化する場合は `cargo` / `rustup target add wasm32-wasip1` / Rust build 時間が上乗せされる。

## Impact

- `make test` が default quality gate である以上、Go-WASM fixture build + runner e2e が every PR の `go-test` job に乗る。
- `docs/specs/go-quality-gates.md` では WASM verification helper を dedicated helper / targeted test として扱う方針を書いている一方、
  実際の e2e には Go-WASM 常設 path が入っており、default gate と dedicated lane の境界が少し分かりにくい。
- Rust-WASM は現時点では opt-in なので CI 時間をまだ増やしていないが、将来に常設化判断をする前に
  「Go-WASM 常設 verification をどこまで default gate に残すか」を整理しないと runtime matrix が増えたときに CI コストが読みにくくなる。

## Follow-up

- `go-test` job の中で WASM 関連 e2e が占める時間を一度測定し、通常 e2e と分けて可視化する。
- Go-WASM の happy path / negative path を default gate に残すか、専用 job または opt-in lane に切り出すかを判断する。
- Rust-WASM を常設 CI に昇格させる場合は、Go CI へそのまま混ぜず、別 job / 別 workflow / 明示 opt-in trigger のいずれかで扱う案を比較する。
- `docs/specs/go-quality-gates.md` と実際の test 配置の boundary がずれて見えないよう、spec か test layout のどちらかを揃える。
