# dungeon external repo SDK release note

このメモは、`dungeon-game-ai-arena` のような external game repo が
`github.com/yoskeoka/ai-arena/gamemaster` を stable dependency として使うときの
最小運用を固定する。

## release tag の役割

- external repo が ai-arena へ安定依存してよい公開 import surface は
  `github.com/yoskeoka/ai-arena/gamemaster` package までとする
- external repo は local workspace checkout や `replace ../ai-arena` ではなく、
  review 済み ai-arena module tag を `go.mod` から参照する
- ai-arena runner / platform host の更新は、consumer repo 側では「ai-arena version を上げる」
  操作として取り込む

## `v0.1.0` の gate

`v0.1.0` を切る前に、少なくとも以下を確認する。

- `gamemaster` package の公開 DTO / NDJSON helper が sidecar 開発に必要な最小面を満たしている
- `cmd/dungeon-gamemaster` が `games/dungeon/...` と `gamemaster` package だけで build できる
- external repo 側 import audit で `gamemaster` を越える ai-arena 依存が残っていない
- external repo 側で local `replace` なしの tagged import build/test/CI が通る
- same-golden local / CI e2e が tag 切替後も維持される

## golden 更新の扱い

- tagged import 切替だけで same-condition regression が壊れた場合、まず移設不備または host version drift として調べる
- golden 更新を許可するのは、`game_version` / `ruleset_version` / deterministic AI / normalized result shape の意図的変更に加えて、
  external repo が意図的に採用する ai-arena runner / platform version を上げた場合に限る
- golden を更新したときは、PR と plan/spec に「どの ai-arena version change を採用したか」を残す
