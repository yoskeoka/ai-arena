# public-match-list-filters
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## 目的と完了境界

匿名 public spectator の `GET /api/v1-alpha/public/matches` を、全 discoverable match の暗黙順・無制限配列ではなく、game scope を server 側で絞り込んだ安定したページとして取得できるようにする。Reversi visualizer は `game_id=reversi` と `game_version_major=1` を固定し、必要な ruleset だけを選べる契約入力にする。

query はすべて任意とし、`page` は 1 始まり・既定 `1`、`limit` は既定 `20` かつ `1..100`、`sort=completed_at`、`sort_order=asc|desc` とする。`sort` の既定は `completed_at`、`sort_order` の既定は `desc` とする。filter は `game_id` 完全一致、`game_version_major` の semantic-version major 一致、`ruleset_version` 完全一致とする。`completed_at` を持たない discoverable record は completed record の後ろに置く。completed timestamp が同じ record 同士と timestamp を持たない record 同士は、`sort_order` と無関係に `match_id` の昇順で tie-break する。ページ応答は top-level の `pagination`、`available_ruleset_versions`、`items` を返す。`pagination` は要求に反映された `page`、`limit`、filter 後かつ page 切り出し前件数 `total`、および `ceil(total / limit)` の `total_pages` を持つ。`total=0` の `total_pages` は `0` とする。`available_ruleset_versions` は game ID と major filter に合う全 discoverable record から昇順・重複なしで導き、ruleset filter と page によって狭めない。例えば `{"pagination":{"page":1,"limit":20,"total":53,"total_pages":3},"available_ruleset_versions":["standard","xot"],"items":[...]}` とする。

無効な整数、範囲外の page/limit、未対応の sort/order、major が正の整数でない query は HTTP 400 にする。既存の selector / anonymous / public-only 境界、selected-run の規則、detail/state/replay resource の key は変更しない。implementation PR の完了は local/CI contract verification と plan cleanup までとし、staging acceptance は main merge 後に online release workflow が同じ SHA を deploy して所有する post-merge verification として別に記録する。staging accepted と報告するのは、その workflow と credential-free 実データの証跡がそろった後だけにする。

## 現状と根拠

- `typespec/namespaces/public/api.tsp:10-40` は list response を `items` だけで定義し、query parameter と page metadata を持たない。
- `internal/platform/service/public_http.go:26-40` は list query を parse せず `PublicQueryService.List` の全結果を返す。
- `internal/platform/service/public_state.go:170-180,250-279` は selected public record を map iteration のまま返し、filter、sort、page を適用しない。
- `internal/platform/service/store_memory.go:195-207` と `internal/platform/service/store_postgres.go:340-371` は service 側で選択済み record の集合を取得する read boundary である。実装は storage に game-specific public projection を渡さない。
- 2026-09-16 に `https://ai-arena-staging-p4ml.onrender.com/version` は `1572fb…` を返し、同 public match list/detail は `completed_at` と `participants` を返さない。現行 main の `ff85f5e` と `2bef4b7` はそれらの public metadata を追加しているため、staging は viewer 契約より古い。
- `docs/specs/platform-public-spectator.md:1-54` は anonymous GET-only の public-only 境界、selected run、completion timestamp / participant provenance を定める。

## 変更マップ

- (MODIFY) `typespec/namespaces/public/api.tsp`: list query model、pagination object、scope-wide available ruleset metadata を wire-contract の正本として追加する。
- (MODIFY) `docs/specs/platform-public-spectator.md`: filter、page、completed timestamp sort、validation failure、安定順序の observable behavior を追記する。
- (MODIFY) `internal/platform/service/public_http.go`: query を strict に decode し、400/503 と public CORS を保ったまま list options を渡す。
- (MODIFY) `internal/platform/service/public_state.go`: selected public record を filter、sort、tie-break、page して、pagination と scope-wide available ruleset metadata を含む page result を組み立てる。
- (MODIFY) `internal/platform/service/public_http_test.go` と `internal/platform/service/public_state*_test.go`: default、境界値、各 filter、複合 filter、asc/desc、null completion、tie-break、page、invalid query、public-only leakage regression を検証する。
- (MODIFY) generated TypeSpec/OpenAPI artifacts: TypeSpec source から再生成し、source と emitted contract の一致を保つ。
- (NO CHANGE) detail/state/replay endpoint、private artifact boundary、operator history endpoint、game-specific UI。

## 実施項目

1. public spectator behavioral spec と TypeSpec を先に更新し、query 名、defaults、範囲、comparison、response metadata、error を固定する。`+/-` の sort shorthand は導入しない。
2. transport adapter に small typed list-option decoder を追加する。unknown/invalid query を曖昧に default せず 400 にし、public service error は既存どおり 503 に正規化する。
3. public read service で selected logical match を作った後に filter し、completed timestamp と常に昇順の `match_id` の全順序で sort してから page を切り出す。page metadata の `total` は filter/sort 後、page 前の件数とする。game/major scope から deduplicate した available ruleset metadata は、ruleset filter や page 外にある値も保持する。
4. TypeSpec artifact を regenerate し、HTTP と service の regression tests を追加する。in-memory と Postgres-backed queue で storage order に依存しないことを確認する。
5. implementation PR では staging acceptance を要求しない。merge 後の online release staging verification が deploy SHA、scoped first page、detail、replay を credential-free request で確認し、historic record の provenance/replay が unavailable の場合は新しい completed Reversi match で acceptance evidence を残す。

## 依存関係と並行性

- Reversi 側の `0005-public-match-discovery-filters` はこの TypeSpec/API 実装を入力にする。AI Arena の implementation PR が merged/deployed するまでは、visualizer は新しい query を呼ばない。
- stg の旧 SHA 更新は API implementation の merge 後に online release workflow が所有する。同一 API contract に依存する UI 実装は、mocked response/tests を先行してよいが、実 service acceptance は deployment verification 完了後に直列で行う。

## 検証

- `pnpm --dir typespec run build` で emitted public OpenAPI を再生成する。
- public HTTP/service tests で query validation、filter、pagination、complete-time ordering、tie-break、匿名 CORS、private field non-disclosure を検証する。
- relevant Go quality gates を writable task cache で実行する。
- implementation PR は staging acceptance を完了と主張しない。post-merge の stg verification で `game_id=reversi&game_version_major=1&page=1&limit=20&sort=completed_at&sort_order=desc` が Reversi major-1 records、pagination、available ruleset metadata を返し、新しい completed Reversi record が visualizer required metadata と `reversi/replay` を返すことを確認する。

## 非目標

- cursor pagination、multi-field sorting、lifecycle filter、public run selection、private artifact access、event stream、visualizer の hosting / rendering は対象外。
