# ゲーム registry 仕様

## 目的

このドキュメントは、AI Arena が複数 game を registered game として扱うための
registry contract を定義する。Phase 3 では plugin 機構までは入れず、repo 内にある game を
共通の descriptor 入口で lookup / build / replay できることを正本化する。

## この spec の責務範囲

この spec が定義するもの:

- registry key
- `GameDescriptor` の最小責務
- fresh run / snapshot resume / history replay の build 入口
- game master 接続形態の表現
- lookup 後に行う metadata 確定と compatibility 検証の責務分離

この spec が定義しないもの:

- game 固有 payload schema
- game master subprocess / trusted external backend transport の詳細
- game plugin / self-service registration の運用

## 参照関係

- `docs/specs/platform-common-contract.md`: metadata / action status / record core schema の正本
- `docs/specs/platform.md`: runner / replay-debug entrypoint と artifact 契約
- `docs/specs/janken-game.md`: `janken` 固有 payload

## registry key

registry の lookup key は `game_id + game_version major` の組とする。

- `game_id` はゲーム系列の識別子
- `game_version major` は semver major を取り出した互換境界

例:

- `janken` + `2`
- `echo-count` + `2`

次は別 registered game とみなす。

- `game_id` が異なる
- `game_version` の major が異なる

`ruleset_version` は registry key に含めない。lookup 後に descriptor の build 入口へ渡し、
各 game が supported ruleset かどうかを検証する。

## registered game の最小要件

platform に registered game を追加するには、少なくとも以下を持つ descriptor を登録する。

- `game_id`
- `game_version major`
- fresh run 用 build 入口
- snapshot resume 用 build 入口
- history replay から snapshot を組み立てる入口
- game master 接続形態を表す動作モード

descriptor は constructor 群の寄せ集めではなく、1 game 系列の起動・再開・replay に必要な入口を
まとめた `GameDescriptor` 相当として扱う。

## `GameDescriptor` 契約

`GameDescriptor` は少なくとも以下を持つ。

- `RegistryKey`
  - `game_id`
  - `game_version major`
- `GameID`
- `BuildMode`
- session build
- history replay から snapshot を構築する build

### `BuildMode`

game program との接続形態は capability flag ではなく descriptor の動作モードとして保持する。

最低でも次を表現できること:

- `in-process`
- `local-subprocess`
- `future-external-adapter`

1 つの registered game は 1 つの `BuildMode` を持つ。
Phase 3 時点では、既存 repo 内 game master は `in-process` だけで登録してよい。
fixture 検証のために別接続形態も試したい場合は、`echo-count` と `echo-count-subprocess` のように
別 `game_id` の registered game として分ける。

## `BuildSpec` 契約

descriptor の build 入口へ渡す最小入力は `BuildSpec` とし、少なくとも以下を含む。

- `GameVersion`
- `Ruleset`
- `Players`

session build は、fresh run ではこの `BuildSpec` を受け取り、
snapshot resume ではこれに加えて snapshot を受け取る。history replay build は event 列と
target turn を受け取る。

## build と compatibility の責務分離

1. runner / replay-debug entrypoint は `game_id + game_version major` で registry lookup する
2. lookup で取得した descriptor に `BuildSpec` を渡す
3. 各 game の build 入口が `ruleset_version` 妥当性を検証し、登録済み `BuildMode` で起動可能な game master session または snapshot を返す
4. platform は build 後に確定した metadata を AI sidecar manifest や record metadata と照合する

このため:

- registry lookup 自体は `ruleset_version` を key に使わない
- `ruleset_version` 不一致は build 後 metadata の compatibility error として扱う
- history replay の可否は capability set ではなく、descriptor が replay build 入口を持つことで表す

## snapshot / history replay 入口

descriptor は replay/debug のために以下を提供する。

- `BuildSession(BuildSpec) -> game master session`
- `BuildSessionFromSnapshot(BuildSpec, snapshot) -> game master session`
- `SnapshotFromHistory(BuildSpec, events, target_turn) -> game.Snapshot`

これにより replay/debug は registry 外へ game 固有 helper を漏らさずに扱う。

## 採用しない案

- constructor registry
  - replay/debug の game 固有入口が registry 外へ漏れやすいため採用しない
- `game_id` 単独 key
  - semver major が互換境界という project 方針を lookup key に反映できないため採用しない
- capability set 中心設計
  - 現時点で必要なのは build/replay 入口と接続形態であり、flag を先に増やしても意味が薄いため採用しない
