# platform-phase3-03-game-master-runtime-boundary
**Execution**: Use `/execute-task` to implement this plan.

## Objective

game master を platform 内の Go object に閉じず、platform から起動・接続される標準的な実行単位として扱える境界を定義する。local subprocess で動く game master と、将来の trusted external game backend 上で動く game master の両方を見据え、platform と game master のやりとりを標準化し、game master 開発者が platform に載せるために満たすべき仕様書を成立させる。

親 plan:

- `platform-phase3-common-interface-contract.md` (`docs/exec-plan/todo/` または `docs/exec-plan/done/`)

depends on:

- `platform-phase3-01-common-contract-surface.md`
- `platform-phase3-02-game-registry.md`

## Scope

- platform と game master の標準契約定義
- game master を別プログラムとして起動・接続する runtime boundary の導入
- local subprocess game master adapter の成立
- 将来の trusted external game backend adapter を見据えた interface 固定
- game master 開発仕様書の作成

この plan では以下は扱わない。

- trusted external game backend への実ネットワーク接続実装
- dungeon game 自体の詳細ルール実装
- online service / persistence backend の本実装

## Spec Changes

### `docs/specs/platform.md`

- platform と game master の責務境界を、in-process 前提ではなく runtime boundary 前提で書き直す
- player session と同様に game master session/runtime を持てることを明記する
- platform が game master に要求する metadata / lifecycle / state exchange / turn resolution 契約を定義する
  - platform は match loop、IO、timeout、artifact、record/persistence を主導する
  - game master は `DecisionStep.requests` により次に問い合わせる player 集合と解決順序を主導する
- trusted external game backend は将来 adapter 差し替えで載る想定であることを明記する
- `DecisionStep.requests` を game master が明示する request 対象 player 集合として扱うことを明記する
  - 複数 player を含む step は同時処理
  - 1 player だけを含む step は逐次処理
  - sequential game で自動 skip したい player は request に含めないことで表現できる
  - game master 実装者が、強制 pass でも毎 turn の public state 更新を露出したい場合は、強制 pass 用 request を送る実装も仕様上有効とする
- `DecisionMode` は runtime step contract に残し、`DecisionStep.requests` の fan-out 形と矛盾しないよう validation する
  - 例: `Sequential` なら request 対象は 1 player
  - `Simultaneous` なら request 対象は複数 player、または game master が同時処理として明示した step
- `turn_mode` はこの phase で compatibility metadata から外し、必要なら将来の game の簡易 description / 分類タグへ寄せる方針を整理する

### New `docs/specs/game-master.md`

- game master 開発者向け仕様書を新設する
- 少なくとも以下を定義する
  - game master metadata
  - 起動方式と sidecar/manifest の要否
  - platform との共通 transport
  - `init` 相当の開始ハンドシェイク
  - decision step の返し方
  - action normalization / validation / apply の責務
  - snapshot / exported snapshot / result の返し方
  - shutdown / error / audit event の扱い
- local subprocess で platform から起動される場合と、trusted external backend adapter 越しの場合で不変な contract を明記する
- turn progression の細部は game master が `DecisionStep.requests` の返し方で決める一方、platform は match loop と request execution を主導することを明記する
- 自動 skip / 強制 pass の扱いを定義する
  - default は request 非送信による skip
  - public state 更新や観測イベントの都合で明示的な強制 pass request を送る実装も許可する
- 最小論理 API 面を定義する
  - `InitializeMatch`
  - `NextDecisionStep`
  - `ApplyDecisionResults`
  - `CurrentSnapshot`
  - `CurrentExportedSnapshot`
  - `CurrentResult`
  - `Shutdown`

### `docs/specs/dungeon-game.md`

- dungeon game はこの game master 開発仕様を満たす別実装として成立する前提を追記する
- 将来別 repo で開発する場合でも platform から載せ替え可能なことを、Phase 3 依存事項として明記する

## Expected Code Changes

- `internal/platform/match/`
  - `game.Master` 直結前提を見直し、session-like な game master boundary を導入する
  - `DecisionStep.requests` を主入力として turn progression を解釈する
- `internal/platform/game/`
  - in-process 実装用 contract と runtime boundary 契約を整理する
- 新規 `internal/platform/gamemaster/` または同等 package
  - game master runtime/session interface
  - local subprocess adapter
  - metadata / handshake / state exchange types
  - 最小論理 API 面に対応する request/response DTO
- `cmd/arena-runner/main.go`
  - registry に登録された game master session を起動できるようにする
- existing games
  - `echo-count` は in-process game として残す
  - `echo-count-subprocess` を local subprocess game master fixture として追加し、e2e で境界を検証する
  - `janken` は既存経路を維持したまま新境界へ適応させ、local subprocess への全面移行はこの plan では求めない

## Verification

- `go test ./...` が通る
- platform が local subprocess として起動した game master と 1 match を完走できる
- `echo-count` と `echo-count-subprocess` がそれぞれ完走できる
- game master metadata / compatibility / lifecycle error が一貫して記録される
- game master 開発仕様書だけを読めば、platform に載せるための最低要件が分かる状態になる
- game master が request 対象 player を明示し、1 player 逐次 / 複数 player 同時 の両方を同じ論理 API で表現できる
- 自動 skip と強制 pass request の両方が仕様上説明可能になる
- 最小論理 API 面が spec と code boundary の両方で対応づく

## Sub-tasks

- [ ] Compare the candidate game master contract shapes and record the selected approach
- [ ] Define the platform <-> game master standard contract in spec
- [ ] Add a dedicated game master development spec
- [ ] Define turn progression semantics around `DecisionStep.requests`
- [ ] Define skip / forced-pass handling rules
- [ ] Define the minimal logical API surface for game master sessions
- [ ] Design a runtime/session abstraction for game master execution
- [ ] Implement a local subprocess game master adapter
- [ ] [parallel] Preserve an in-process adapter for existing games during migration
- [ ] [parallel] Keep `janken` on the existing path while adapting it to the new boundary
- [ ] Add fixture verification for both `echo-count` and `echo-count-subprocess`
- [ ] [parallel] Add a fixture or minimal sample game master executable for black-box verification
- [ ] Update runner/registry integration without adding generic game master mode selection
- [ ] Add verification for lifecycle, metadata compatibility, and state exchange

## Parallelism

- spec 策定と sample executable の準備は並行で進められる
- local subprocess adapter と existing in-process adapter 維持は、境界 fixed 後に並行で進められる

## Risks and Mitigations

- game master 側まで JSON/NDJSON 契約をそのまま持ち込むか、別 contract にするかで迷いやすい
  - mitigation: player と同じ transport を流用できる部分と、game master 特有の責務を分けて比較する
- sequential / simultaneous を metadata で固定し過ぎると、game master 側の柔軟な step 設計と衝突する
  - mitigation: 実際の turn progression は `DecisionStep.requests` で表し、`turn_mode` はこの phase で compatibility metadata から外す
- `DecisionMode` と `DecisionStep.requests` の両方を残すと矛盾状態が入り得る
  - mitigation: `DecisionMode` は runtime step contract の validation 用語とし、fan-out 形と矛盾しないことを spec で明記する
- 既存 in-process 実装をすぐ全部置き換えると移行コストが高い
  - mitigation: first step では in-process adapter を残し、標準契約への写像を先に固定する
- external backend を想定し過ぎて plan が広がる
  - mitigation: 今回は local subprocess で標準契約を成立させ、networked adapter は後続に送る

## Design Decisions

- external backend 対応だけでなく、別 repo の game master を platform から載せられることを primary goal に置く
- game master 開発者向け仕様書を Phase 3 の成果物に含める
- 採用案は「game master 専用の論理契約を定義し、transport は adapter で差し替える」案とする
- match loop の主導権は platform に残すが、turn progression の細部は game master が `DecisionStep.requests` で表現する
- game master は request 対象 player を明示できる
  - 複数 player を返せば同時処理
  - 1 player を返せば逐次処理
- `DecisionMode` は metadata ではなく runtime step contract に残し、`DecisionStep.requests` と矛盾しないよう validation する
- sequential game の自動 skip は request 非送信で表現できる
- game master 実装者が毎 turn の public state 更新や観測都合を優先したい場合は、強制 pass request を送る実装も許可する
- `echo-count` と `echo-count-subprocess` は同挙動の別 fixture game として登録し、通常経路に mode 切替を持ち込まない
- `janken` は既存経路を保ったまま新境界へ適応させる
- `turn_mode` は当面残さず、この phase で compatibility metadata から外す
