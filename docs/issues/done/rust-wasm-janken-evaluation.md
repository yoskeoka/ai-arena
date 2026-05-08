# Rust WASM `janken` evaluation record

## Summary

Phase 4 の multi-language evaluation では、Rust を最初の non-Go candidate として評価した。
この評価は完了済みであり、結果は以下の通り。

- official support は引き続き Go のみ
- Rust は `experiment-only` lane として評価経路を追加済み
- Rust sample、build helper、`janken` verification path は repo に追加済み
- local verification では Rust-WASM module を build し、`janken` match 完走を確認済み

## Completed verification

前提:

- `rustup target add wasm32-wasip1`

実施済み確認:

- `make build-janken-rust-wasm`
- `make run-janken-rust-wasm-eval`
- `AI_ARENA_EXPERIMENT_RUST_WASM=1 go test ./e2e -run TestArenaRunnerJankenRustWASMEvaluationPath -count=1`

確認できたこと:

- `janken` match は `completed` で終了した
- `<output-dir>/<match-id>/record.json` と `history.json` が生成された
- Rust-WASM player の `stderr_bytes` が 0 より大きく、runtime 経由で `stderr` capture が効いた
- Go supported path を変えずに、同じ runtime contract で non-Go module を流せた

## Notes

- 最初の build blocker は `wasm32-wasip1` target 未導入だった
- これは未解決問題ではなく、Rust evaluation lane の実行前提である
- TypeScript / Python は将来候補として残すが、この記録では扱わない
- Ruby は今回の優先対象から外した
