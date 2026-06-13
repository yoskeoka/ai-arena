# Platform Service General Submission 仕様

## 目的

このドキュメントは、Phase 7 の general operator lane で扱う
`game registration` と `AI submission` の最小 entity / validation contract を定義する。

この spec が固定するのは、1 試合要求そのものではなく、
後続の match request / scheduling が参照する durable identity と validation surface である。

## この spec の責務範囲

この spec が定義するもの:

- `game registration` の最小 identity と metadata
- `AI submission` の最小 identity と validation 結果
- operator-facing registration route が同期的に確認する内容
- preset queue lane から general lane へ接続する変換点

この spec が定義しないもの:

- match request / scheduling policy
- ranking 集計
- rerun / retry / cancellation
- public self-service upload portal

## 参照関係

- `docs/specs/platform-game-registry.md`: registered game lookup の正本
- `docs/specs/platform-service-skeleton.md`: single-match `match submission` の正本
- `docs/specs/platform-service-operator-api.md`: operator-facing HTTP route の正本

## エンティティ

### Game Registration

`game registration` は、operator lane で選択可能な registered game の plain-data view である。

最小項目:

- `registration_id`
- `game`
  - `game_id`
  - `game_version`
  - `ruleset_version`
- `build_mode`
- `builder_id`
- `supported_rulesets`
- `source`
  - `manual`
  - `preset`
- `source_id`
  - preset 由来なら `preset_id`

`registration_id` は stable identity であり、default では
`game_id + game_version major` から決まる deterministic id にしてよい。

### AI Submission

`AI submission` は、1 つの registered game に対して admission 済みの AI artifact identity を表す。

最小項目:

- `ai_submission_id`
- `game_registration_id`
- `game`
  - `game_id`
  - `game_version`
  - `ruleset_version`
- `artifact_ref`
- `display_name`
- `runtime_kind`
- `ai_id`
- `validation_state`
  - initial contract では `ready` のみ
- `source`
  - `manual`
  - `preset`
- `source_id`
  - preset 由来なら `preset_id`

initial contract では synchronous validation に成功した record だけを exposed してよい。
非同期 review queue や `pending` 状態は後続へ送る。

## Validation

### Game Registration Validation

`game registration` を受け付けるときは、少なくとも次を同期的に確認しなければならない。

- `game_id + game_version major` が registry lookup 可能であること
- `ruleset_version` が lookup した descriptor の `supported_rulesets` に含まれること
- `build_mode` / `builder_id` / `supported_rulesets` が descriptor 由来 metadata と矛盾しないこと

validation 後に保存する metadata view は、lookup 結果の plain-data projection とする。

### AI Submission Validation

`AI submission` を受け付けるときは、少なくとも次を同期的に確認しなければならない。

- 参照先 `game_registration_id` が存在すること
- `artifact_ref` がその lane で解決可能であること
- sidecar manifest または fallback runtime から得られる metadata が
  registration の `game` と互換であること
- runtime entrypoint が最小 startability を満たすこと

validation 成功後は `ai_id` と `runtime_kind` を registration record と一緒に exposed してよい。

## Preset Queue との関係

preset queue lane は Phase 6 confirmation 専用 lane であり、
general operator lane の canonical entity ではない。

ただし preset lane は general lane と完全に分断してはならない。
preset から queue へ積むとき、service は少なくとも次の変換点を持ってよい。

1. preset definition から `game registration` identity を materialize する
2. preset participant ごとに `AI submission` identity を materialize する
3. queue へ積む 1 試合要求は、materialized entity と同じ `game` / `artifact_ref` を参照する

この変換により、preset lane は first remote landing の bootstrap に留めつつ、
後続の match request / scheduling lane が参照する stable identity を先に固定できる。

## Deferred Follow-Ups

- DB-backed durability
- AI upload binary storage
- retired / superseded / suspended lifecycle
- operator auth と ownership scope
