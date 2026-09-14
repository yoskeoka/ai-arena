# admitted WASI game master の public replay を保持する

> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## 目的と完了境界

admitted WASI game master が valid な public replay を提供した完了 match で、匿名の観戦者が
terminal replay を取得できるようにする。platform は runtime cleanup 境界をまたいで game
生成 replay を保持し、既存の bounded replay artifact と metadata を永続化する。

完了は回帰テストを伴う platform の限定修正とする。修正前に完了した terminal record の
backfill、public HTTP / TypeSpec schema の変更、Reversi 固有 replay payload の decode は対象外とする。

## 根拠と現在の障害

- `internal/platform/registry/wasm_resolver.go:32-58` は admitted WASI game-master
  session を materialize し、`cleanupSession` 経由で返す。
- `internal/platform/registry/wasm_resolver.go:66-79` は base session interface だけを
  embed するため、cleanup wrapper が optional game-master capability を保持しない。
- `internal/platform/match/match.go:549-574` は supplied session が
  `gamemaster.PublicReplaySession` を公開するときだけ terminal public replay を capture する。
- `internal/platform/service/worker_local.go:227-254` と
  `internal/platform/service/worker_s3.go:83-96` は match record が capture 済み replay を
  含むときだけ replay bytes と metadata を永続化する。
- staging match `match-0a225ea4-3eb4-42d7-86be-834c09ae4720` は Reversi `1.1.0` と
  available な final public state で完了した一方、public replay は unavailable である。release 済み
  Reversi game master は `current_public_replay` を提供するが、wrapper により runner が認識できない。

## 契約変更

`docs/specs/platform-public-spectator.md` に観測可能な platform の責務を追記する。admitted WASI
game master が valid terminal public replay を提供するとき、lifecycle/resource cleanup は既存 public
replay contract での replay の保持と公開を妨げてはならない。replay を提供しない game と artifact の
integrity / retention failure は、従来どおり unavailable とする。

wire response と unavailable semantics は変わらないため TypeSpec は変更しない。

## 変更マップ

- `(MODIFY) docs/specs/platform-public-spectator.md`: admitted WASI execution cleanup をまたぐ
  valid game-generated terminal replay の保持を規定する。
- `(MODIFY) internal/platform/registry/wasm_resolver.go`: shutdown 時 cleanup の挙動を変えず、
  materialize 済み game-master session の optional public-replay capability を cleanup session が
  保持・forward するようにする。
- `(MODIFY) internal/platform/registry/registry_test.go`: terminal replay を提供する admitted WASI
  descriptor/session を用いた focused regression を追加する。返却された session が capability を公開し、
  replay result を forward すること、および shutdown 時 cleanup を継続することを確認する。

## 実施手順

1. opaque で game-owned な replay、および public/private boundary を維持したまま、先に spectator
   contract を更新する。
2. wrapped session が対応するとき、WASI cleanup wrapper が `CurrentPublicReplay` を forward するように
   拡張する。non-provider に対して replay data を生成せず、既存の optional-capability absence/error
   behavior を返す。
3. admitted WASI descriptor を resolve する registry-level regression test を追加し、返却 session が
   optional replay provider を保持し、format、version、payload を変更せず forward することを確認する。
   wrapper が shutdown 時に materialized directory を削除することも確認する。
4. focused Go test と適用される repository quality gate を実行する。deploy 後に staging で新しい
   Reversi `v1.1.0` match を submit・完了させ、anonymous detail、state、replay endpoint が available
   で相互に整合することを確認する。修正前 match は受入根拠に使わない。

## 依存関係と除外範囲

- merge 済み public spectator API と release 済み Reversi `v1.1.0` game artifact に依存するが、
  両方ともすでに存在する。
- 本障害に `reversi-ai-arena` の変更は不要である。
- `docs/issues/0122-phase8-reversi-public-replay-fixture-contract.md` は open のままとする。本修正は
  publication を復旧するが、別途必要な cross-repository fixture contract は確立しない。

## 検証

- `go test ./internal/platform/registry ./internal/platform/match ./internal/platform/service`
- repository が定める `make test` と適用 lint target。
- 新たに完了した Reversi `1.1.0` match を使う staging acceptance: detail は replay availability と
  metadata を返し、`/state` は available かつ terminal、`/replay` は `format: "reversi/replay"`、
  `version: "1"`、valid payload bytes を返す。

## Addresses

N/A — staging verification 中に発見した production regression を記録する。対応する external issue はない。
