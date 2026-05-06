# platform-phase3-02-game-registry
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`arena-runner` と replay/debug 経路に残っている `switch` ベースの game 選択を、複数 game を登録・選択できる game registry に置き換える。Phase 3 以降の dungeon game 追加や別 repo 由来の game master 接続を見据え、platform が game を識別・起動・互換性確認する共通入口を固定する。

親 plan:

- `platform-phase3-common-interface-contract.md` (`docs/exec-plan/todo/` または `docs/exec-plan/done/`)

depends on:

- `platform-phase3-01-common-contract-surface.md`

## Scope

- registered game の registry abstraction 導入
- `arena-runner` / replay/debug の game 選択経路の registry 化
- game metadata と ruleset 選択の入口統一
- game program の接続形態を descriptor の動作モードとして扱える入口の固定

この plan では以下は扱わない。

- game master subprocess / external backend transport
- game plugin/self-service registration
- WASM runtime 統合

## Spec Changes

### `docs/specs/platform.md`

- runner は `game_id` と `ruleset_version` から対象 game を選ぶことを、registry 経由の contract として明記する
- runner の責務と game registry の責務を分離して書く
- compatibility 判定のタイミングを、game registry lookup 後の game metadata 確定フェーズとして整理する

### New registry spec or section

- platform が保持する registered game の最小要件を定義する
- 少なくとも以下を含める
  - `game_id`
  - `game_version major`
  - `ruleset_version`
  - fresh run 用 build 入口
  - snapshot resume 用 build 入口
  - replay/debug に必要な snapshot/history 復元能力
- registry の登録単位は constructor 群ではなく `GameDescriptor` 相当の descriptor として固定する
- registry key は `game_id + game_version major` の組とする
  - major version が異なれば別 game とみなす project 方針に合わせる
  - `ruleset_version` は registry key には入れず、descriptor 配下の build 時検証で扱う
- game program との接続形態は capability ではなく descriptor の動作モードとして持つ
  - 例: in-process, local-subprocess, future-external-adapter
- compatibility 判定は registry lookup 自体ではなく、lookup 後に `catalog` が担う metadata 検証で行う
- ruleset 妥当性判定は各 game の build 入口が担う

## Expected Code Changes

- `cmd/arena-runner/main.go`
  - `newMaster*` の `switch` と早期 `unsupported game` 判定を registry lookup に置き換える
- 新規 `internal/platform/registry/`
  - `GameDescriptor`
  - `BuildSpec`
  - game registration API
  - runner / replay から共有される game lookup
- `internal/platform/replay/`
  - game 固有分岐を registry 経由に置き換える
- `internal/games/echo/`, `internal/games/janken/`
  - registry 登録点を追加する

`GameDescriptor` の最小責務は以下で固定する。

- `RegistryKey` (`game_id + game_version major`)
- `GameID`
- fresh run 用 build
- snapshot resume 用 build
- history replay から snapshot を組み立てる入口
- game master 接続形態を表す動作モード

`BuildSpec` の最小項目は以下で固定する。

- `GameVersion`
- `Ruleset`
- `Players`

この plan では capability set は導入しない。
理由:

- 現時点で必要な差分は descriptor が持つ build/replay 入口と接続形態で十分である
- `CanReplayHistory` や `CanResumeSnapshot` のような capability flag を先に増やしても、現状の code path では descriptor API の具体責務以上の意味を持ちにくい
- 接続形態は「できる/できない」ではなく、platform がどのモードで game program を扱うかの選択情報として持つ方が自然である
- constructor registry は採用しない
  - 理由: replay/debug の入口が registry 外へ漏れやすい
- registry key を `game_id` のみにする案は採用しない
  - 理由: major version が異なれば別 game とみなす前提を key に反映した方が、lookup 時点で意図が明確になる

## Verification

- `go test ./...` が通る
- runner から `echo-count` / `janken` を registry 経由で起動できる
- replay/debug でも registry 経由で対象 game を復元できる
- 不明な `game_id` / 非対応 `ruleset_version` が一貫した error になる
- descriptor の動作モードが runner 側で解釈可能な形になっている

## Sub-tasks

- [ ] Define the `GameDescriptor`-based registered game abstraction in spec
- [ ] Define the registry key as `game_id + game_version major`
- [ ] Define descriptor-level game master connection modes
- [ ] Implement a shared game registry package or module
- [ ] [parallel] Convert runner game selection to registry lookup
- [ ] [parallel] Convert replay/debug game selection to registry lookup
- [ ] Register existing `echo-count` and `janken` implementations
- [ ] Add tests for unknown game / incompatible ruleset / successful lookup

## Parallelism

- runner と replay の切り替えは registry API 固定後に並行で進められる
- existing game registration は registry skeleton 作成後に独立して進められる

## Risks and Mitigations

- registry が constructor 集約だけに留まり、後続の game master 標準化につながらない
  - mitigation: registration payload を `GameDescriptor` にし、接続形態を descriptor の動作モードとして持たせる
- replay/debug が game 固有 helper に強く結合している
  - mitigation: まず lookup 入口を統一し、内部 helper の抽象化は必要最小限に留める

## Design Decisions

- plugin 機構までは入れず、まず repo 内 registered game の共通入口を固定する
- registry の登録単位は constructor 群ではなく `GameDescriptor` とする
- `GameDescriptor` は fresh run / snapshot resume / history replay の 3 入口を持つ
- registry key は `game_id + game_version major` とする
- game master との接続形態は capability ではなく descriptor の動作モードとして保持する
- registry は後続の game master subprocess / external backend adapter の受け皿になる形にする
