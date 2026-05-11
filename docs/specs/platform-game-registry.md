# ゲーム registry 仕様

## 目的

このドキュメントは、AI Arena が複数 game を registered game として扱うための
registry contract を定義する。Phase 3 では plugin 機構までは入れず、repo 内にある game を
共通の descriptor 入口で lookup / build / replay できることを正本化する。

## この spec の責務範囲

この spec が定義するもの:

- registry key
- 永続化できる登録情報と process-local な起動情報の境界
- 永続化レコード参照、runtime 解決、lookup 公開の責務分離
- registered game が備える起動・再開・replay 入口の最小契約
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

`ruleset_version` は registry key に含めない。lookup 後に registered game の build 入口へ渡し、
各 game が supported ruleset かどうかを検証する。

## persisted descriptor record

永続化 backend に保存する registered game metadata は、runtime の function を含まない
plain data として扱う。この spec では、その保存単位を安定した抽象概念として
`DescriptorRecord` と呼ぶ。

`DescriptorRecord` は少なくとも以下を持つ。

- `game_id`
- `game_version_major`
- `build_mode`
- `builder_id`
- ruleset / build 制約を表す metadata

canonical な一意キーは `game_id + game_version_major` の composite key とする。
必要なら `registry_key` という論理名を持ってよいが、これは composite key の derived /
denormalized field として扱う。

Phase 3 の in-memory 実装でも、hard-coded registration は最終的に保存可能な record として
store へ渡す。将来 DB-backed store へ差し替える場合も、保存対象はこの shape までとする。

## registry lookup 層

runner / replay から見える registry lookup は、少なくとも以下 3 責務に分ける。

- 永続化された registered game record を key で読む責務
- 読み出した record を process-local な起動情報へ解決する責務
- その 2 つを束ねて runner / replay へ lookup を提供する責務

実装上は `RegistryStore`、`DescriptorResolver`、`Registry` などの名前を使ってよいが、
spec として重要なのは責務分離そのものである。この分割により、永続化 backend は record 読み出し層だけ
差し替えればよく、build/replay function の解決は process 内に閉じ込められる。

## registered game の最小要件

platform に registered game を追加するには、少なくとも以下を持つ登録情報と runtime 入口を用意する。

- `game_id`
- `game_version major`
- `builder_id`
- `build_mode`
- fresh run 用 build 入口
- snapshot resume 用 build 入口
- history replay から snapshot を組み立てる入口
- ruleset / build 制約 metadata

persisted `DescriptorRecord` は DB 保存可能な plain data に限定する。
runtime の function を持つ起動情報は constructor 群の寄せ集めではなく、
1 game 系列の起動・再開・replay に必要な入口をまとめた process-local object として扱う。

## runtime descriptor 契約

lookup 後に runner / replay が受け取る process-local な起動情報は、少なくとも以下を持つ。
この spec では、その起動情報を安定した抽象概念として `GameDescriptor` と呼ぶ。

- lookup 済みの game 系列識別子
- lookup 済みの `game_version major`
- `BuildMode`
- ruleset / build 制約 metadata
- session build
- history replay から snapshot を構築する build

`GameDescriptor` は store に保存する shape ではない。保存済み record と process 内の builder registration
を突き合わせて構築する runtime object であり、fresh run / snapshot resume / history replay の
build 入口を持つ。

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

## build input 契約

registered game の build 入口へ渡す最小入力は、少なくとも以下を含む。
この spec では、その入力単位を安定した抽象概念として `BuildSpec` と呼ぶ。

- target `game_version`
- target `ruleset_version`
- players

session build は、fresh run ではこの build input を受け取り、
snapshot resume ではこれに加えて snapshot を受け取る。history replay build は event 列と
target turn を受け取る。

## build と compatibility の責務分離

1. runner / replay-debug entrypoint は `game_id + game_version major` を指定して registry lookup を要求する
2. registry は対応する persisted record を選び、runtime descriptor へ解決する
3. 解決した descriptor に build input を渡す
4. 各 game の build 入口が `ruleset_version` 妥当性を検証し、登録済み `BuildMode` で起動可能な game master session または snapshot を返す
5. platform は build 後に確定した metadata を AI sidecar manifest や record metadata と照合する

このため:

- registry lookup 自体は `ruleset_version` を key に使わない
- persisted record lookup と runtime descriptor 解決は分離する
- `ruleset_version` 不一致は build 後 metadata の compatibility error として扱う
- history replay の可否は capability set ではなく、descriptor が replay build 入口を持つことで表す

## snapshot / history replay 入口

descriptor は replay/debug のために以下を提供する。

- fresh run を game master session に変換する入口
- resume snapshot を game master session に変換する入口
- history と target turn から snapshot を再構築する入口

これにより replay/debug は registry 外へ game 固有 helper を漏らさずに扱う。

## 採用しない案

- constructor registry
  - replay/debug の game 固有入口が registry 外へ漏れやすいため採用しない
- `game_id` 単独 key
  - semver major が互換境界という project 方針を lookup key に反映できないため採用しない
- capability set 中心設計
  - 現時点で必要なのは build/replay 入口と接続形態であり、flag を先に増やしても意味が薄いため採用しない
