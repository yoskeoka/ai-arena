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

game form は manifest の technical field や arbitrary artifact ref を持たず、uploaded game
artifact の activate だけを送る。AI form は scope、bot name、uploaded AI artifact、および new
bot / existing bot revision の choice だけを送る。

- selected game release と ruleset は admitted immutable artifact から解決可能でなければならない。
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

## Preset Queue との関係

preset lane は dedicated queue identity を持たず、必要な scope と bot/revision を materialize
して general lane と同じ immutable artifact identity を参照する。match request / scheduling
policy、ranking aggregate、public self-service portal、asynchronous review はこの文書の範囲外とする。
