# platform-phase2-implementation
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`docs/specs/platform.md` を実装可能な最小単位へ分解し、platform コアの初期実装を段階的に成立させる。

この plan の主目的は以下。

- JSON-RPC 2.0 + NDJSON transport を持つ platform コアを Go で実装する
- 長寿命 AI session、同時行動/順番制の両 turn model、match record/export を実装する
- 実装確認用の deterministic な platform e2e fixture game を追加し、各段階を unit test または外形 e2e で閉じる
- ADR で維持されている Phase 2 実証ゲーム `janken` へ進む前に、platform 単体の動作保証面を先に固める

## Planning Context

- `docs/project-plan.md` は platform をゲーム非依存基盤として位置付け、同時行動/順番制の両対応、JSON-RPC 2.0、stderr 記録、観戦向け全体状態公開を要求している
- `docs/design-decisions/adr.md` では以下が既に決まっている
  - AI 実行環境は最終的に WASM + wazero
  - AI 通信は stdin/stdout + JSON-RPC 2.0
  - Phase 2 の最初の実装検証はローカルプロセス実行を使う
  - Phase 2 の実証ゲームは `janken`
- `docs/design-decisions/core-beliefs.md` に従い、spec を先に更新し、実装は spec と test で拘束する

過去判断との整合:

- `janken` を Phase 2 の主たる実証ゲームとして扱う ADR は維持する
- 今回追加するのは `janken` の代替ではなく、platform 単体検証のための e2e fixture game とする
- これにより、turn orchestration / timeout / protocol handling の不具合と、ゲーム固有ロジックの不具合を切り分けやすくする

## E2E Test Game Options

### Option A: `janken` だけで platform e2e を兼ねる

利点:

- 追加の fixture game spec が不要
- Phase 2 の主実証ゲームに直接つながる

欠点:

- 同時行動・順位計算・ゲーム固有解決が同時に入るため、platform 不具合の切り分けが重い
- 順番制 turn model を別途検証できない
- protocol/timeout/recording の初期確認としてはゲーム固有ノイズが多い

### Option B: オウム返し fixture game を platform e2e 専用に追加する

利点:

- deterministic で transcript を固定しやすく、外形 e2e が壊れにくい
- 同じ message schema で simultaneous / sequential の両 mode を持たせやすい
- platform の責務だけを先に閉じられる

欠点:

- `janken` とは別に fixture 用 spec と sample AI が増える
- 隠し情報や richer action validation はほとんど検証できない

### Option C: オウム返し fixture game を拡張し、fault injection AI も同梱する

利点:

- happy path だけでなく timeout / invalid action / protocol violation の platform 挙動まで deterministic に検証できる
- 追加ゲームは 1 つのまま、AI 側差し替えだけで失敗系を確認できる

欠点:

- fixture game 自体の責務が少し増える
- 「ゲーム」と「テスト用 AI 群」の境界を plan で明記しないと肥大化しやすい

## Recommendation

推奨は **Option C**。

理由:

- `docs/specs/platform.md` の主要リスクはゲームの面白さではなく、transport / deadline / turn orchestration / logging / record export の正しさにある
- `janken` は Phase 2 の richer integration として残しつつ、その前段に deterministic fixture を入れる方が ADR の「先にローカルプロセス実行で切り分ける」と整合する
- オウム返し game 単体では順位差や失敗系の確認が弱いが、fault injection AI を同梱すれば platform の責務検証として十分な厚みになる

**Human confirmation needed before execution**:
この plan は `janken` を置き換えず、`platform` 専用 fixture として `echo-count` 系ゲームを追加する前提で組む。

## Recommended Fixture Design

fixture game 名は仮に `echo-count` とする。

ルール:

- ゲームマスターは deterministic に整数カウンタを進める
- 各 turn request で対象プレイヤーへ期待値となる数字を渡す
- AI は受け取った数字をそのまま返す
- 一致すれば `accepted` として解決する
- ゲームマスターに渡す turn 入力は、platform 側で正規化した `accepted` または `no_action` のどちらかに限定する
- `no_action` になった理由は game master に埋め込まず、platform record 側で原因分類を保持する

mode:

- `simultaneous`: 同一 turn で両 AI に同じ数字を送り、全員の応答後に解決する
- `sequential`: 手番プレイヤー 1 人だけに数字を送り、応答ごとに次の手番へ進む

この案の評価:

- ユーザー提案どおり、同一 fixture で simultaneous / sequential の両 turn model を検証できる
- transcript が単純なので e2e assertion を「送信順」「受信順」「match record」「final snapshot」で固定しやすい
- 一方で、隠し情報・複雑な legal action・順位決定の厚みは弱いので、`platform` の最終受け入れをこれだけで済ませるのは不足

改善提案:

- happy-path AI だけでなく `timeout-ai` / `invalid-action-ai` / `bad-json-ai` を fixture 群として追加する
- placement の差が必要な test では 1 人だけ fault injection AI を使い、score/placement/timeout count まで検証する
- `janken` はこの plan 完了後の follow-up 実装で、隠し情報・同時解決・順位付けの richer coverage を担わせる

### `no_action` と platform 記録分類

ここは execution 前に spec で明確化する。

原則:

- game master に渡す正規化済み turn outcome は `accepted` または `no_action` だけにする
- platform は `no_action` に至った理由を別フィールドで記録する
- game master は理由分類に依存せず、ゲーム仕様どおり `no_action` を処理する

推奨する platform 記録分類:

- `accepted`
  - request に対応する合法レスポンスを期限内に返した
- `invalid-timeout`
  - 期限内応答なし
- `invalid-protocol-malformed`
  - JSON parse 不可、NDJSON framing 不正、JSON-RPC envelope 不正
- `invalid-protocol-mismatched-id`
  - `id` 不一致、response 相関不能
- `invalid-illegal-action`
  - JSON-RPC としては正しいが、ゲーム仕様上の action schema または legal action に違反

必要なら execution 時に `invalid-protocol-unexpected-output` のような細分類を追加してよいが、少なくとも上記 4 種は区別できるようにする。

`echo-count` fixture での扱い:

- game master は `accepted` なら一致値を検証して成功を記録する
- `accepted` だが値が期待値と違う場合は、platform ではなく game spec 側の無効行動なので `invalid-illegal-action` を記録しつつ game master へは `no_action` として渡す
- `invalid-timeout` / `invalid-protocol-*` は platform 層で記録し、game master へは `no_action` として渡す

この分離により、platform test では「どう壊れたか」を見え、game test では「`no_action` をどう処理するか」だけを見ればよくなる。

## Spec Changes

実装前に以下の spec 更新を行う。

### 1. `docs/specs/platform.md`

- 初期実装スコープを追記する
  - Phase 2a: local process runtime adapter
  - Phase 2b: WASM adapter は後続
- match record / exported snapshot / stderr capture の最小データ形を明記する
- transport/protocol violation の記録項目を実装可能な粒度に具体化する
- game master に渡す正規化 action と、platform record に残す failure reason を分離して定義する

### 2. `docs/specs/platform-fixture-echo-count.md` を新規追加

- `echo-count` fixture game のルール、`init` / `turn` / `game_over` payload、mode 別進行、score/placement の定義を書く
- simultaneous / sequential の両 mode の example transcript を載せる
- failure mode 用 AI を使った expected record 例を載せる
- `accepted` / `no_action` の game master 入力と、`invalid-timeout` / `invalid-protocol-malformed` / `invalid-protocol-mismatched-id` / `invalid-illegal-action` の記録例を載せる

### 3. `docs/specs/janken-game.md`

- `janken` の役割を「platform fixture 完了後の richer integration game」として補足する
- `echo-count` fixture と責務が重ならないことを明記する

ADR 追加は不要の見込み。
理由:

- `janken` を主実証ゲームとする既存 ADR を変更しない
- `echo-count` は test fixture の追加であり、アーキテクチャ方針の変更ではない

## Expected Code Changes

初回実装では以下のような Go module 構成を想定する。

- `go.mod`
- `cmd/arena-local/`
  - local process runtime で match を起動する CLI
- `internal/platform/protocol/`
  - JSON-RPC 2.0 envelope
  - NDJSON reader/writer
  - request id matching
- `internal/platform/runtime/`
  - AI runtime interface
  - local process adapter
  - stderr capture / lifecycle management
- `internal/platform/session/`
  - `init` / `turn` / `game_over` 送受信
  - deadline / timeout / protocol violation handling
- `internal/platform/match/`
  - match loop
  - simultaneous / sequential scheduler
  - match result / match record
- `internal/platform/game/`
  - game master interface
  - exported snapshot interface
- `internal/games/echo/`
  - `echo-count` fixture game master
- `internal/games/janken/`
  - 後続で入れる richer integration game 実装の受け皿
- `testdata/ai/echo/`
  - echo AI
  - timeout AI
  - invalid action AI
  - bad JSON AI
- `e2e/`
  - CLI 起動ベースの black-box tests

実際の package 分割は execution で調整してよいが、以下は崩さない。

- protocol と game logic を分離する
- runtime adapter と session/match loop を分離する
- fixture game と main game (`janken`) を分離する

## Execution Strategy

### Task 1: Go module と protocol 最小実装を立ち上げる

- `go.mod` と最小 test target を追加する
- JSON-RPC 2.0 envelope 型、NDJSON reader/writer、request/response correlation を実装する

Verification:

- unit test: valid request/response encode-decode
- unit test: 複数 message の NDJSON framing
- unit test: malformed JSON / wrong `id` / invalid envelope の判定

### Task 2: local process runtime adapter を実装する

- AI runtime interface を定義する
- 子プロセス起動、stdin/stdout/stderr 接続、shutdown を持つ local adapter を実装する
- stderr を phase-aware に蓄積する仕組みを入れる

Verification:

- unit test: 起動成功時に stream が接続される
- unit test: stderr capture 上限が適用される
- unit test: process start failure が `init` 前失敗として扱われる

### Task 3: AI session 層を実装する

- `init`, `turn`, `game_over` の送信 API を作る
- per-request deadline と timeout を扱う
- protocol violation を turn failure として記録できるようにする
- game master に渡す正規化 action を `accepted` / `no_action` にそろえ、failure reason は別 record に保持する

Verification:

- unit test: `init` ACK を正常受理できる
- unit test: `turn` timeout が `no_action` 相当として返る
- unit test: `game_over` が notification として送られ response を待たない
- unit test: bad JSON / mismatched id / unexpected stdout を protocol violation として記録する
- unit test: `invalid-timeout` / `invalid-protocol-malformed` / `invalid-protocol-mismatched-id` がそれぞれ別 reason で記録される

### Task 4: game master 契約と match loop を実装する

- game master interface を定義する
- simultaneous / sequential scheduler を実装する
- turn 収集結果を game master に適用する

Verification:

- unit test: simultaneous mode で全員応答待ち後に 1 回だけ resolve される
- unit test: sequential mode で手番プレイヤーだけに request が送られる
- unit test: timeout / invalid action が game master へ `no_action` 相当で渡る

### Task 5: match record / exported snapshot / observability を実装する

- turn 境界 snapshot
- player ごとの timeout / invalid / protocol violation count
- stderr / lifecycle event / final placement を含む match record を定義する
- player ごとの `action_status` と `failure_reason` を turn 単位で残せるようにする

Verification:

- unit test: turn ごと snapshot が残る
- unit test: player event counters が正しく集計される
- unit test: final result に placement と score が入る
- unit test: `no_action` と `failure_reason` が独立して記録される

### Task 6: `echo-count` fixture game を実装する

- simultaneous / sequential 両 mode を持つ deterministic game master を作る
- accepted / invalid / timeout の score ルールを定義する
- exported snapshot を安定した JSON 形で出せるようにする
- game master は `accepted` または `no_action` だけを入力として受ける

Verification:

- unit test: simultaneous mode の解決規則
- unit test: sequential mode の手番遷移
- unit test: score / placement / summary の計算
- unit test: illegal echoed value は game spec 上 `no_action` に落ち、platform record には `invalid-illegal-action` が残る

### Task 7: fixture AI 群を実装する

- `echo-ai`: 受信数字をそのまま返す
- `timeout-ai`: 一部 turn を意図的に無応答にする
- `invalid-action-ai`: schema 上不正な action を返す
- `bad-json-ai`: protocol violation を起こす

Verification:

- 外形 e2e 前に各 AI を単体で起動し、期待 stdout/stderr を返す最小 test を作る

### Task 8: black-box e2e で platform 単体の happy path を閉じる

- CLI から `echo-count` simultaneous match を 2 echo AI で起動する
- CLI から `echo-count` sequential match を 2 echo AI で起動する

Verification:

- e2e: simultaneous transcript, final score, final snapshot, stderr capture が期待通り
- e2e: sequential transcript, turn order, final snapshot が期待通り

### Task 9: black-box e2e で失敗系を閉じる

- 片方を `timeout-ai` に差し替えた match
- 片方を `invalid-action-ai` に差し替えた match
- 片方を `bad-json-ai` に差し替えた match

Verification:

- e2e: timeout count / invalid action count / protocol violation count が match record に出る
- e2e: timeout / malformed / mismatched-id / illegal-action が別 reason で記録される
- e2e: failure player だけ placement が悪化する
- e2e: 残り player の進行は継続される

### Task 10: `janken` 実装へつなぐ richer integration の入口を作る

- `janken` game master の package と test skeleton だけ作るか、もしくは follow-up plan を切る
- `echo-count` だけでは不足する coverage を `janken` 側へ明示的に送る

Verification:

- docs review: `echo-count` で担保済みの責務と `janken` に残す責務が区別されている

## Sub-tasks

- [ ] Spec update: `platform.md` / `platform-fixture-echo-count.md` / `janken-game.md`
- [ ] [parallel] Bootstrap protocol package and tests
- [ ] [parallel] Design runtime/session interfaces
- [ ] [depends on: Bootstrap protocol package and tests, Design runtime/session interfaces] Implement local runtime adapter
- [ ] [depends on: Implement local runtime adapter] Implement AI session layer
- [ ] [depends on: Implement AI session layer] Implement match loop and game master contract
- [ ] [depends on: Implement match loop and game master contract] Implement match record and snapshot export
- [ ] [depends on: Implement match loop and game master contract] Implement `echo-count` fixture game
- [ ] [depends on: Implement `echo-count` fixture game] Add fixture AI programs
- [ ] [depends on: Add fixture AI programs, Implement match record and snapshot export] Add happy-path e2e tests
- [ ] [depends on: Add happy-path e2e tests] Add failure-mode e2e tests
- [ ] [depends on: Add failure-mode e2e tests] Decide whether `janken` starts in the same execution PR or a follow-up plan

## Parallelism

- protocol package 実装と runtime/session interface 設計は独立して進められる
- match record 実装と fixture game 実装は match loop 契約が固まった後なら並行できる
- simultaneous / sequential e2e は同一 fixture 上で別ケースとして分けられる

## Risks and Mitigations

- `echo-count` だけで platform 受け入れを終えると、隠し情報・複雑 action schema・順位 tie-break の coverage が不足する
  - mitigation: `janken` を follow-up integration として明示的に残す
- local process adapter の設計が WASM adapter の将来差し替えを妨げる可能性がある
  - mitigation: runtime interface を先に固定し、adapter を差し替え可能にする
- e2e が transcript 全比較に寄りすぎると壊れやすい
  - mitigation: transport log の全文一致ではなく、turn order / counters / score / snapshot の意味的 assertion を中心にする

## Open Questions

- `janken` 実装をこの plan の execution に含めるか、それとも `platform` foundation 完了後に別 exec plan を切るか
- `echo-count` fixture spec を `platform.md` の付録として持つか、独立 spec にするか

推奨:

- `janken` は follow-up plan に分離する
- `echo-count` fixture は独立 spec にする
