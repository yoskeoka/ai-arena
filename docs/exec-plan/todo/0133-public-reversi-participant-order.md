# public-reversi-participant-order
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: https://github.com/yoskeoka/ai-arena/issues/365

## 目的と完了境界

匿名 public spectator API の既存 `participants` 配列を変更せず、完了した standard Reversi match では index 0 を Black、index 1 を White と解釈できる契約にする。各要素は現在の `display_name` と `ai_submission_id` を使い、viewer が private data、mutable registry、推測によって色や bot 名を補完する必要がない状態を完了境界とする。

この plan は object shape への migration、`bot_name` への rename、run selector の追加を行わない。Reversi 以外の game は引き続き game ruleset に従って participant order を解釈する。

## 現状と根拠

- `typespec/namespaces/public/api.tsp:13-25` は public match の任意 `participants` 配列を `player_id`、`display_name`、`ai_submission_id` として定義している。
- `docs/specs/platform-public-spectator.md:34-39` は admission 時の participant provenance を submitted player order のまま返し、consumer が game ruleset と配列順を解釈すると定めている。
- `internal/platform/service/request.go:244-280` は request participant の順序で immutable submitted player を作り、`internal/platform/service/public_state.go:283-301` は同順序で public projection を作る。
- `internal/platform/service/worker_local.go:112-140` は submitted player order で game master へ player を渡す。Reversi provider の `games/reversi/src/gamemaster.rs:389-402` は先頭 player から Black の決定を要求する。
- `internal/platform/service/public_http_test.go:18-49` は projection の順序を unit test するが、standard Reversi の list/detail/state public HTTP response を Black/White semantics として結合検証していない。

## Change map

- (MODIFY) `docs/specs/platform-public-spectator.md`: existing generic participant-order rule を参照した上で、completed standard Reversi の two-player positional meaning を observable public behavior として定める。
- (MODIFY) `typespec/namespaces/public/api.tsp`: existing wire fields を維持したまま、generated public contract へ Reversi-only positional interpretation を説明として載せる。
- (MODIFY) `internal/platform/service/public_http_test.go`: distinct Black/White submissions を用い、anonymous list/detail/state が順序、公開名、immutable revision、completed timestamp を正しく返し、private fields を出さないことを検証する。
- (NO CHANGE) Postgres schema, operator API, participant object shape, and public route family: this delivery only documents and proves an existing compatible contract.

## Black-box specification changes

1. A completed public match whose game metadata identifies standard Reversi exposes exactly two complete participant provenance entries when the selected run has complete provenance. Entry 0 is the Black bot and entry 1 is the White bot. Each keeps its public `display_name` and immutable `ai_submission_id`; the browser retains the full values for requests and displays only its own safe presentation form.
2. The same selected-run participant order and immutable terminal `completed_at` are observable from list, detail, and state responses. Promotion or later mutable metadata updates do not change the selected response's pinned participant identities or completion time.
3. The endpoint remains credential-free and GET-only. It never exposes role-inference inputs or private provenance such as bot IDs, artifact references, locators, credentials, records, snapshots, histories, or run selection controls.

## Work

1. Add a concise Reversi-specific clause to the public spectator specification. Keep the generic array contract authoritative for other games and state that the clause applies only to completed standard Reversi two-player matches.
2. Add matching TypeSpec documentation without renaming or reshaping `PublicParticipant`. Regenerate the checked-in API artifacts if the TypeSpec build changes them.
3. Extend the public HTTP test setup with distinct first/second submitted Reversi players and a completed selected run. Exercise list, detail, and state through the anonymous route rather than only calling the projection helper.
4. Assert positional Black/White semantics, `display_name`, `ai_submission_id`, immutable `completed_at`, selected-run consistency, CORS, and the absence of private fields in every response. Keep malformed or historical incomplete provenance behavior unchanged.
5. Run the scoped Go tests, TypeSpec generation/checks, and applicable repository quality gates. Delete this completed plan only from its later execution branch after evidence is captured and its implementation PR is prepared.

## Dependencies and parallelism

This plan depends on the merged public metadata delivery in PR #364. Documentation/TypeSpec clarification and route-test authoring can proceed in parallel after the fixture shape is agreed; generated-artifact review follows TypeSpec changes. The Reversi visualizer may consume the existing compatible array contract, but must reject incomplete or non-two-player Reversi metadata until this contract is verified on its selected API base.

## Verification

- TypeSpec/public artifact checks preserve the existing `participants[]` field names and do not add a private or operator route.
- Public HTTP tests prove list/detail/state return the selected completed Reversi run with index 0 Black and index 1 White identities, an immutable completion time, and no private fields.
- Existing generic participant-order, legacy-incomplete-provenance, selected-run, and anonymous CORS tests remain green.
- Run the repository's applicable Go, TypeSpec, and workflow quality gates before the implementation PR.

## Non-goals

- Changing `participants[]` into Black/White objects or renaming `display_name`.
- Assigning player colours for arbitrary games or adding a platform-level role field.
- Exposing run IDs as controls, historical attempts, operator metadata, or private replay inputs.
