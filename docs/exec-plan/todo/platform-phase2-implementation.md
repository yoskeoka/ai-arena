# platform-phase2-implementation
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`docs/specs/platform.md` を実装可能な最小単位へ分解し、platform コアの初期実装を段階的に成立させる。

この plan の主目的は以下。

- JSON-RPC 2.0 + NDJSON transport を持つ platform コアを Go で実装する
- 長寿命 AI session、同時行動/順番制の両 turn model、match record/export を実装する
- turn 境界で restart-from-snapshot 可能な snapshot を出力し、platform / game protocol の開発・デバッグに使えるようにする
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

追加前提:

- game とそれに対応する AI は、同じ game protocol 契約を実装していることを事前検証できる必要がある
- そのため、game 仕様と AI 提出物の双方が共有する `game_protocol_id` を導入する
- ID は中央集権的な採番を前提にせず、少なくとも UUIDv7 のような十分衝突しにくい stable ID を使えるようにする
- この ID は runner の match 起動時だけでなく、将来の game 登録 / AI 登録 / AI 更新時の互換性バリデーションにも使う

`game_protocol_id` の修正案:

- `game_protocol_id` は game 名ではなく、`init` / `turn` / `game_over` の payload shape、action schema、snapshot restart に必要な metadata を含む「互換プロトコル契約」を識別する ID として扱う
- 互換性を壊さないルール調整や balance 調整は `ruleset_version` で表現し、`game_protocol_id` は維持してよい
- `init` / `turn` / `game_over` の必須フィールド変更、型変更、action schema の非後方互換変更、snapshot 復元 contract の変更が入る場合は `game_protocol_id` を変える
- Phase 2 の local subprocess 実装では、AI metadata は sidecar manifest から読む前提を既定にする。将来 WASM custom section などへ移しても、runner が見る論理 metadata 項目は維持する

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
- `no_action` に至った理由は match record に別フィールドで記録する
- game master は理由分類に依存せず、ゲーム仕様どおり `no_action` を処理する
- transport / JSON-RPC として壊れているかどうかは platform が判定する
- game 固有 schema や legal move かどうかは game 側 validator が判定する

推奨する記録分類:

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

ただし責務境界は以下。

- `invalid-timeout`
  - platform が判定する
- `invalid-protocol-malformed`
  - platform が判定する
- `invalid-protocol-mismatched-id`
  - platform が判定する
- `invalid-illegal-action`
  - platform 単独ではなく、game validator の判定結果として記録する

必要なら record 上は `failure_source=platform|game` を持たせ、同じ `no_action` でも原因の責務境界を追えるようにする。

必要なら execution 時に `invalid-protocol-unexpected-output` や `invalid-protocol-late-response` のような細分類を追加してよいが、少なくとも上記 4 種は区別できるようにする。

インターフェース修正案:

- session 層は AI から返った生の `result` payload を保持したまま match 層へ返す
- game package は `ValidateAction(rawResult)` を持ち、`accepted` または `invalid-illegal-action -> no_action` へ正規化する
- match loop は platform failure と game validation failure を同じ `TurnOutcome` で扱うが、`failure_reason` と `failure_source` を分離して記録する
- game master の resolve/apply 系 API は、validation 済みの `accepted | no_action` だけを受け取る

`echo-count` fixture での扱い:

- game master は `accepted` なら一致値を検証して成功を記録する
- `accepted` 候補だが値が期待値と違う場合は、platform ではなく game spec 側の無効行動なので、game validator が `invalid-illegal-action` を返し、match record へ記録しつつ game master へは `no_action` として渡す
- `invalid-timeout` / `invalid-protocol-*` は platform 層で記録し、game master へは `no_action` として渡す

この分離により、platform test では「どう壊れたか」を見え、game test では「`no_action` をどう処理するか」だけを見ればよくなる。

### timeout 後の遅延レスポンス方針

長寿命 session では、timeout 判定後に古い request への response が遅れて到着するケースを明示的に扱う必要がある。

修正案:

- timeout 済み request の `id` に対応する遅延レスポンスは、その turn の有効入力へ復帰させず破棄する
- 遅延レスポンスは `invalid-protocol-late-response` として記録し、少なくとも player event counters と turn record から辿れるようにする
- 次の request を送る前に、session は stdout reader 側で stale response を識別できる必要がある
- 遅延レスポンス単発では即 kill しないが、閾値超過時は他の protocol violation と同様に AI 強制停止対象へ含めてよい

## snapshot の目的と非目標

この plan における turn 境界 snapshot の主目的は、本番障害時の完全復旧ではなく、platform と game protocol の開発・デバッグ支援である。

主目的:

- 特定 turn 直前の game state を保存し、bug 再現や局面切り出しをやりやすくする
- `arena-runner` から snapshot を入力して、その局面から game を再開できるようにする
- game master の state 遷移、timeout 処理、invalid action 処理を局所的に検証できるようにする

非目標:

- AI player のプロセスメモリまで含めた完全 continuation
- 本番障害時の厳密な in-flight 復旧
- turn ごとの replay history 出力そのものをこの plan で完成させること

用語整理:

- `snapshot`: game master を turn 境界から再開するための canonical state と補助 metadata
- `exported snapshot`: 観戦・記録向けに公開してよい状態表現
- `restart-from-snapshot`: snapshot を入力して match をその局面から再開すること。AI メモリは復元しないため、`resume` ではなくこの表現を使う

replay について:

- 将来的には game master 側 RNG seed と history 出力を組み合わせた replay player を作る価値が高い
- ただし snapshot が正しく出力できれば後続開発で十分に着手しやすいため、この plan では restart 可能 snapshot の定義と出力までを優先する

## Spec Changes

実装前に以下の spec 更新を行う。

### 1. `docs/specs/platform.md`

- 初期実装スコープを追記する
  - Phase 2a: local process runtime adapter
  - Phase 2b: WASM adapter は後続
- match record / snapshot / exported snapshot / stderr capture の最小データ形を明記する
- snapshot の主目的が開発・デバッグ用の restart-from-snapshot であり、本番完全復旧は非目標であることを明記する
- snapshot に必要な最低 metadata
  - `game_protocol_id`
  - `ruleset_version`
  - turn number
  - current turn mode state
  - game master canonical state
  - 必要なら RNG seed または RNG state
- transport/protocol violation の記録項目を実装可能な粒度に具体化する
- game master に渡す正規化 action と、match record に残す failure reason を分離して定義する
- platform 判定可能な failure と game 判定の illegal action を分けて定義する
- raw AI response を game validator が正規化し、その後の game master には `accepted | no_action` だけを渡す contract を定義する
- timeout 後の遅延レスポンス (`late response`) の記録方針と破棄方針を定義する
- `game_protocol_id` を game metadata / AI metadata / `init` payload に含める
- runner と将来の登録フローで `game_protocol_id` 一致確認を行うことを定義する
- `game_protocol_id` と `ruleset_version` の責務差分、および AI metadata の取得元としての sidecar manifest 既定を定義する

### 2. `docs/specs/platform.md` の fixture appendix

- `echo-count` fixture game のルール、`init` / `turn` / `game_over` payload、mode 別進行、score/placement の定義を書く
- restart-from-snapshot に必要な snapshot 入出力形を定義する
- simultaneous / sequential の両 mode の example transcript を載せる
- failure mode 用 AI を使った expected record 例を載せる
- `accepted` / `no_action` の game master 入力と、`invalid-timeout` / `invalid-protocol-malformed` / `invalid-protocol-mismatched-id` / `invalid-illegal-action` の記録例を載せる
- `invalid-protocol-late-response` と init/shutdown failure の expected record 例を載せる
- 特に `invalid-illegal-action` は game 側判定であり、platform 単独では決めないことを明記する
- `echo-count` の `game_protocol_id` と、AI metadata 側での一致要件を明記する

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
- `cmd/arena-runner/`
  - オンラインマッチングや常駐 server を介さず、CLI 引数や設定ファイルで指定した game master と AI player 群から match 進行を直接開始する単発 runner
- `internal/platform/protocol/`
  - JSON-RPC 2.0 envelope
  - NDJSON reader/writer
  - request id matching
- `internal/platform/runtime/`
  - AI runtime interface
  - local subprocess adapter
  - stderr capture / lifecycle management
- `internal/platform/session/`
  - `init` / `turn` / `game_over` 送受信
  - deadline / timeout / protocol violation handling
- `internal/platform/match/`
  - match loop
  - simultaneous / sequential scheduler
  - match result / match record
  - restart-from-snapshot entrypoint
- `internal/platform/game/`
  - game master interface
  - action validator interface
  - exported snapshot interface
- `internal/platform/catalog/`
  - game metadata
  - AI metadata
  - `game_protocol_id` validation
- `internal/games/echo/`
  - `echo-count` fixture game master
- `internal/games/janken/`
  - 後続で入れる richer integration game 実装の受け皿
- `testdata/ai/echo/`
  - echo AI
  - timeout AI
  - invalid action AI
  - bad JSON AI
  - late response AI
  - init-timeout AI
  - exit-after-init AI
  - hung-after-game-over AI
- `e2e/`
  - CLI 起動ベースの black-box tests

実際の package 分割は execution で調整してよいが、以下は崩さない。

- protocol と game logic を分離する
- runtime adapter と session/match loop を分離する
- fixture game と main game (`janken`) を分離する
- runner の責務は「CLI 引数や設定ファイルで与えた入力から match を起動して結果を出すところまで」に留め、将来の server 常駐プロセス責務と混ぜない
- protocol 互換性判定は game 名の文字列比較ではなく `game_protocol_id` で行う

## Execution Strategy

### Task 1: Go module と protocol 最小実装を立ち上げる

- `go.mod` と最小 test target を追加する
- JSON-RPC 2.0 envelope 型、NDJSON reader/writer、request/response correlation を実装する
- `game_protocol_id` を含む最小 metadata 型を定義する

Verification:

- unit test: valid request/response encode-decode
- unit test: 複数 message の NDJSON framing
- unit test: malformed JSON / wrong `id` / invalid envelope の判定
- unit test: metadata の `game_protocol_id` 必須チェック
- unit test: `game_protocol_id` と `ruleset_version` の責務差分に沿った validation

### Task 2: local process runtime adapter を実装する

- AI runtime interface を定義する
- 子プロセス起動、stdin/stdout/stderr 接続、shutdown を持つ local subprocess adapter を実装する
- stderr を phase-aware に蓄積する仕組みを入れる
- runner が参照できる AI metadata 読み出し口を定義する

Verification:

- unit test: 起動成功時に stream が接続される
- unit test: stderr capture 上限が適用される
- unit test: process start failure が `init` 前失敗として扱われる
- unit test: AI metadata の `game_protocol_id` を取得できる
- unit test: `game_over` 後の shutdown timeout 超過で強制停止できる

### Task 3: AI session 層を実装する

- `init`, `turn`, `game_over` の送信 API を作る
- per-request deadline と timeout を扱う
- protocol violation を turn failure として記録できるようにする
- game master に渡す正規化 action を `accepted` / `no_action` にそろえ、failure reason は別 record に保持する
- platform 層では transport/protocol 系 failure までを判定し、game legal move 判定は game 層へ委譲する
- timeout 後に届いた stale response を後続 turn の応答として誤採用しないようにする

Verification:

- unit test: `init` ACK を正常受理できる
- unit test: `turn` timeout が `no_action` 相当として返る
- unit test: `init` timeout または bad ACK が init failure として記録される
- unit test: `game_over` が notification として送られ response を待たない
- unit test: bad JSON / mismatched id / unexpected stdout を protocol violation として記録する
- unit test: `invalid-timeout` / `invalid-protocol-malformed` / `invalid-protocol-mismatched-id` がそれぞれ別 reason で記録される
- unit test: timeout 後の late response が `invalid-protocol-late-response` として記録され、次 turn の response と混線しない

### Task 4: game master 契約と match loop を実装する

- game master interface を定義する
- action validator interface を定義する
- simultaneous / sequential scheduler を実装する
- turn 収集結果を game master に適用する

Verification:

- unit test: simultaneous mode で全員応答待ち後に 1 回だけ resolve される
- unit test: sequential mode で手番プレイヤーだけに request が送られる
- unit test: timeout / invalid action が game master へ `no_action` 相当で渡る
- unit test: raw AI result は game validator を通るまで resolve 対象へ入らない

### Task 5: match record / exported snapshot / observability を実装する

- turn 境界 snapshot
- player ごとの timeout / invalid / protocol violation count
- stderr / lifecycle event / final placement を含む match record を定義する
- snapshot は restart-from-snapshot 用 canonical state と exported snapshot を区別して定義する
- player ごとの `action_status` と `failure_reason` を turn 単位で残せるようにする
- 必要なら `failure_source` も持たせ、platform 起因か game 起因か区別できるようにする
- init failure / turn failure / shutdown failure を phase ごとに残せるようにする

Verification:

- unit test: turn ごと restart 可能 snapshot が残る
- unit test: player event counters が正しく集計される
- unit test: final result に placement と score が入る
- unit test: `no_action` と `failure_reason` が独立して記録される
- unit test: `invalid-illegal-action` は game 側判定の結果として記録される
- unit test: stderr truncation と shutdown timeout 超過が lifecycle event として残る

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
- `late-response-ai`: timeout 後に古い request への response を遅延送信する
- `init-timeout-ai`: `init` request に応答しない
- `exit-after-init-ai`: `init` 後に早期終了する
- `hung-after-game-over-ai`: `game_over` 後も終了せず強制停止を必要とする

Verification:

- 外形 e2e 前に各 AI を単体で起動し、期待 stdout/stderr を返す最小 test を作る

### Task 8: black-box e2e で platform 単体の happy path を閉じる

- `arena-runner` から `echo-count` simultaneous match を 2 echo AI で起動する
- `arena-runner` から `echo-count` sequential match を 2 echo AI で起動する

Verification:

- e2e: simultaneous transcript, final score, final snapshot, stderr capture が期待通り
- e2e: runner が `game_protocol_id` 一致ケースだけ起動を許可する
- e2e: sequential transcript, turn order, final snapshot が期待通り
- e2e: sidecar manifest 由来 metadata で `game_protocol_id` / `ruleset_version` が期待通り読まれる

### Task 9: black-box e2e で失敗系を閉じる

- 片方を `timeout-ai` に差し替えた match
- 片方を `invalid-action-ai` に差し替えた match
- 片方を `bad-json-ai` に差し替えた match
- 片方を `late-response-ai` に差し替えた match
- `init-timeout-ai` または `exit-after-init-ai` を指定した match
- `hung-after-game-over-ai` を含む match
- `game_protocol_id` 不一致の AI を指定した match

Verification:

- e2e: timeout count / invalid action count / protocol violation count が match record に出る
- e2e: timeout / malformed / mismatched-id / illegal-action が別 reason で記録される
- e2e: late response が別 reason で記録され、後続 turn へ混線しない
- e2e: failure player だけ placement が悪化する
- e2e: 残り player の進行は継続される
- e2e: init failure は match 開始前または開始直後に明示記録される
- e2e: `game_over` 後に終了しない AI は shutdown timeout 後に強制停止され、その lifecycle event が残る
- e2e: `game_protocol_id` 不一致なら runner が開始前に明示エラーで落ちる

### Task 10: restart-from-snapshot を追加する

- `arena-runner` が snapshot file を入力として受け取れるようにする
- game master canonical state と必要 metadata から、指定 turn 境界の局面を再開できるようにする
- 再開時の AI は新規プロセスとして起動し、snapshot に対応する `init` / 可視状態を渡す
- これは開発・デバッグ主用途であり、本番完全復旧を保証しない

Verification:

- unit test: snapshot serialize / deserialize 後に game master state を復元できる
- e2e: 途中 turn の snapshot から `echo-count` match を再開できる
- e2e: restart 後は AI メモリ continuity を保証しないことが明示される

### Task 11: `janken` 実装へつなぐ richer integration の入口を作る

- `janken` game master の package と test skeleton だけ作るか、もしくは follow-up plan を切る
- `echo-count` だけでは不足する coverage を `janken` 側へ明示的に送る

Verification:

- docs review: `echo-count` で担保済みの責務と `janken` に残す責務が区別されている

## Sub-tasks

- [ ] Spec update: `platform.md` appendix / `janken-game.md`
- [ ] Define `game_protocol_id` metadata and validation rules
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
- [ ] [depends on: Add failure-mode e2e tests] Write the follow-up exec plan for `janken` richer integration

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
- snapshot を本番完全復旧前提で設計すると、AI メモリ復元不能とのギャップで責務が膨らむ
  - mitigation: この plan では restart-from-snapshot を開発・デバッグ用途に限定し、完全 continuation は非目標と明記する

## Resolved Decisions

- `janken` はこの plan の execution に含めず、platform foundation 実装完了後に改めて exec plan を作成する
- `echo-count` は独立 game spec を持たず、`platform.md` の付録 fixture として扱う

理由:

- `echo-count` は platform 検証用 fixture であり、ゲームとしての独立価値がほぼない
- 新ゲームプロトコル開発の参照先としては `janken` や後続の本命ゲームの方が有用
