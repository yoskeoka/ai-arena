# dungeon sidecar が `internal/platform/*` に依存している

## Summary

`platform-phase5-01-dungeon-fixed-map-mvp` では、`cmd/dungeon-bot-local/main.go` の
`internal/platform/protocol` 依存は外したが、`cmd/dungeon-gamemaster/main.go` は依然として
`internal/platform/catalog` / `internal/platform/game` / `internal/platform/gamemaster` /
`internal/platform/protocol` へ依存している。

Phase 5 plan の意図は「mono repo 上に置いているだけで、dungeon game 側は別 repo へ切り出せる
境界を先に固定する」ことだったため、local subprocess の sidecar 実装が `internal` 前提のままだと
その境界が中途半端になる。

対象:

- `cmd/dungeon-gamemaster/main.go`
- 必要なら `cmd/dungeon-bot-local/main.go` と合わせた sidecar 共通 I/O 層
- `cmd/dungeon-map-helper/main.go` のような game-design / debug / verification 用 CLI
- `games/dungeon/` と platform 側 adapter の責務分離

## Why It Matters

- `dungeon` を将来別 repo へ移したいとき、sidecar entrypoint がそのまま持ち出せない
- 「game domain は top-level non-internal package tree」という今回の方針と説明がずれる
- subprocess verification path が platform 実装詳細に引きずられ、外部実装の検証価値が落ちる

## Proposed Solution

次のどちらか、またはその組み合わせで refactor する。

1. sidecar 向けの最小 public transport / DTO を `internal` 外へ切り出す
   - 例: JSON-RPC NDJSON の最小 envelope と `initialize_match` / `current_snapshot` などの
     DTO を `pkg` または `games/dungeon/sidecar` 相当へ出す
2. `cmd/dungeon-gamemaster` を platform adapter と dungeon implementation に分離する
   - command 側は公開 DTO だけを使い、platform `internal` 依存は adapter 側へ閉じ込める

どちらの形でも、`cmd/dungeon-bot-local` と同様に
「mono repo に置いてあるが、別 repo 実装として成立する」ことを機械的に確認できる構造へ寄せたい。

補足:

- `cmd/dungeon-map-helper` のような game-design / game-debug / verification 用 CLI 自体は許容する
- ただしそれらも dungeon game 本体と同様に、将来別 repo へそのまま引っ越せる依存境界で作る
- `games/dungeon/*` や sidecar / helper command の冒頭には、
  「mono repo に置いているだけで、将来別 repo へ移せる前提のコードである」ことを
  明示する短い package / file comment を入れる方針も検討対象とする

## Scope Boundary

- 今回の PR では `dungeon-bot-local` の `internal` 依存除去までに留める
- `dungeon-gamemaster` を含む sidecar boundary の整理は別 PR で扱う
