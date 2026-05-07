# Rust WASM `janken` evaluation lane

## Summary

Phase 4 の multi-language evaluation では、Rust を最初の non-Go candidate として扱う。
official support は Go のみとし、Rust は `experiment-only` lane に留める。

## Expected success path

以下を再現できれば、Phase 4 の Rust evaluation は成功扱いとしてよい。

- prerequisite:
  - `rustup target add wasm32-wasip1`
- `make build-janken-rust-wasm`
- `make run-janken-rust-wasm-eval`

2026-05-08 のローカル確認では、最初の build blocker は `wasm32-wasip1` target 未導入だった。
target 追加後は、上記 helper と `AI_ARENA_EXPERIMENT_RUST_WASM=1 go test ./e2e -run TestArenaRunnerJankenRustWASMEvaluationPath -count=1`
で `janken` evaluation lane を再現できた。

成功時に確認すること:

- `janken` match が `completed` で終わる
- `<output-dir>/<match-id>/record.json` と `history.json` が生成される
- Rust-WASM player の `stderr_bytes` が 0 より大きい
- Go supported path を変えずに、同じ runtime contract で non-Go module を流せる

## Build blockers to isolate

- `cargo` / `rustup` が見つからない
- `rustup target add wasm32-wasip1` 未実施
- `cargo build --target wasm32-wasip1 --release` 自体が失敗する
- build output を sidecar manifest が期待する `.wasm` path へ配置できない

## Runtime blockers to isolate

- `arena-runner` が manifest を解決できても module instantiate に失敗する
- `stdout` が JSON-RPC NDJSON 以外を混在させ、`invalid-protocol-malformed` になる
- `stdin` close 後の shutdown が収束せず、forced shutdown 扱いになる
- WASI capability 不足または toolchain 依存差分で runtime-stopped になる

## Boundary

- TypeScript / Python は将来候補として残すが、この note では扱わない
- Ruby は今回の優先対象から外す
- この note は evaluation 観測の受け皿であり、外部 developer guide ではない
