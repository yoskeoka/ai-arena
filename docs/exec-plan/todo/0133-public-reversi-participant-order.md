# public-reversi-participant-order
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: https://github.com/yoskeoka/ai-arena/issues/365

## 目的と完了境界

匿名 public spectator API の既存 `participants` 配列を変更せず、match request の participant 順序を submitted player、game 処理、selected run の結果記録、public projection まで一貫して保持する契約にする。各要素は現在の `display_name` と `ai_submission_id` を使い、consumer が private data や mutable registry から participant identity を補完する必要がない状態を完了境界とする。

この plan は object shape への migration、`bot_name` への rename、run selector の追加、game 固有の role・color・人数の定義を行わない。順序の意味付けは各 game provider または consumer の責務であり、platform は順序を変更しない。

## 現状と根拠

- `typespec/namespaces/public/api.tsp:13-27` は public match の任意 `participants` 配列を `player_id`、`display_name`、`ai_submission_id` として定義している。
- `docs/specs/platform-public-spectator.md:34-39` は admission 時の participant provenance を submitted player order のまま返し、consumer が game ruleset と配列順を解釈すると定めている。
- `internal/platform/service/request.go:244-280` は request participant の順序で immutable submitted player を作り、`internal/platform/service/public_state.go:283-301` は同順序で public projection を作る。
- `internal/platform/service/worker_local.go:112-140` は submitted player order で game master へ player を渡す。
- `internal/platform/service/public_http_test.go:18-49` は projection の順序を unit test するが、distinct participant を持つ selected run の list/detail/state public HTTP response がこの順序を一貫して保持することを結合検証していない。

## Change map

- (MODIFY) `docs/specs/platform-public-spectator.md`: match request から public projection まで participant order を保持し、game 固有の意味付けを platform が行わない observable behavior を定める。
- (MODIFY) `typespec/namespaces/public/api.tsp`: existing wire fields を維持したまま、generated public contract に participant array の順序保持を説明として載せる。
- (MODIFY) `internal/platform/service/public_http_test.go`: distinct participant を用い、anonymous list/detail/state が順序、公開名、immutable revision、completed timestamp を正しく返し、private fields を出さないことを検証する。
- (NO CHANGE) Postgres schema, operator API, participant object shape, and public route family: this delivery only documents and proves an existing compatible contract.

## Black-box specification changes

1. complete provenance を持つ selected run は、match request で受け取った participant entry の個数と順序を保って public `participants` として返す。各 entry は public `display_name` と immutable `ai_submission_id` を保持する。platform は game 固有の role、color、人数、または配列順の意味を追加・検証しない。
2. 同じ selected run の participant order と terminal `completed_at` は list、detail、state で不変に観測できる。promotion は selected run 自体を切り替え、新しく選ばれた run の identity、participant sequence、completion time を返してよい。
3. endpoint は credential-free、GET-only のままとする。public participant sequence 以外の private/internal input、bot ID、artifact reference、locator、credential、record、snapshot、history、run selection control は出さない。

## Work

1. public spectator specification と TypeSpec に participant sequence の保持を追記する。既存 `PublicParticipant` の wire field、game 固有の意味付け、他 game の契約は変更しない。
2. distinct first/second participant を持つ completed selected run で anonymous list/detail/state を検証する。順序・個数を変更せず projection し、完全でない historical provenance は従来どおり metadata を省略することを確認する。
3. participant order、`display_name`、`ai_submission_id`、immutable `completed_at`、promotion による selected-run switch、CORS、private field 非露出を検証する。
4. scoped Go test、TypeSpec generation/check、repository quality gate を実行する。完了した plan は後続の execution branch で PR 準備後に削除する。

## Dependencies and parallelism

この plan は PR #364 の public metadata delivery が merge 済みであることに依存する。documentation/TypeSpec の明確化と route test の作成は fixture shape の合意後に並行でき、generated artifact の review は TypeSpec 変更後に行う。Reversi visualizer を含む consumer は既存の互換配列契約を利用できるが、順序の game 固有の意味付けと不完全な metadata の扱いは各 consumer の責務とする。

## Verification

- TypeSpec/public artifact checks preserve the existing `participants[]` field names and do not add a private or operator route.
- Public HTTP tests prove list/detail/state return the selected completed run with its submitted participant sequence, an immutable completion time, and no private fields.
- Existing generic participant-order, legacy-incomplete-provenance, selected-run, and anonymous CORS tests remain green.
- Run the repository's applicable Go, TypeSpec, and workflow quality gates before the implementation PR.

## Non-goals

- Changing `participants[]` into role-specific objects or renaming `display_name`.
- Assigning player roles, colours, counts, or positional semantics for any game.
- Exposing run IDs as controls, historical attempts, operator metadata, or private replay inputs.
