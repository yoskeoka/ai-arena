# Lessons

## [2026-05-12] 連続 execution plan にまたがる issue は最後の plan まで閉じない

- **Mistake**: 今回の plan で部分的に refactor が進んだ段階で、連続 refactor line 全体の親 issue を `docs/issues/done/` へ移してしまった
- **Pattern**: 「今回の差分で issue の方向性に着手したこと」と「issue に書かれた follow-up line 全体を完了したこと」を分離せず、途中 PR で close 判定してしまう
- **Rule**: issue が後続 plan までまたがる tracking 用の親 issue なら、途中の execution PR では `done/` へ移さない。PR body に `remains open: <closing plan>` を明記し、issue の close は最後の plan を終えた PR にだけ載せる
- **Applied**: `docs/issues/dungeon-post-phase5-refactor-before-feature-expansion.md` と `platform-phase5-06` 系の連続 refactor plan、今後の multi-plan refactor tracking issue 運用

## [2026-05-12] 再発調査用 issue には失敗 output を残す

- **Mistake**: unrelated regression を `docs/issues/` へ切り出したとき、失敗テスト名と推測だけを書き、実際の assertion や event log 抜粋を十分に残さなかった
- **Pattern**: 「後で main で再現すればよい」と考えて、その場でしか取れない failure output を durable note に落としきらない
- **Rule**: 偶発かもしれない test/runtime failure を `docs/issues/` へ記録するときは、少なくとも実行コマンド、失敗した test 名、代表 assertion、再調査に効く output 抜粋を同じ issue に残す。完全ログの保存先があるならその path も書く
- **Applied**: `docs/issues/done/arena-runner-e2e-init-regression.md`、今後の ai-arena verification blocker 切り出し全般

## [2026-05-09] heuristic AI の e2e は固定期待値にしない

- **Mistake**: runtime/e2e 確認まで reference AI の思考品質を直接担保しようとして、seed 付きでも heuristic 変更に弱い assertion を足しかけた
- **Pattern**: 「runtime 経路が正しいか」と「baseline 戦略が妥当か」を同じ e2e で固定しようとして、fixture の安定性より bot 調整自由度を優先してしまう
- **Rule**: heuristic な reference AI は unit test と manual verification で評価する。e2e では seed 固定の scripted AI を使い、runtime 経路と結果整合だけを確認する
- **Applied**: `e2e/arena_runner_wasme2e_test.go` の dungeon WASM 確認、今後の dungeon / imperfect-information game の AI verification 設計

## [2026-04-26] ドキュメント言語方針の適用範囲を先に固定する

- **Mistake**: 「docs を日本語にする」を広く解釈し、コミットメッセージや PR メタデータまで同じ方針で寄せる前提で進めかけた
- **Pattern**: リポジトリ内ドキュメント言語と、コード・VCS・GitHub メタデータの言語を分離せずに扱ってしまう
- **Rule**: 言語方針を扱うときは、少なくとも `docs/*`、`AGENTS.md`、生成物 UI/メッセージ、コードコメント、コミットメッセージ、PR title/description の6領域に分けて適用範囲を明示する
- **Applied**: `AGENTS.md` の運用ルール、`docs/design-decisions/adr.md` の内部ドキュメント言語 ADR、今後の `ai-arena` ドキュメント/PR 運用

## [2026-04-26] timeout / invalid action の仕様は逃げ得を先に潰す

- **Mistake**: `no_action` を通常の引き分け解決に巻き込むと、意図的に応答しないことへ戦術上の余地を残してしまう
- **Pattern**: timeout や invalid action を「単なる欠損入力」として扱い、インセンティブ設計まで詰めずに仕様化してしまう
- **Rule**: 対戦ゲームで timeout / invalid action を定義するときは、まず「意図的に出さないことにメリットがないか」を確認し、必要なら個別敗北扱いを明記する
- **Applied**: `docs/specs/janken-game.md` の `no_action` 解決規則、今後の AI Arena ゲーム仕様

## [2026-04-27] 起動確認とゲーム初期化確認を分ける

- **Mistake**: `init` の成功をそのまま runtime health 全体の確認とみなすと、platform が確認できる範囲とゲーム固有初期化の範囲が混ざる
- **Pattern**: 実行基盤の起動成功と、ゲームプロトコル上の準備完了を同じ `ready` 概念で扱ってしまう
- **Rule**: platform spec では「load / instantiate / stream 接続」と「game init request に応答できること」を分けて定義し、ゲーム固有 readiness は `init` 応答の意味として各ゲームが定義する
- **Applied**: `docs/specs/platform.md` の起動確認記述、`docs/specs/janken-game.md` の `ready: true` の意味づけ

## [2026-04-29] match end は one-way notification で定義する

- **Mistake**: 試合終了後の最終結果通知と AI の後処理を、通常の request/response と同じ形で扱うと、不要な応答義務と runtime 終了責務が曖昧になる
- **Pattern**: 「結果を渡すこと」と「終了猶予を与えること」を別メカニズムにせず、末尾プロトコルを詰め切らない
- **Rule**: AI が最終振り返りやレポート出力を行う必要があるゲームでは、試合終了メッセージは response 不要の `game_over` notification とし、必要なら `shutdown_after_ms` を明示する
- **Applied**: `docs/specs/platform.md` の `game_over` / shutdown 記述、`docs/specs/janken-game.md` の最終通知例

## [2026-04-29] platform 共通仕様とゲーム固有運用の境界を残す

- **Mistake**: `stderr` の公開タイミングや AI 差し替え条件まで platform 共通仕様で固定すると、ゲームごとの面白さや運営ルール差分を潰してしまう
- **Pattern**: 保存責務と公開タイミング、再起動フックと発火条件を同じ層で定義してしまう
- **Rule**: platform spec では「保存する」「差し替え前処理フックを持つ」までを定義し、公開タイミングや差し替え条件はゲーム仕様へ委ねる
- **Applied**: `docs/specs/platform.md` の `stderr` 取得タイミングと `turn` 前差し替え前処理の記述

## [2026-05-07] fixture 検証都合を product 向け切替機能へ一般化しない

- **Mistake**: `echo-count` の subprocess 検証 needs を、そのまま `arena-runner` の `--game-master-mode` と registry の複数 mode 切替へ一般化した
- **Pattern**: e2e fixture の都合で必要な分岐を、通常経路の user-facing 設定や汎用 registry contract に昇格させてしまう
- **Rule**: ある分岐が fixture/e2e の等価性確認にしか必要ないなら、まず別 fixture game か test-only registration で閉じる。通常利用者が選ばない切替を product path に足さない
- **Applied**: `echo-count` / `echo-count-subprocess` の分離、`arena-runner` の game-master mode 削除、Phase 3 runtime boundary 設計

## [2026-05-07] execution 系依頼は PR 初回 follow-up まで止めない

- **Mistake**: 実装完了時点で区切ってしまい、user が期待している `commit -> push -> PR 作成 -> CI/初回 follow-up` まで進めずに止まった
- **Pattern**: `execute-task` の完了条件をローカル実装とテスト成功に寄せすぎて、repo workflow 上の landing steps を会話上の「次の依頼待ち」と誤認する
- **Rule**: user が `commit` や `PR作成まで` を含む execution 完了を求めたら、完了報告は少なくとも `commit -> push -> PR 作成 -> 30秒待機後の CI/check 確認` を終えてから行う。途中経過は commentary で出し、final は workflow の停止条件を満たした後に限る
- **Applied**: `execute-task` 後の `review-task` 運用、`ai-arena` / `ww` の実装ブランチ handoff、今後の PR 作成依頼全般

## [2026-05-08] spec では実装シンボル名より責務境界を書く

- **Mistake**: spec の説明を補強するつもりで、`registry.Registry.Lookup*` のような実装コード名に寄せた表現を許容しかけた
- **Pattern**: 現在のコード構造をそのまま spec に写してしまい、責務は同じでも小さなリファクタリングで spec 修正が必要な書き方になる
- **Rule**: spec では concrete な関数名・メソッド名・型名をむやみに持ち込まず、まず責務・入出力・境界を書く。実装で使っている型や interface を書く場合も、それが安定した抽象概念として spec の主語になっているときだけに限る
- **Applied**: `docs/specs/platform-game-registry.md` の lookup 流れ、今後の platform / registry / runner 系 spec 全般

## [2026-05-08] AI runtime 差分と game id を混同しない

- **Mistake**: Go-WASM AI player の検証線を追加するとき、AI player runtime の差分をそのまま game id 差分へ持ち込み、`janken-wasm` を別 game として切り出した
- **Pattern**: `runner` が受ける AI player runtime の違いと、game master / game ruleset identity の違いを同じ層で扱ってしまう
- **Rule**: game master 実装と ruleset が同一なら game id は分けない。`local-subprocess` と `wasm-wasi` のような AI player runtime 差分は、まず同一 game id の sidecar/runtime 設定差分として表現する
- **Applied**: `janken` の Go-WASM verification path、今後の AI runtime fixture 設計

## [2026-05-08] dungeon 系コードは別 repo 移設前提をコメントと依存境界で明示する

- **Mistake**: `dungeon` の local sidecar / helper command を mono repo 内 command として追加したとき、`internal` 依存を避ける意図と「将来別 repo へ移せる」前提をコード上で十分に明示しなかった
- **Pattern**: 設計意図は plan/spec にはあるが、コード冒頭の comment や command 依存境界に反映されず、mono repo 専用の実装に見えてしまう
- **Rule**: `dungeon` の game code、sidecar、debug/verification CLI は、mono repo に置いているだけで将来別 repo へ引っ越せる前提で作る。`internal` 依存を持ち込まないだけでなく、その意図を package / file comment 冒頭にも短く書く
- **Applied**: `games/dungeon/*`、`cmd/dungeon-bot-local`、`cmd/dungeon-gamemaster`、`cmd/dungeon-map-helper`、今後の game-side helper/sidecar 設計

## [2026-05-08] Go の exported API comment は慣習ではなく lint で守る

- **Mistake**: exported symbol に comment が不足しているのに、Go では標準で強制されているはずだと曖昧に扱い、repo 側の lint 設定を先に確認しなかった
- **Pattern**: 言語慣習と repo の quality gate を同一視し、何が CI で保証されているかを設定ファイルで確かめる前に前提化してしまう
- **Rule**: Go の doc comment 品質を語るときは、「言語仕様」「一般的慣習」「この repo の lint 設定」を分けて確認する。comment を必須運用にしたいなら、慣習に期待せず lint rule と issue で明示する
- **Applied**: `ai-arena/Makefile` の lint policy、`games/dungeon/*` の exported comment 追加、今後の Go package 公開 API 全般

## [2026-05-08] seed 入力契約と乱数源の内部表現を分ける

- **Mistake**: `rng_seed` をそのまま `int64` 契約で持ち続ける前提で実装を進めると、human-friendly な replay/debug seed と `rand/v2` の内部 seed material を同じ層で固定してしまう
- **Pattern**: CLI / snapshot / exported snapshot で扱う外部 seed と、実装が乱数源へ渡す seed material を分離せずに設計してしまう
- **Rule**: seed-aware game を実装するときは、外部契約では replay/debug しやすい string seed を持ち、binary seed material をそのまま hex string で保持する。内部ではその hex を decode して `rand/v2` へ渡し、hash で別の seed material へ写像しない
- **Applied**: `docs/specs/dungeon-game.md` の `rng_seed` 契約、`cmd/arena-runner` の `--rng-seed`、`games/dungeon` の generated layout 実装、今後の seed-aware game / replay path

## [2026-05-09] platform spec に game 固有 encoding を漏らさない

- **Mistake**: `game-master` の共通契約に、dungeon の `64` 桁 hex seed のような game 固有 encoding をそのまま書き込んだ
- **Pattern**: 個別ゲームの設計で得た知見を platform 契約へ一般化するときに、「共通で守る責務」と「その game だけの具体表現」を分離しきれない
- **Rule**: platform / game-master / runner の共通 spec では、共通で必要な受け渡し責務と保存責務だけを書く。seed の encoding、PRNG 種別、payload shape など game ごとの具体表現は各 game spec に閉じ込める
- **Applied**: `docs/specs/game-master.md` の `rng_seed` 契約、今後の platform 共通 spec と game spec の責務分離

## [2026-05-09] fresh seed 生成責務を platform に持ち込まない

- **Mistake**: `rng_seed` を opaque string として扱う方針にした後も、fresh run の seed 自動生成を runner / registry 側へ残してしまった
- **Pattern**: replay 用の保存・再投入責務と、game 固有 encoding に従った fresh seed 生成責務を同じ層で扱ってしまう
- **Rule**: platform は `rng_seed` を opaque string として保存・再投入するだけに留める。fresh run で seed 未指定時の初期 seed 生成は game master の責務とし、encoding や RNG 選択を platform に教えない
- **Applied**: `cmd/arena-runner` の `--rng-seed` 取扱い、`internal/platform/registry` の dungeon builder、`docs/specs/platform.md` / `docs/specs/game-master.md` / `docs/specs/dungeon-game.md`

## [2026-05-09] replay source に seed があるなら CLI override を許さない

- **Mistake**: `record.json` / `snapshot.json` から `rng_seed` を復元できる場合でも、`--rng-seed` を併用したときに「優先してよい」と曖昧な契約にしてしまった
- **Pattern**: source-of-truth file に含まれる deterministic input と、CLI override を同時に許して優先順位問題を残してしまう
- **Rule**: replay/resume source がすでに `rng_seed` を持つなら、その値を source of truth とする。`--rng-seed` の同時指定は、同値比較で黙認せず明確に reject する
- **Applied**: `cmd/arena-runner` の `--record-input` / `--snapshot-input` と `--rng-seed` の組み合わせ、`docs/specs/platform.md` の replay/debug 契約

## [2026-05-12] deterministic regression golden は code 内 struct に埋め込まない

- **Mistake**: deterministic regression の expected 値を test code 内の Go struct と `reflect.DeepEqual` に置いた
- **Pattern**: golden の更新理由と差分確認導線を code 側に閉じると、PR review と AI-assisted 更新時の境界が曖昧になる
- **Rule**: deterministic regression golden は review しやすい外部 JSON file に置き、test は rerun determinism と golden parity を機械的に比較する
- **Applied**: `ai-arena/e2e/*` の deterministic regression test、特に dungeon の normalized result golden

## [2026-05-18] ai-arena の spec で game code を repo 内既定に見せない

- **Mistake**: `docs/specs/game-master.md` に external repo 配置を例外扱いする補足を書き、ai-arena repo 内に game 実装が置かれているのが通常であるかのような含みを残した
- **Pattern**: 移行中の一時的な repo 配置を基準に spec を書き、platform 契約の通常前提と bootstrap / e2e 用の例外配置を分離できていない
- **Rule**: ai-arena の platform spec では、game master / sidecar は repo 外実装が通常前提として読める書き方にする。ai-arena repo 内に残る実装や fixture は e2e・移行・比較元の都合である場合だけ個別に書き、external repo をわざわざ例外扱いしない
- **Applied**: `docs/specs/game-master.md` の sidecar / transport 記述、今後の ai-arena 側 external game repo 関連 spec 全般

## [2026-05-18] external repo 移行後の golden 更新理由は採用 version change で書く

- **Mistake**: external repo 移行後の deterministic golden 更新理由を、ai-arena repo 内で platform 実装を直接直した場合だけで考えかけた
- **Pattern**: host platform を consumer repo から versioned dependency として取り込む段階に移っているのに、golden 更新理由を mono repo 内の直接修正前提でしか表現しない
- **Rule**: external repo が ai-arena runner / platform を host として使う段階では、golden 更新理由の 1 つを「consumer repo が意図的に採用する ai-arena version change」として書く。direct code edit と import version update を混同しない
- **Applied**: `docs/specs/platform.md` と `docs/specs/dungeon-game.md` の deterministic golden 運用、今後の external repo tagged import verification 全般

## [2026-05-18] game repo 完全移管時は ai-arena 側 game spec を移行メモ化しない

- **Mistake**: dungeon game の repo 外移管後も、`docs/specs/dungeon-game.md` を ownership note として残しつつ、same-golden parity だけで ai-arena 側削除 gate を閉じられる前提で考えかけた
- **Pattern**: historical context を spec として温存しようとして spec-code parity を崩し、削除前に移し切るべき verification path の列挙も不足する
- **Rule**: game 実装と game 固有 verification 資産が ai-arena から完全に外へ出る段階では、その game 専用 spec file は ai-arena から削除する。削除 gate は canonical golden だけでなく、その game の fixture bot、WASM/Rust AI player、CI coverage まで移行完了を列挙してから閉じる
- **Applied**: `0041-dungeon-external-repo-migration-03-ai-arena-removal.md`、`0042-dungeon-external-repo-removal-gate-verification.md`、今後の external game repo removal plan 全般

## [2026-05-18] platform spec が consumer repo の所有物に許可を出す書き方をしない

- **Mistake**: game 固有 verification asset について、`consumer repo 側で持ってよい` のように ai-arena が許可を与える書き方を spec に入れた
- **Pattern**: platform repo が責務境界を説明する場面で、非責務領域を「外部がやってよいこと」と表現してしまい、ownership と権限の境界を曖昧にする
- **Rule**: ai-arena の spec では、platform の責務を明示し、非責務領域は `ai-arena が規定しない` と書く。consumer repo や game 開発側の所有物に対して、platform が許可を与える文型を使わない
- **Applied**: `docs/specs/game-master.md`、`docs/specs/platform.md`、今後の external repo ownership / verification asset 記述全般

## [2026-05-18] ai-arena spec を consumer 向け開発ガイド化しない

- **Mistake**: `docs/specs/platform.md` と `docs/specs/game-master.md` に、ai-arena が提供する契約そのものではなく、external repo 側の開発・ownership・verification 運用まで混ぜて書きかけた
- **Pattern**: platform spec が説明すべき「提供する外形契約」と、別 repo の利用者向けガイドや migration 文脈を同じ文書で扱ってしまう
- **Rule**: ai-arena の spec は、platform / runner / SDK が何を提供し、どの共通契約を固定するかだけを簡潔に書く。external game repo の開発方法、asset 配置、ownership、verification 運用は別の guide / plan / migration note に分離する
- **Applied**: `docs/specs/platform.md`、`docs/specs/game-master.md`、今後の external repo 前提の spec wording 全般

## [2026-05-23] 先行 plan の一時ガードは後続 plan に具体化して残す

- **Mistake**: `0052` の fresh-run only 実装で fail-fast guard と未対応入口を追加しながら、その具体的な制約を `0054` にすぐ反映しないまま進めかけた
- **Pattern**: first milestone と follow-up を分けるとき、code/spec に入れた暫定制約は残るのに、後続 plan には抽象的な目的しか残らず、次の実装者が「どの guard を何で置き換えるか」を再発見する必要が出る
- **Rule**: 現在の execution plan が後続 plan を block/inform しており、そこで一時的な fail-fast guard・未対応 API・暫定 precedence を導入した場合は、その具体名と解除条件を同じ branch で後続 plan に追記する
- **Applied**: `docs/exec-plan/done/0052-external-gamemaster-manifest-registration-01-dev-runner-overlay.md` と `docs/exec-plan/todo/0054-external-gamemaster-manifest-registration-03-resume-replay-follow-up.md`、今後の milestone 分割された plan chain 全般

## [2026-05-24] advisory review 指摘は queue/orchestration 境界の不整合を先に潰す

- **Mistake**: worker dispatch 実装を PR 化した時点で、runner が返す `match_id` の整合性、summary が参照する artifact の実在、runtime 起動への worker `ctx` 伝播といった orchestration 境界の整合性チェックが抜けていた
- **Pattern**: happy-path の queue lifecycle 完了に意識が寄ると、runner/service 境界の「参照先が実在するか」「返却値が submission と一致するか」「cancel が start path に届くか」を後回しにしやすい
- **Rule**: queue/worker/runner/persist をまたぐ新規 orchestration を追加するときは、少なくとも `returned identifiers match submission`、`summary references only persisted artifacts`、`worker ctx reaches startup path` の3点を初回 PR 前に明示確認し、足りなければ test で固定する
- **Applied**: `internal/platform/service/worker.go`、`internal/platform/service/worker_local.go`、`internal/platform/service/integration_test.go`、今後の service-side orchestration 実装全般

## [2026-05-24] runtime kind と admission tier を同じ軸で書かない

- **Mistake**: external game master の official policy を詰める場面で、`local-subprocess` / `wasm-wasi` / `future-external-adapter` の runtime kind と、`built-in` / `sandboxed submission` / `external adapter` の ownership・運用 tier を混ぜて考えかけた
- **Pattern**: execution topology と admission ownership を 1 つの分類に潰すと、`built-in` 化すべき route と sandboxed runtime を優先すべき route が spec 上でぶれる
- **Rule**: official external game master policy を書くときは、まず admission tier を分け、その中で許可する runtime kind を決める。`docker` は候補として残しても、未サポートなら候補のまま明記し、暗黙に support 済みのように書かない
- **Applied**: `docs/specs/platform-game-registry.md`、`docs/specs/platform.md`、`docs/specs/game-master.md`、`docs/design-decisions/adr.md`、今後の runtime/admission policy 記述全般

## [2026-05-28] production storage target と local/CI harness を同じ lane とみなさない

- **Mistake**: durable store plan の infra target を production の `Neon Postgres` 前提で読んだまま進めると、local contributor verification と CI で何を立てるかが実装前に明文化されない
- **Pattern**: deploy target を決めた時点で development harness まで自明だと扱い、spec と code は進むのに local/CI の再現可能な backend 導入手順が後追いになる
- **Rule**: production 向けの storage target が決まっている plan でも、local/CI で同じ contract をどう検証するかを同じ branch で固定する。spec には contract だけを書き、Docker compose や CI service container などの harness は `docs/development/` と workflow に分離して残す
- **Applied**: `0056-platform-online-foundation-02-01-durable-store-and-write-model`、今後の durable backend / managed service 導入を伴う execution plan 全般

## [2026-05-28] docs-only follow-up で issue 追記するだけなら quality gate を回し直さない

- **Mistake**: CI failure を `docs/issues/` に記録して PR へ含めるだけの follow-up でも、直前の code verification と同じ感覚で local test / lint を再実行し続けかねない
- **Pattern**: 「PR に追加 commit を積む」ことだけを見て、変更内容が docs-only かどうかを quality gate 判断に反映できていない
- **Rule**: failure note や issue logging だけの docs-only follow-up では、対象 code を変えていない限り local test / lint を回し直さない。必要なら既存の verification 結果を保持したまま docs diff だけ commit する
- **Applied**: `0056` PR follow-up での flaky CI issue 追記、今後の docs/issues 追加だけを行う review follow-up 全般

## [2026-05-29] DB が責務に入る read/write path は Postgres lane で固定する

- **Mistake**: `0057` の query read-path 追加で、durable queue/read model の主要検証を `InMemoryQueueStore` ベースの test に置いたまま PR を出しかけた
- **Pattern**: interface 越しに同じ API を叩けると、storage-specific behavior を確認すべき責務まで generic in-memory test で十分だと扱ってしまう
- **Rule**: ai-arena の service で DB が責務に入る write/read/query path は、少なくとも 1 本は `AI_ARENA_PG_TEST_DSN` を使う Docker/Postgres lane で検証する。逆に file-backed / in-memory lane だけが責務の path には Docker を持ち込まない
- **Applied**: `internal/platform/service/query_test.go`、`internal/platform/service/store_postgres_test.go`、今後の durable queue / read model / operator query 実装全般

## [2026-05-31] user の言語指定は途中でも即時反映する

- **Mistake**: 実装作業中の途中報告を英語で続けたあと、user から「結果は日本語で」と修正されて初めて切り替える状態になった
- **Pattern**: workflow 実行と verification に意識が寄ると、会話の出力言語もその時点の repo 既定や直前の自分の文脈で惰性運転しやすい
- **Rule**: user が出力言語を明示したら、その時点以降の commentary/final は即時その言語へ切り替える。repo の docs 言語方針や直前の返答言語を優先しない
- **Applied**: この workspace での execute-task handoff、review follow-up、verification 結果報告全般

## [2026-05-31] spec には現在の契約だけを書き、plan の達成文脈を混ぜない

- **Mistake**: spec に「follow-up 実装後は」のような plan 完了前提の文言を入れ、現在の契約説明ではなく実装段階の文脈を持ち込んだ
- **Pattern**: 直前の exec-plan の目的や実装順序を頭に置いたまま spec を書くと、最終状態の契約だけを簡潔に書くべき箇所に rollout 文脈が混ざりやすい
- **Rule**: spec では最新の契約を現在形で直接書く。`この plan で`、`follow-up 実装後`、`初回は` のような実装順序・移行段階・milestone 文脈は plan/issue に残し、spec 本文には持ち込まない
- **Applied**: `docs/specs/platform-game-registry.md` を含む ai-arena の spec 更新全般、今後の exec-plan 実装に伴う spec 記述全般

## [2026-06-03] README を API カタログ化しない

- **Mistake**: `arena-service` の README 追記で、起動導線だけで足りる場面なのに個別 endpoint 一覧まで書いてしまった
- **Pattern**: ローカル起動方法を書くタスクで、変化しやすい API surface の詳細を README に抱え込んでしまい、spec と README の二重管理を増やす
- **Rule**: README の runtime/how-to 節は起動導線と確認手順に留める。route 一覧や payload contract のような変化しやすい API detail は `docs/specs/` を正本にし、README へ重複列挙しない
- **Applied**: `README.md` の `arena-service` / `operator-ui` ローカル起動案内、今後の local runbook 追記全般

## [2026-06-03] infra 前提のローカル確認手順は deploy-shaped lane を主導線にする

- **Mistake**: `arena-service` の README で、infra deploy 前の人間向け確認手順なのに in-memory queue 起動を主導線に置き、`ARENA_SERVICE_POSTGRES_DSN` の取得/用意方法も省いた
- **Pattern**: 実装者にとって最短の起動経路を、そのまま reviewer/operator 向けの確認手順に流用してしまい、durable backend を含む本命 lane の準備手順が欠ける
- **Rule**: infra 導入前のローカル確認手順を書くときは、deploy-shaped lane を主導線に置く。managed service 相当をローカル harness で置き換える場合は、起動、schema apply、DSN、停止まで README から辿れるように書く。in-memory や lightweight lane は補助導線として扱う
- **Applied**: `README.md` の `arena-service` ローカル起動案内、今後の Postgres/R2/Pages を伴う local verification runbook 全般

## [2026-06-03] local frontend 検証手順では backend health check と dev proxy を先に固める

- **Mistake**: `operator-ui` のローカル確認導線で backend 起動確認の 1 ステップを置かず、frontend から直接 `127.0.0.1:10000` へ fetch させたままにした
- **Pattern**: local frontend の確認手順を書くとき、backend が未起動・別 port・一時再起動中の状態を吸収する dev proxy / health check を後回しにして、connection error をそのまま UI に露出させる
- **Rule**: local frontend と local API を組み合わせる runbook では、少なくとも 1 つの backend health check と、同一 origin で確認できる dev proxy か明示的な base URL 指定手順を先に固定する
- **Applied**: `operator-ui/vite.config.ts` の local proxy、`README.md` の `healthz` 確認と `pnpm run dev` 導線、今後の local UI verification 全般

## [2026-06-04] browser harness の plan では regression lane と実環境調査 lane を分けて書く

- **Mistake**: `operator-ui` の Playwright foundation を計画・実装するとき、deterministic regression lane の整備をそのまま browser harness 全体の到達点として扱い、user が期待していた real local `arena-service` + `operator-ui` の調査/証跡 lane を plan に切り出さなかった
- **Pattern**: Playwright や browser automation を導入すると、「壊れにくい fixture regression」を先に作る判断自体は妥当でも、実装確認・調査・screenshot capture の実運用 lane を同じ言葉で吸収したつもりになりやすい
- **Rule**: UI/browser harness を計画するときは、少なくとも `deterministic regression lane` と `real local inspection/capture lane` を分けて要件確認する。user が「人間の代わりの確認」「調査の目と手」「PR 用 screenshot」を期待しているなら、fixture lane だけで満たしたと解釈しない
- **Applied**: `0068` / `0069` 後続の operator UI browser verification planning、今後の ai-arena local browser harness / Playwright / MCP 導入全般
