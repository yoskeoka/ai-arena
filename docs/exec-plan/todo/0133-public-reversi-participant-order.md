# public-reversi-participant-order
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: https://github.com/yoskeoka/ai-arena/issues/365

## 目的と完了境界

匿名 public spectator API の既存 `participants` 配列を変更せず、完了した standard Reversi match では index 0 を Black、index 1 を White と解釈できる契約にする。各要素は現在の `display_name` と `ai_submission_id` を使い、viewer が private data、mutable registry、推測によって色や bot 名を補完する必要がない状態を完了境界とする。

この plan は object shape への migration、`bot_name` への rename、run selector の追加を行わない。Reversi 以外の game は引き続き game ruleset に従って participant order を解釈する。

## 現状と根拠

- `typespec/namespaces/public/api.tsp:13-27` は public match の任意 `participants` 配列を `player_id`、`display_name`、`ai_submission_id` として定義している。
- `docs/specs/platform-public-spectator.md:34-39` は admission 時の participant provenance を submitted player order のまま返し、consumer が game ruleset と配列順を解釈すると定めている。
- `internal/platform/service/request.go:244-280` は request participant の順序で immutable submitted player を作り、`internal/platform/service/public_state.go:283-301` は同順序で public projection を作る。
- `internal/platform/service/worker_local.go:112-140` は submitted player order で game master へ player を渡す。`yoskeoka/reversi-ai-arena` の main（PR #47 作成時点）の `games/reversi/src/gamemaster.rs:389-402` は先頭 player から Black の決定を要求する。
- `internal/platform/service/public_http_test.go:18-49` は projection の順序を unit test するが、standard Reversi の list/detail/state public HTTP response を Black/White semantics として結合検証していない。

## Change map

- (MODIFY) `docs/specs/platform-public-spectator.md`: existing generic participant-order rule を参照した上で、completed standard Reversi の two-player positional meaning を observable public behavior として定める。
- (MODIFY) `typespec/namespaces/public/api.tsp`: existing wire fields を維持したまま、generated public contract へ Reversi-only positional interpretation を説明として載せる。
- (MODIFY) `internal/platform/service/public_http_test.go`: distinct Black/White submissions を用い、anonymous list/detail/state が順序、公開名、immutable revision、completed timestamp を正しく返し、private fields を出さないことを検証する。
- (NO CHANGE) Postgres schema, operator API, participant object shape, and public route family: this delivery only documents and proves an existing compatible contract.

## Black-box specification changes

1. `game_id = "reversi"`、major version 1、`ruleset_version = "standard"` の completed selected run は、Reversi provider が two-player admission を完了し provenance が完全な場合に限り、ちょうど二つの participant entry を返す。entry 0 は Black、entry 1 は White であり、各 entry は public `display_name` と immutable `ai_submission_id` を保持する。
2. 同じ selected run の participant order と terminal `completed_at` は list、detail、state で不変に観測できる。promotion は selected run 自体を切り替え、新しく選ばれた run の identity と completion time を返してよい。
3. endpoint は credential-free、GET-only のままとする。公開の participant order 以外の private/internal role-inference input、bot ID、artifact reference、locator、credential、record、snapshot、history、run selection control は出さない。

## Work

1. public spectator specification と TypeSpec に Reversi 固有の位置的意味を追記する。他 game の generic array contract と既存 `PublicParticipant` の wire field は変更しない。
2. distinct first/second Reversi player を持つ completed selected run で anonymous list/detail/state を検証する。exactly-two precondition を満たさない Reversi record はこの保証の対象外であることも negative test で示す。
3. Black/White order、`display_name`、`ai_submission_id`、immutable `completed_at`、promotion による selected-run switch、CORS、private field 非露出を検証する。
4. scoped Go test、TypeSpec generation/check、repository quality gate を実行する。完了した plan は後続の execution branch で PR 準備後に削除する。

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
