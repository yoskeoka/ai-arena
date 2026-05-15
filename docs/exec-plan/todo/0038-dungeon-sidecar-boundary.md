# dungeon-sidecar-boundary
**Execution**: Use `/execute-task` to implement this plan.

Addresses:

- `docs/issues/0013-dungeon-sidecars-should-not-depend-on-internal-platform-protocol.md`

## Objective

`cmd/dungeon-gamemaster` を「dungeon game 側が将来別 repo へ持ち出せる sidecar entrypoint」として
成立させつつ、platform が game master と通信・接続するための内部実装は引き続き `internal` に残す。
この plan では、portable にすべき共有資産を sidecar SDK 候補へ寄せ、platform の runtime/session/registry
実装とは明確に分離するだけでなく、将来 `dungeon-gamemaster` 系ファイルを新 repo へ物理移動し、
import path を置換することが主作業になる状態へ近づける。

関連 issue:

- `docs/issues/0013-dungeon-sidecars-should-not-depend-on-internal-platform-protocol.md`

depends on:

- `docs/exec-plan/done/platform-phase5-01-dungeon-fixed-map-mvp.md`
- `docs/exec-plan/done/platform-phase3-03-game-master-runtime-boundary.md`

## Context

- `docs/project-plan.md` の Phase 5 は、dungeon 固有コードを `internal` package に依存させず、
  将来別 repo へ移せる境界を保つ方針を明示している
- `docs/specs/game-master.md` は game master の論理 API と local subprocess transport mapping を定義しているが、
  current implementation ではその payload 型と subprocess loop 実装が `internal/platform/*` に残っている
- `cmd/dungeon-bot-local` と `cmd/dungeon-map-helper` はすでに `games/dungeon/...` だけへ依存している一方、
  `cmd/dungeon-gamemaster` は `catalog` / `game` / `gamemaster` / `protocol` / `session` に依存している
- 今回直したいのは「platform の内部コードを広く public に出すこと」ではなく、
  game master sidecar 実装者が共有できる API/DTO と、platform 内部の adapter 実装を分離することである
- ただし今回の目標は同一 repo 内で境界だけを説明して終わることではなく、
  新 repo 側へ持っていくファイル群が platform internal 実装から独立して存在できるように package 配置と import 関係も整えることである

## Scope

- dungeon game master sidecar が依存してよい公開境界を定義する
- local subprocess game master transport の SDK 候補 API/DTO を `internal` 外へ切り出す
- platform 内部 adapter がその SDK 候補 API/DTO と `internal/platform/*` 実装の橋渡しを行う構造へ寄せる
- `cmd/dungeon-gamemaster` を公開境界だけへ依存する構成へ移す
- `cmd/dungeon-gamemaster` とその周辺 package 群が、新 repo へ移す際に
  「ファイル移動 + import path 置換 + 必要最小限の module/CI 配線」で成立しやすい配置へ寄せる
- 関連 spec を、SDK 候補と platform internal implementation の責務分離に合わせて更新する

この plan では以下は扱わない。

- `catalog` / `runtime` / `session` / `registry` の public 化
- trusted external game backend の実ネットワーク adapter 実装
- 実際の `dungeon-gamemaster` 新 repo 作成、履歴移管、CI/bootstrap 整備
- `dungeon-map-helper` の機能追加や dungeon ルール変更
- issue の execution 完了前に `docs/issues/0013-dungeon-sidecars-should-not-depend-on-internal-platform-protocol.md` を閉じること

## Design Decision

この plan では追加 ADR は作らない。

- public に出すのは「game master sidecar を実装するための API/DTO/transport helper」に限定する
- platform が game master subprocess を起動・監視・互換性判定する責務は `internal/platform/*` に残す
- `cmd/dungeon-gamemaster` は public sidecar API を実装する thin entrypoint とし、dungeon domain 以外の判断は持ち込まない
- platform 内部では adapter 層を置き、public sidecar API と `internal/platform/game` / `internal/platform/gamemaster` /
  `internal/platform/catalog` の間を変換する
- portable 側 package は、新 repo へ移したときに platform repo の `internal` package や
  ai-arena repo root 前提 helper を追加で必要としない配置を優先する

却下する案:

- `internal/platform/*` の既存型をそのまま public へ昇格する
  - sidecar SDK 候補と platform 内部実装の境界が消え、将来も internal 実装都合が外部契約へ漏れるため採らない
- dungeon 専用 package のみに閉じた DTO を作る
  - dungeon issue の対処には見えても、game master sidecar という共通面の分離ができず、SDK 候補の境界として弱いため採らない
- 最初から別 repo を立てて、その repo 境界を前提に一気に配線し直す
  - 最終形の証明はしやすいが、module/CI/bootstrap まで同時に抱えて今回の境界整理そのものが見えにくくなるため、この plan では採らない

## Spec Changes

### `docs/specs/game-master.md`

- local subprocess game master contract について、次を明記する
  - game master sidecar 実装者が依存してよいのは public sidecar API/DTO である
  - platform の runtime/session/registry/catalog は internal implementation detail である
  - transport mapping の method 名と payload 契約は SDK 候補として安定させるが、platform 内部 symbol 名とは切り離す
  - 将来 sidecar 実装を別 repo へ移す際も、この public 契約が import path 以外はそのまま使えることを狙う

### `docs/specs/platform-common-contract.md`

- game master / player / platform が共有する core schema のうち、
  sidecar SDK 候補へ載せる DTO 範囲と platform internal-only responsibility を整理する

### `docs/specs/platform.md`

- dungeon の portable boundary 説明を補強し、
  game master sidecar は public SDK 候補に依存してよいが platform adapter 実装は `internal` に残す方針を補足する

### `docs/specs/dungeon-game.md`

- dungeon sidecar entrypoint が dungeon domain と public sidecar API のみへ依存する前提を補足する
- dungeon sidecar 実装を新 repo へ移すときに ai-arena 側へ残すべきものと一緒に持っていけるものの境界を補足する

## Expected Code Changes

### public sidecar API package

- game master local subprocess contract 用の non-internal package を追加する
- 少なくとも以下をその package へ移す、または internal alias ではなく正本として置く
  - metadata DTO
  - decision mode / action status / failure reason / snapshot 系 core DTO
  - `initialize_match` / `next_decision_step` / `normalize_action` / `apply_decision_results` /
    `current_snapshot` / `current_exported_snapshot` / `current_result` / `shutdown`
    に対応する request/response payload 型
  - stdio JSON-RPC request loop を書きやすくする最小 helper
- package の import 先は、新 repo 側へ移したときに ai-arena の `internal` package や
  platform orchestration package へ戻らなくてよい形を優先する

### platform internal adapter boundary

- `internal/platform/gamemaster` などの内部実装は、
  public sidecar API package を使って subprocess transport を扱う形へ寄せる
- `catalog` / `runtime` / `session` / `registry` は internal のまま維持する
- 必要なら `internal/platform/game` が持つ alias 群を整理し、
  platform core が internal 側の責務語彙で読める一方で、sidecar 実装は internal 型を直接 import しない状態にする
- platform 側に残る code は「portable 側を import して使う consumer」として整理し、
  portable 側から platform 側を逆参照しない片方向依存に揃える

### `cmd/dungeon-gamemaster`

- `internal/platform/*` 直接 import をやめる
- dungeon domain と public sidecar API package だけを使って request loop / payload decode / response encode を実装する
- package comment を今回の境界に合わせて更新する
- 可能なら sidecar entrypoint の近傍 helper も同じ portable package 群へ寄せ、
  新 repo 化するときに command 本体と一緒に移せる範囲を広げる

### verification and tests

- `cmd/dungeon-gamemaster` の unit test を public sidecar API 前提へ更新する
- platform 側 adapter test で、public sidecar API 経由でも既存の metadata compatibility / snapshot / result 契約が維持されることを確認する
- portable 側 package 群について、platform repo 外へ移したと仮定しても import graph 上
  `internal/platform/*` へ戻らないことを確認する

## Sub-tasks

- [ ] game master sidecar SDK 候補として公開する DTO / helper 範囲を spec に固定する
- [ ] public sidecar API package を追加する
- [ ] `internal/platform/gamemaster` の subprocess adapter を public sidecar API 経由へ寄せる
- [ ] `cmd/dungeon-gamemaster` の internal 依存を外す
- [ ] `cmd/dungeon-gamemaster` と一緒に新 repo へ持っていく候補 package 群を整理し、片方向依存へ揃える
- [ ] dungeon spec と platform spec の portable boundary wording を更新する
- [ ] execution 完了時に `docs/issues/0013-dungeon-sidecars-should-not-depend-on-internal-platform-protocol.md` を `docs/issues/done/` へ移す

## Parallelism

- [parallel] spec wording 更新と public DTO/package の切り出しは、公開する責務範囲が固まれば並行で進められる
- [parallel] `cmd/dungeon-gamemaster` の entrypoint 整理と platform adapter 側の変換整理は、public sidecar API package の shape が固まれば並行で進められる
- [parallel] portable 側 package の import graph 整理と platform adapter 側の consumer 化は、public sidecar API package の shape が固まれば並行で進められる
- final verification は adapter 更新と command 側更新の両方が揃ってから行う

## Risks and Mitigations

- public sidecar API が platform 内部都合を引きずり、結局 SDK ではなく internal mirror になる
  - mitigation: sidecar 実装に本当に必要な DTO / helper だけを公開し、runtime/session/registry/caching などは internal に残す
- dungeon issue の解消だけを見て API が dungeon 専用 shape へ寄りすぎる
  - mitigation: `docs/specs/game-master.md` の共通 contract を正本にし、dungeon 固有 payload は game spec 側へ閉じ込める
- alias の整理が中途半端だと platform core と sidecar SDK の責務境界がコード上で読みにくい
  - mitigation: internal package 側は adapter / orchestration 語彙、public package 側は transport contract 語彙で役割を分ける
- command 側だけ public package へ寄せても platform adapter が旧 internal contract に残ると二重定義が発生する
  - mitigation: execution では command 側と internal adapter 側を同じ public DTO 正本へ揃える
- portable package が ai-arena repo 固有の helper や前提に暗黙依存すると、新 repo 化で追加剥離作業が残る
  - mitigation: file move と import path 置換以外に何が残るかを execution 中に洗い出し、portable 側へ押し戻せるものは先に押し戻す

## Verification

The execution PR is complete when the following are true.

- `cmd/dungeon-gamemaster` が `internal/platform/*` を import しない
- public sidecar API package だけで local subprocess game master entrypoint を実装できる
- `internal/platform/gamemaster` など platform 側は引き続き `internal` に閉じたまま、public sidecar API 経由で game master subprocess と通信できる
- `cmd/dungeon-gamemaster` とその近傍 portable package 群を別 repo へ移す想定で見たとき、
  残作業が主に「ファイル移動」「module/import path 置換」「repo bootstrap の最小配線」で説明できる
- `docs/specs/game-master.md`, `docs/specs/platform-common-contract.md`, `docs/specs/platform.md`, `docs/specs/dungeon-game.md` が最終境界と一致する
- execution 完了時に `docs/issues/0013-dungeon-sidecars-should-not-depend-on-internal-platform-protocol.md` が `docs/issues/done/` へ移動している
