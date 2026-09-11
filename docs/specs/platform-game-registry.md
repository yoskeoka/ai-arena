# ゲーム registry 仕様

## 目的

このドキュメントは、AI Arena が複数 game を registered game として扱うためのregistry contract を定義する。

## この spec の責務範囲

この spec が定義するもの:

- registry key
- 永続化できる登録情報と process-local な起動情報の境界
- built-in registry と runner-local overlay descriptor の境界
- 永続化レコード参照、runtime 解決、lookup 公開の責務分離
- registered game が備える起動・再開・replay 入口の最小契約
- game master 接続形態の表現
- lookup 後に行う metadata 確定と compatibility 検証の責務分離
- operator-facing game registration が参照する最小 metadata

この spec が定義しないもの:

- game 固有 payload schema
- game master subprocess / trusted external backend transport の詳細
- game plugin / self-service registration の運用
- consumer repo が official registered game として admission される手順
- 1 つの game registration に参加する AI submission の lifecycle

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

### 同一 major の release 選択

1 つの registry key には、同一互換系列の複数の admitted game release を保持してよい。
release は bundle manifest の exact semantic `game_version` と immutable artifact digest を持つ。

- 通常の registry lookup は、同一 `game_id + game_version major` に属する admitted release のうち、
  semantic version が最大のものを返す。
- 新しい同一-major release の登録は、旧 release を物理削除しない。旧 release は監査、障害調査、
  既存 match が記録した artifact digest の再 materialize のため保持する。
- match の実行時には、通常 lookup が選んだ release の artifact digest を記録する。後からより新しい
  release が登録されても、記録済み match の game master bytes は変わってはならない。
- prerelease を official registry に admission するか、同一 release を再 upload したときの
  idempotency は artifact bundle contract が定める。通常 lookup は admission 済みの release だけを比較する。

online service の通常 lookup では、admission 済みの外部 release と built-in release を同じ候補集合として
比較してはならない。外部 admission tier に対象の `game_id + game_version major` が 1 件以上ある場合は、
その tier の release だけから最大 semantic version を選び、built-in release は shadow される。外部 admission
tier に対象 key がない場合に限り、built-in release tier を fallback として通常どおり解決する。この優先順位は
upload 済み release の選択に適用され、active scope が exact artifact digest を snapshot する契約を置き換えない。

## persisted descriptor record

永続化 backend に保存する registered game metadata は、runtime の function を含まない plain data として扱う。
この spec では、その保存単位を `DescriptorRecord` と呼ぶ。

`DescriptorRecord` は少なくとも以下を持つ。

- `game_id`
- exact semantic `game_version` とそこから導く `game_version_major`
- immutable artifact digest
- `build_mode`
- `builder_id`
- ruleset / build 制約を表す metadata
- WASI runtime args と memory-page limit

release の canonical な一意性は artifact digest と `game_id + exact game_version` で保つ。通常 lookup の
key は `game_id + game_version_major` の composite key とする。
必要なら `registry_key` という論理名を持ってよいが、これは composite key の derived /
denormalized field として扱う。

runner-local な dev overlay は、この persisted `DescriptorRecord` を増やさない。
consumer-supplied manifest から一時的な local-subprocess descriptor を構築する経路は、
runner process 内だけで完結する overlay として扱い、DB / catalog へ official registration を
追加しない。

official registration path は、runner-local overlay path とは別の admission policy に従う。
運営が review して built-in game として取り込む経路、制約付き runtime に載せて platform 管理下で
運用する経路、trusted external backend を official external adapter として登録する経路は区別して扱う。
どの経路でも persisted record 化してよいのは、その tier で admission 済みの registered game だけとする。

admitted WASI bundle は archive 保存と validation の成功後、scope を作成する前に complete な immutable
descriptor metadata を durable store へ保存する。保存失敗は admission の成功として返さない。このとき
archive object が残ることは許容するが、lookup は archive bytes を読んだり補償削除を試みたりしない。
scope は既存の exact admitted release を参照し、scope が新 release へ移っても過去 release record は保持する。

同一 immutable artifact digest の検証済み bundle を再 admission したときは、complete な既存 descriptor を
変更せず idempotent に扱う。過去 schema の都合で WASI runtime args と memory-page limit **だけ**が欠落した
既存 descriptor は、同じ検証済み manifest から得た値で補完してよい。この repair は artifact digest、game
identity、exact version、build mode、builder、ruleset metadata、scope identity を変更してはならない。いずれかの
immutable metadata が一致しない場合、または runtime field 以外も欠落している場合は、既存 release を更新せず
conflict/error とする。DB への推測値の一括 backfill や archive bytes の保存はこの経路に含めない。

operator-facing general lane では、runtime descriptor 自体ではなく、
operator が選択・検証に使う plain-data metadata view を exposed してよい。
この view は少なくとも次を含む。

- `registration_id`
- `game_id`
- `game_version`
- `ruleset_version`
- `build_mode`
- `builder_id`
- `supported_rulesets`

`registration_id` は operator lane の stable identity であり、
default では `game_id + game_version major` から決まる deterministic id にしてよい。
この metadata view は `GameDescriptor` の alias ではない。
operator-facing validation surface が必要とする plain data だけを取り出した view とし、
build/replay function を含めてはならない。

## registry lookup 層

runner / replay から見える registry lookup は、少なくとも以下 3 責務に分ける。

- 永続化された registered game record を key で読む責務
- 読み出した record を process-local な起動情報へ解決する責務
- その 2 つを束ねて runner / replay へ lookup を提供する責務

実装上は `RegistryStore`、`DescriptorResolver`、`Registry` などの名前を使ってよいが、
spec として重要なのは責務分離そのものである。この分割により、永続化 backend は record 読み出し層だけ
差し替えればよく、build/replay function の解決は process 内に閉じ込められる。

runner が dev overlay manifest を受け取る場合でも、責務分離の原則は変えない。

- built-in path:
  - persisted record を読む
  - record を runtime descriptor へ解決する
- runner-local overlay path:
  - consumer-supplied manifest を検証する
  - manifest を local-subprocess runtime descriptor へ解決する

どちらの path でも、runner が後段で扱うのは同じ `GameDescriptor` 抽象である。

official sandboxed artifact の admission registry は、built-in descriptor を名前で引く通常 lookup とは別に、
admission 済み descriptor record を immutable artifact identity で exact に解決できなければならない。
artifact-backed game activation はこの exact path を使う。uploaded descriptor の eligibility は bundle admission と
manifest / descriptor の整合性で決まり、exact identity の lookup 失敗や metadata 不整合時に version lookup、
built-in descriptor、game ID hard-code へ fallback してはならない。

Postgres を configured した operated service では external/admitted tier の metadata を durable store から
毎回読む。通常 key lookup は durable tier に対象 key がない場合だけ built-in へ fallback してよいが、
durable read failure または invalid metadata は lookup failure とし fallback してはならない。exact artifact
lookup は external-only であり、version や built-in へ fallback しない。service restart/redeploy 後も同じ
exact digest を request admission と worker session construction で解決できなければならない。

この failure contract は incomplete durable descriptor にも適用する。通常 lookup と exact artifact lookup は
incomplete row を skip して別 version や built-in を選んではならず、activation と match admission もその error を
返す。usable 状態への復旧は lookup 時の補完ではなく、上記の検証済み同一 artifact の再 admission に限る。

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

サポートする形式:

- `in-process`
- `local-subprocess`
- `future-external-adapter`

1 つの registered game は 1 つの `BuildMode` を持つ。
fixture 検証のために別接続形態も試したい場合は、`echo-count` と `echo-count-subprocess` のように
別 `game_id` の registered game として分ける。

consumer-supplied manifest overlay は、このうち `local-subprocess` だけを現時点ではサポートする。
`future-external-adapter` やその他 runtime kind の一時 overlay は未対応。

official registration で許可する `BuildMode` は次の想定。

- `official built-in`
  - 運営自身が実装した game master
  - 信頼できるパートナーから source 提供を受け、運営が review / CI / release 管理を引き受ける game master
  - persisted record の `BuildMode` は `in-process` でよい
- `official sandboxed submission`
  - source を built-in 化せず、platform 管理下の制約付き runtime で運用する game master
  - 第一候補の runtime kind は `wasm-wasi` とする
  - `local-subprocess` は host 隔離が弱いため、この tier では admission しない
  - `docker` / OCI container は将来候補に留め、現時点では未サポートとする
- `official external adapter`
  - GPU / LLM / 専用 service など、platform 単体では抱えにくい計算資源を必要とする trusted external game backend
  - persisted record の `BuildMode` は `future-external-adapter` とする

`docker` / OCI container を将来 `official sandboxed submission` に加える場合でも、
少なくとも network isolation、filesystem / capability 制限、resource limit、image / dependency scan をadmission 条件として要求する。これらの要件を満たす具体例が出るまでは、未サポート候補として扱う。

operator-facing game registration validation は、少なくとも次を同期的に確認しなければならない。

- `game_id + game_version major` が registry lookup 可能であること
- 要求された `ruleset_version` が descriptor の `supported_rulesets` に含まれること
- exposed する `build_mode` / `builder_id` / `supported_rulesets` が lookup 結果と矛盾しないこと

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

runner-local overlay path でも metadata validation の順序は同じとする。
ただし manifest overlay で起動する game master metadata の source of truth は manifest であり、
CLI flag 群や built-in record が metadata を上書きしてはならない。

## snapshot / history replay 入口

descriptor は replay/debug のために以下を提供する。

- fresh run を game master session に変換する入口
- resume snapshot を game master session に変換する入口
- history と target turn から snapshot を再構築する入口

これにより replay/debug は registry 外へ game 固有 helper を漏らさずに扱う。

online service が persisted match から replay / resume / audit 導線を作る場合も、
service が担うのは `game_id` / `game_version` / `ruleset_version`、player 順序 / `artifact_ref`、
`record` / `snapshot` / `history` / exported/public snapshot locator の join までとする。
その read path が history replay や snapshot resume を実行するときは、
解決済み metadata と source-of-truth `record.json` または derived helper artifact を
runner / registry の既存 build 入口へ渡し、game 固有 rebuild を service 層へ持ち込んではならない。

dev-only manifest overlay も同じ 3 入口を持つ。

- fresh run は manifest metadata と runtime entrypoint から local-subprocess session を起動する
- snapshot resume は同じ manifest metadata / runtime entrypoint に `resume snapshot` を渡して session を起動する
- history replay は manifest metadata を source of truth にした fresh session を起動し、
  persisted `history` / `record.event_log` に記録された action status を turn 境界まで再適用して
  snapshot を再構築する

この overlay path でも service 層や caller が game 固有 rebuild helper を直に持ってはならない。
runner / replay-debug path が扱うのは、manifest から解決した descriptor、artifact から読んだ metadata /
snapshot / history、そして既存の game master session 論理 API だけとする。

## 採用しない案

- constructor registry
  - replay/debug の game 固有入口が registry 外へ漏れやすいため採用しない
- `game_id` 単独 key
  - semver major が互換境界という project 方針を lookup key に反映できないため採用しない
- capability set 中心設計
  - 現時点で必要なのは build/replay 入口と接続形態であり、flag を先に増やしても意味が薄いため採用しない
