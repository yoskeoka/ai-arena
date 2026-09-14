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
