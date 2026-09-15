# Public Spectator Contract

## 対象範囲

public spectator resource は、logical match の exported-state-only view を匿名で提供する。
operator の read model とは独立しており、operator artifact locator、delegated URL、credential、private
snapshot、event log、stderr、admitted bundle bytes を公開してはならない。

field-level HTTP contract の正本は `typespec/namespaces/public/api.tsp` とする。この文書は観測可能な
振る舞いと境界だけを定義する。

## versioning と transport

public API family は `/api/v1-alpha/public/` に version を持つ。alpha resource は credential-free、`GET`
only であり、`Access-Control-Allow-Origin: *` を返す。alpha contract は意図的に不安定であるため、breaking
change は path と TypeSpec contract を同時に更新する。将来の stable release は別の versioned route family を
追加し、alpha contract を暗黙に再定義してはならない。payload-local version（monotonic state version と replay
format version）は HTTP API version の代替ではない。

queue、state store、artifact store の障害は not-found と混同してはならない。public HTTP adapter は内部詳細を
出さない generic unavailable response（HTTP 503）へ正規化し、HTTP 404 は discoverable な対象がない場合だけに
用いる。

## visibility と selected run

resource key は `match_id` だけである。client は `run_id` を選べない。official completed run がある match は
その run を選び、それ以外は current active/latest run を選ぶ。promotion は selected run を atomically に切り替える。
read は promotion の途中で official run が 0 件または複数件の状態を観測してはならない。state version は selected
`(match_id, run_id)` namespace 内で単調増加し、selected run の変更は新しい namespace を開始する。

queued と leased match は discoverable ではない。running、persisting、terminal match は discoverable である。
未 publish の state と読めない replay は documented unavailable result で表す。backend error や private storage
detail は観測可能であってはならない。terminal lifecycle の state response は state が unavailable でも polling stop
を示す `retry_after_ms = 0` を返す。

各 public match projection は、selected run が admission 時に固定した participant provenance を返す。participant は
match request の submitted player order と個数を変えず、`player_id`、display name、immutable AI submission/revision
identity だけを含む。同じ selected run では、この sequence と terminal completion timestamp を list、detail、state
の各 resource から不変に観測できる。promotion は selected run 自体を切り替えるため、新たに選ばれた run の sequence と
completion timestamp を返してよい。platform は game 固有の player role、color、人数、または配列順の意味を追加・検証せず、
consumer は配列順と game ruleset を使って解釈する。新規 admission は bot と raw AI submission のどちらの経路でも
この provenance を固定する。歴史的 record に完全な provenance がない場合、platform は現在の operator registry や private
artifact から補完せず participant metadata を省略する。

selected run が terminal artifact の durable persistence に成功して `completed` になったとき、projection は immutable な
completion timestamp を返す。heartbeat、promotion、ranking などの更新時刻をその代わりに使ってはならない。未完了、persistence
前に failed になった run、および completion timestamp を持たない historical record はこの timestamp を省略する。

## match list

`GET /api/v1-alpha/public/matches` は discoverable logical match の selected run だけを返す。query はすべて任意で、
`game_id` は完全一致、`game_version_major` は game semantic version の major の一致、`ruleset_version` は完全一致で
絞り込む。`page` は 1 始まりで既定 `1`、`limit` は既定 `20` かつ `1..100`、`sort` は `completed_at` だけを受け付け既定も
`completed_at`、`sort_order` は `asc` または `desc` で既定は `desc` とする。未知の query、無効な整数、範囲外の page / limit、
正でない major、未対応の sort / sort order は HTTP 400 とする。

completed timestamp を持つ record は timestamp 順に並び、timestamp を持たない record は sort order にかかわらず最後に
置く。同じ timestamp の record と timestamp を持たない record は、sort order にかかわらず `match_id` の昇順で並ぶ。応答は
`items` に加え、filter 後かつ page 前の `total`、要求へ反映した `page` と `limit`、`ceil(total / limit)` の `total_pages`
を含む `pagination` を返す。`total = 0` の `total_pages` は `0` とする。

`available_ruleset_versions` は `game_id` と `game_version_major` filter に合う discoverable selected run から、ruleset filter
と page を適用する前に導出する。値は重複なしの昇順とし、ruleset filter の結果が空でも scope に存在する値を保持する。

## state と replay の境界

latest-state resource は atomically published された exported snapshot だけから構成する。state publication は
match の turn timeout を無制限に延長してはならず、bounded な失敗方針を持つ。client は monotonic version と lifecycle
により stale response を捨て、terminal lifecycle 後に polling を止められる。

terminal replay bytes は game が生成した versioned public replay artifact だけから返す。platform は bounded locator と
metadata を保持してよいが、private artifact を decode、filter、redirect、re-envelope して public replay を導出しては
ならない。read は 1 MiB の上限を超えて materialize してはならず、payload の size と digest を advertised metadata と
照合する。1 MiB 超、欠落、retention、integrity failure、unsupported artifact は同じ unavailable result にする。

admitted WASI game master が valid な terminal public replay を提供するとき、platform は runtime の lifecycle と
resource cleanup をまたいでその replay capability を保持し、既存の replay artifact と metadata の永続化・公開へ渡す。
この保持は game-owned で opaque な payload を変更せず、replay を提供しない game と artifact の integrity または
retention failure は従来どおり unavailable とする。
