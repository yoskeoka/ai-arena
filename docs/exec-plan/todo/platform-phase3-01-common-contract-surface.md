# platform-phase3-01-common-contract-surface
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Phase 2 で `platform.md` と実装各所に分散している platform 共通契約を、Phase 3 の基礎として整理し直す。特に AI player / platform / game master の間で共有される語彙、payload 境界、failure 分類、record 上の共通表現を固定し、後続の game registry 化と game master 標準化の前提を揃える。

親 plan:

- `platform-phase3-common-interface-contract.md` (`docs/exec-plan/todo/` または `docs/exec-plan/done/`)

## Scope

- platform 共通契約の spec 切り出しと再整理
- failure 分類の用語統一と code/spec 対応
- action status / metadata / record core schema の整理
- AI player 向け共通契約と game master 向け共通契約の境界整理

この plan では以下は扱わない。

- game 選択機構の registry 化
- game master を別プログラムとして起動する実行経路
- trusted external game backend 接続の実装
- 個別 game の詳細ルール追加

## Spec Changes

### `docs/specs/platform.md`

- Phase 2 で定義した platform 共通面と fixture appendix を整理し、platform core の責務と game 固有仕様の境界を明確化する
- `accepted` / `no_action`、failure 分類、late response、runtime stop の扱いを、実装済み contract に合わせて明確化する
- `init` / `turn` / `game_over` の共通メソッド契約と、game 固有 payload を載せる場所の責務分離を明記する
- `game_id` / `game_version` / `ruleset_version` の責務と compatibility ルールを再整理する
- match record / event log / snapshot / exported snapshot のうち game 非依存コア部分を再整理する

### New common-contract spec

- AI player / platform / game master が共有する platform 共通契約を、個別 game spec から独立した spec として追加する
- この spec には少なくとも以下を含める
  - metadata 契約
  - request / response / notification の共通 envelope
  - decision step と action status の意味
  - failure 分類
  - record core schema
- spec filename は `docs/specs/platform-common-contract.md` とする
- 章立ては以下で固定する
  - 目的
  - この spec の責務範囲
  - 参照関係
  - 共通 metadata 契約
  - 共通 transport 前提
  - 共通メソッド契約
  - Decision Step 契約
  - Action Status 契約
  - Failure 分類
  - Record Core Schema
  - Snapshot / Exported Snapshot の責務境界
  - platform と game 固有仕様の分担
- `platform-common-contract.md` は共通語彙と責務境界の正本とし、以下は含めない
  - JSON-RPC / NDJSON の細かな framing 実装詳細
  - 個別 game の payload schema
  - game master 開発仕様の詳細
  - registry 実装詳細
- `turn_mode` はこの plan で共通 metadata から外す方向で整理する
  - Phase 2 では compatibility metadata として存在したが、Phase 3 では `DecisionStep.requests` による実行形態表現へ寄せる
  - simultaneous / sequential は将来の game 分類タグや説明項目として扱う余地を残すが、platform 共通 contract の必須 compatibility key にはしない

### `docs/specs/dungeon-game.md`

- Phase 3 で dungeon game が依存する platform 共通契約への参照関係を明記する
- dungeon game 側で後続定義するのは game 固有 payload と validation であることを明記する

## Expected Code Changes

- `internal/platform/catalog/`
  - metadata と compatibility 判定の責務を整理する
- 新規 `internal/platform/contract/`
  - platform 共通語彙の正本を置く
- `internal/platform/session/`
  - failure 分類、status、共通 request lifecycle の表現を見直す
- `internal/platform/game/`
  - decision step / action status / snapshot core の責務を整理する
- `internal/platform/match/`
  - `contract` へ寄せた共通型の参照先を切り替える
- 既存 test を failure 分類・共通 contract の観点で整理する

`internal/platform/contract/` の収容型は以下で固定する。

- `types.go`
  - `GameMetadata`
  - `DecisionMode`
  - `MatchStatus`
  - `ActionDecision` または同等の action status kind
  - `FailureReason`
  - `ActionStatus`
- `payloads.go`
  - `InitParams`
  - `TurnParams`
  - `GameOverParams`
- `snapshots.go`
  - `Placement`
  - `MatchResult`
  - `PlayerSnapshot`
  - `Snapshot`
  - `ExportedPlayerSnapshot`
  - `ExportedSnapshot`

以下はこの plan では `contract` に置かない。

- `protocol` package が持つ JSON-RPC envelope 型
- `match.Record`
- `match.Event`
- `game.Master`
- registry abstraction
- replay helper
- game 固有 state 型

package 分割は execution で調整してよいが、以下は維持する。

- player protocol と game logic を分離する
- game 固有 validation failure と platform failure を分離する
- 共通契約は `echo-count` や `janken` の appendix に埋もれず、後続 game から参照可能にする

## Verification

- `go test ./...` が通る
- failure 分類の用語と実装値が spec と一致する
- `accepted` / `no_action` と failure reason の組み合わせ制約が test で確認できる
- metadata compatibility 判定が spec 通りである
- `platform.md` と新しい共通契約 spec の責務分担がレビュー可能な形になる
- `contract` package の収容型と、既存 package の責務境界が review で追える

## Sub-tasks

- [ ] Reorganize platform common contract sections in `docs/specs/platform.md`
- [ ] Add a dedicated spec for platform common interface contract
- [ ] Normalize failure 分類 terminology across specs and code
- [ ] Fix the chapter structure of `docs/specs/platform-common-contract.md`
- [ ] Define the concrete type inventory for `internal/platform/contract/`
- [ ] [parallel] Audit `catalog`, `session`, and `game` packages for duplicated contract definitions
- [ ] [depends on: Audit `catalog`, `session`, and `game` packages for duplicated contract definitions] Consolidate shared contract types/constants
- [ ] [depends on: Define the concrete type inventory for `internal/platform/contract/`, Audit `catalog`, `session`, and `game` packages for duplicated contract definitions] Migrate shared types into `internal/platform/contract/` in two steps
- [ ] Update unit tests to assert the normalized contract surface

## Parallelism

- spec 再整理と code audit は並行で進められる
- 共通 type の寄せ先決定後に code/test 更新を進める

## Risks and Mitigations

- `platform.md` の責務を削り過ぎると fixture 仕様の参照性が落ちる
  - mitigation: core contract と fixture appendix の導線を明記する
- failure 分類の命名変更が広がる
  - mitigation: spec で先に最終形を固定し、code は最小移行で揃える

## Design Decisions

- Phase 2 実装済みの transport / session / match contract を土台とし、再発明はしない
- 今後の docs/plan では「failure 分類」で統一する
- `platform-common-contract.md` は transport framing の細部ではなく、共通語彙と責務境界の正本に絞る
- `internal/platform/contract` は shared vocabulary に限定し、`Record` / `Event` / `Master` / registry までは抱え込まない
- 既存コードの移行は 2 段階で進める
  - まず `contract` package を追加し、既存 package から参照する
  - その後、重複した旧型や string literal を整理する
- `turn_mode` は互換性維持対象ではなく、この phase で共通 metadata から削除対象として扱う
