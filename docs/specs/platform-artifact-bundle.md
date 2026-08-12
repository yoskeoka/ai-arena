# Artifact bundle 仕様

## 目的

official game / AI submission は immutable な `arena-bundle/v1` ZIP として受け付ける。
manifest の正本は `schemas/arena-bundle-v1.schema.json` と typed Go model である。

## Observable behavior

- archive は root の `manifest.json` と manifest が宣言する単一 WASM module だけを持つ。
- admission は archive path、重複 entry（大文字・小文字だけが異なるものを含む）、link、undeclared entry、WASM magic/version を拒否する。受け付ける archive は 64 MiB 以下、manifest は 1 MiB 以下、module の展開後サイズは 32 MiB 以下であり、各 entry の圧縮率は 100:1 以下でなければならない。
- manifest は schema と同じ required field・kind ごとの形・resource budget を満たす。AI bundle では `ai_id` を必須とする。optional な数値 budget は未指定なら許可するが、明示した `0` は schema の minimum 違反として拒否する。runtime は compile 可能な WASI WebAssembly module であり、function、memory、table、global のいずれの import も未許可 module を要求してはならない。検証失敗は保存も registry 登録も行わない。
- service は bundle bytes の SHA-256 digest を artifact identity として返し、同一 bytes を idempotently 保存する。
- filesystem と R2 の bundle store は同じ digest と同じ private materialization layout を返す。同一 digest への並行保存でも digest と異なる既存 bytes を上書きしない。
- operated-service の受入証跡は、同一の検証済み game / AI bundle bytes を filesystem と S3-compatible store の両方へ upload し、registry 登録、queue acceptance、worker materialization、および game / AI WASI session start まで完走させる。完走した run は accepted digest を game と全 seat に保持し、temporary materialization directory を残さない。外部 game の release asset を検証する場合は、公開 URL と `SHA256SUMS` の pinned digest を照合してから同じ bundle-only runner input に渡す。
- game bundle は ruleset と player-count / game-defined bot limit を宣言し、AI bundle は `ai_id` と対象 game/ruleset を宣言する。AI admission は selected game registration と game id、semver major、ruleset が互換であることを同期的に確認する。
- game release は `game_id + major` で lookup し、admitted release の semver 最大を選ぶ。request acceptance 時には、その時点で選択された game digest と各 AI digest を run に snapshot する。retry / rerun は snapshot を再解決せず、同じ digest を引き継ぐ。
- worker は game / AI の各 digest を private directory へ materialize して同じ match runtime を起動する。AI の試合記録上の identity は seat の `player_id` ではなく、admitted immutable artifact digest とする。materialization directory は player session の終了後に削除する。queue と worker は caller 提供の host path / URL を artifact identity として受け取らない。match の deadline/cancellation は registry lookup、session construction、WASI process lifetime に一貫して適用され、短命な resolver lookup context に session lifetime は束縛されない。host filesystem と network capability は付与しない。
