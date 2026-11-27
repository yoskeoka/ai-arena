# 0138 Janken の Rust フィクスチャにローカル lint を追加する

> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## 目的と完了条件

Janken の Rust フィクスチャに影響する変更を担当したエージェントが、実行先と同じ `wasm32-wasip1` 向けにローカルで Clippy を実行できるようにする。実装 PR の最新コミットで、フィクスチャの全ターゲット lint が既存の Rust-WASM ビルドと E2E に先立って成功し、既存の E2E も通った時点で完了とする。

[ワークスペース計画 0048](https://github.com/yoskeoka/vibe-coding-workspace/pull/172) の子計画。[Adventure 実装 PR](https://github.com/yoskeoka/reversi-adventure/pull/312) で、担当範囲だけの lint、ジョブ数の既定値と上書き、Cargo の失敗伝播を確認済み。ここでは同じローカル運用を単独の WASM フィクスチャに適用する。Reversi AI Arena の子計画とは独立して進められる。

## 仕様と変更箇所

- (MODIFY) `docs/specs/janken-game.md:252–282` の Phase 4 Multi-language Evaluation Lane と `AGENTS.md` に検証方法を記す。フィクスチャの Rust ソース、マニフェスト、ビルドスクリプト、または取り込み入力を変更した場合は、引き渡し前に対象の Clippy を実行する。共有入力やツールチェーンの変更でもフィクスチャへの影響を確認し、影響範囲を絞れない場合はローカル確認と CI の確認範囲を記録する。Rust に影響しない変更では Rust lint を省く。通常は lib／bin、テスト、example、bench と対応する条件付きコードの変更時は全ターゲットを選び、必要な非デフォルト feature は Cargo 直接実行時に明示する。引き渡しにはフィクスチャ、ターゲット、該当 feature、結果を記録する。
- (MODIFY) `Makefile:2–13,42,154–156,230–240`。フィクスチャのマニフェスト `testdata/ai/janken/janken-rust-wasm-ai/Cargo.toml` と `wasm32-wasip1` を使う `make rust-clippy-local` と `make rust-clippy` を追加する。パッケージ指定は不要。ローカルは `CLIPPY_ALL_TARGETS=1` の場合に全ターゲットを検査し、ジョブ数は未指定時に `2` として呼び出し側の指定を尊重する。CI 用の入口は常に `--all-targets` を渡す。両入口は `-- --no-deps -D warnings` を渡し、既存の `CARGO` と `CARGO_TARGET_DIR` に従う。同じビルドディレクトリを使う実 Cargo コマンドは直列に実行する。
- (MODIFY) `.github/workflows/wasm-verification.yml:4–31,60–97`。既存の Rust ターゲットに加えて Clippy コンポーネントを用意し、`rust-wasm` ジョブで `make rust-clippy` を既存の Rust-WASM 検証ステップより前に実行する。双方で既存の CI 用 Cargo キャッシュを使う。Janken フィクスチャと Makefile の既存の起動条件と、Go-WASM ジョブを維持する。

## 実施順序と検証

1. この計画のマージ後に仕様を更新し、続いて Make、エージェント指示、CI を変更する。Cargo 呼び出しの記録とワークフロー定義の確認は独立して進められる。
2. 記録用の代替 `CARGO` で、両コマンドのマニフェスト、WASI ターゲット、Clippy の引数、ローカルの全ターゲット切り替え、CI の全ターゲット固定、ジョブ数の既定値と上書き、Cargo の失敗伝播を確認する。
3. 実際のフィクスチャで `make rust-clippy-local` と `make rust-clippy` を実行し、`make test-wasm-rust` で既存の動作検証を行う。workflow lint と `actionlint` で変更した定義を確認する。
4. 実装 PR の最新コミットで全ターゲット lint と Rust-WASM E2E の実行順・結果を確認する。実装 PR に証跡を記録し、検証後にこの計画を削除して実装 PR をワークスペース計画 0048 に引き渡す。

計画 PR では Markdown の構造、参照先、`git diff --check` および workflow lint を確認する。
