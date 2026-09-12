# Platform Service General Submission 仕様

## 目的

この文書は Phase 7 general operator lane の immutable game release、stable competition
scope、owner 付き AI bot、immutable AI submission revision の durable contract を定義する。
後続の match request / scheduling / ranking はここで定義する identity を参照する。

## Entity Contract

- game release は uploaded game artifact の immutable record であり、exact game version と
  artifact digest を持つ。operator だけが upload 済み artifact を activate できる。
- competition scope は `game_id + game_version_major + ruleset_version` で識別する stable
  identity である。scope は active release、exact player count、manifest 由来の
  `max_active_bots_per_owner`、ruleset budget を持つ。compatible patch release の activation
  は scope identity を変えない。
- AI bot は `bot_id`、owner account、scope、user-visible `bot_name`、`active|retired` を持つ
  stable identity である。
- AI submission revision は bot に属する immutable admission record であり、artifact
  digest、runtime/AI identity、validation state、created time を持つ。
- active bot の active revision は 0 または 1 件である。existing bot の new revision は bot
  identity、ranking identity、quota slot を変えない。

既存 HTTP surface の `game_registration_id` は後方互換のため scope id を指す。field-level
wire contract の正本は `typespec/` とする。

## Validation and Lifecycle

game form は ZIP を upload して admission 成功 response を受け、その response の admitted artifact と
manifest 由来の ruleset のうち operator が選択した一つだけで activate する。client は game ID、game
version、artifact digest、legacy registration ID を手入力または改変して activation してはならない。複数
ruleset を持つ bundle では、選択肢は admission response が返す候補に限定する。AI form は scope、bot name、
new bot / existing bot revision の choice と AI bundle ZIP を受け取る。client は選択した scope と ZIP を
admission へ渡し、現在選択中の scope と file に対応する最新の成功 response の admitted AI artifact identity
だけで bot を create/revise する。client が artifact digest、AI submission ID、runtime/AI identity、または
artifact reference を手入力・改変して bot revision を作成してはならない。file/scope の変更または AI admission
の失敗時は prior admission を無効化し、bot/revision を作成してはならない。

既存の metadata-only game registration request は migration-period の compatibility input として
受け付けてよいが、新規 operator operation の正本ではない。この input から作る legacy scope は
artifact-backed activation を代替しない。

既存の AI submission create/list HTTP surface も、bot/revision identity 導入前の match request を移行または
再現する compatibility input として維持してよい。ただし新規 operator UI はこの legacy surface を create/list
の入口として表示してはならず、artifact reference を要求してはならない。legacy record を参照する既存 match
request の挙動は変えない。

- selected game release と ruleset は admitted immutable artifact から解決可能でなければならない。
- artifact digest を指定する activation は、その digest に対応する admitted game release を exact に解決する。
  この経路では built-in registry の事前登録、game ID の whitelist、または同一 major の latest release
  lookup を eligibility の前提にしてはならない。artifact manifest は game identity、exact version、
  ruleset と runtime/build metadata の source of truth とする。
- artifact-backed activation は、selected digest、manifest、admitted descriptor の game identity、exact version、
  requested ruleset、build mode、builder identity が整合するときだけ保存する。未 admission の digest、
  game 以外の artifact、manifest にない ruleset、または metadata の不整合は同期的に拒否し、scope を保存してはならない。
- artifact-backed scope から作る match submission は、scope が snapshot した selected game artifact identity を
 そのまま引き継がなければならない。match admission は、その identity の admitted descriptor を exact に解決し、
  submission の game ID、exact game version、ruleset と descriptor が一致することを queue record の保存および
  dry-run より前に検証する。未 admission の digest、exact lookup の失敗、game ID / version / ruleset の不一致は
  同期的に拒否し、その場合は version lookup や built-in fallback を行わず、queue record を作成してはならない。
- artifact identity を持たない legacy または local built-in submission は compatibility path として扱う。ただし
  artifact-backed scope の match をこの path に落としてはならず、外部 admission release が選択可能な online
  service ではその release を優先する通常 registry lookup の結果だけを使用する。
- player count と owner quota は selected ruleset manifest 由来でなければならない。
- AI artifact の game id、semver major、ruleset、runtime は target scope と互換でなければならない。
- bot name は owner + scope 内で trim、Unicode case-fold、連続 whitespace の一文字化をした
  normalized value が一意でなければならない。
- scope/account の active bot count、name uniqueness、create/revise/retire 判定は同一 transaction
  で直列化する。new bot は limit 未満だけ成功し、existing bot revision は slot を消費しない。
- retire は new match selection から除外するが、bot/revision と過去 run/ranking reference を
  削除してはならない。

## Authorization and Durability

- game artifact upload/release activation は operator-only とする。
- bot create/revise/retire/list は authenticated internal surface であり、acting account を owner
  とする。operator の代理 submit と ownership transfer は後続へ送る。
- Postgres mode では release、scope、bot、revision、active revision relation は restart 後も残る。
  process-local store は Postgres mode の source of truth にしてはならない。

## Retired Preset Read Compatibility

新規の game、bot、match request は registered scope と admitted bot を明示する general lane
だけで作成する。保存済みの `source=preset` record は read、detail、ranking、replay の互換性のため
保持し、この lane はその record を削除または書換えない。match request / scheduling policy、ranking
aggregate、public self-service portal、asynchronous review はこの文書の範囲外とする。
