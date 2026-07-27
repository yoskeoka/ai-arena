# Artifact bundle 仕様

## 目的

official game / AI submission は immutable な `arena-bundle/v1` ZIP として受け付ける。
manifest の正本は `schemas/arena-bundle-v1.schema.json` と typed Go model である。

## Observable behavior

- archive は root の `manifest.json` と manifest が宣言する単一 WASM module だけを持つ。
- admission は archive path、重複 entry、link、undeclared entry、WASM magic を拒否する。
- service は bundle bytes の SHA-256 digest を artifact identity として返し、同一 bytes を idempotently 保存する。
- game release は `game_id + major` で lookup し、admitted release の semver 最大を選ぶ。開始済み match は選択済み digest を固定する。
- worker は digest を private directory へ materialize して WASI runtime を起動する。host filesystem と network capability は付与しない。
