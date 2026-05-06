# platform-phase4-wasm-runtime

## Objective

WASM/WASI を AI の正式な実行経路として成立させ、ダンジョンゲームの AI player を WASM 前提で開発できる状態にする。検証はまず Go でビルドした WASM を基準とし、多言語 WASM の検証は実装コストを見ながら `janken` 側で進められるようにする。

## Goal

- AI 提出形式として想定している WASM/WASI を、開発中の暫定扱いではなく正式な実行経路として扱える状態にする
- ダンジョンゲームの AI player を Go から WASM へビルドして動かす前提を固定する
- `janken` を WASM 実行検証の先行ゲームとして使い、Go 製 AI の検証を優先して進められる状態にする
- 多言語でビルドした WASM の検証を、ダンジョンゲーム本体ではなく `janken` 側で評価できる位置づけにする
- Phase 5 のダンジョンゲーム開発が、WASM 実行方式の未確定さで止まらない状態にする

## Out of Scope

- 実装方式の確定
- wazero まわりの詳細設計
- 多言語 WASM サポート範囲の確定
- 実装着手

## Follow-up

- この plan は phase goal を固定するための skeleton であり、runtime adapter、検証項目、sample AI、CI 方針などの HOW は後続の詳細 plan に分割して扱う
- Go 製 WASM の成立確認と、多言語 WASM の扱いは別 plan で優先順位を分けて具体化する
