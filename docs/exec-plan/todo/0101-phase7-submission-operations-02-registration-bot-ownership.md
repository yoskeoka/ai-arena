# phase7-submission-operations-02-registration-bot-ownership
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective

validated game/AI bundle の上に、durable game registration、competition scope、owner 付き AI bot、
immutable AI submission revision を成立させる。1 account が
`game_id + game major + ruleset` ごとに manifest 所有の上限 N まで active bot を持てること、
bot artifact を更新しても bot identity と ranking identity が分裂せず N を余計に消費しないことを
completion boundary とする。

## Domain Contract

- immutable game release は exact version と game artifact digest を持つ。
- stable competition scope は `game_id + game_version_major + ruleset_version` で識別する。
- scope は active game release、exact `player_count`、`max_active_bots_per_owner` と ruleset budget を持つ。
- game upload/register/activate は operator-only とする。
- `AI bot` は stable `bot_id`、owner account、scope、user-visible `bot_name`、`active|retired` を持つ。
- `AI submission` は bot に属する immutable revision で、artifact ID/digest、validation state、created time を持つ。
- bot の active revision は 0 または 1 件。new revision は N を消費しない。
- N は active bot identity 数を数え、scope/account 単位で transactionally enforce する。
- bot name は owner + scope 内で normalize 後に一意とする。
- retire は new match selection から除外するが、過去 run/ranking identity を削除しない。
- first UI は authenticated internal surface のままとし、acting account を owner とする。
  operator の代理 submit / ownership transfer / public self-service portal は後続へ送る。

Game form と AI form は technical manifest field や arbitrary artifact ref を持たない。
Game は uploaded game artifact を activate し、AI は scope、bot name、uploaded AI artifact、
new bot / existing bot revision の control-plane choice だけを送る。

## Existing Implementation References

- `docs/specs/platform-service-general-submission.md`
  - current flat game/AI entity and deferred DB/upload/ownership, lines 37-109, 129-137
- `internal/platform/service/general.go`
  - current records, in-memory stores, registration methods, lines 45-83, 107-166, 274-317
- `cmd/arena-service/main.go`
  - current in-memory general/request store wiring, lines 282-290
- `internal/platform/service/auth.go`
  - authenticated account identity, lines 73-88
- `internal/platform/service/postgres/schema/03_account_roles.sql`
  - durable account FK target, lines 1-6
- `operator-ui/src/routes/operator/GamesPage.tsx`
  - current metadata-only create form
- `operator-ui/src/routes/operator/SubmissionsPage.tsx`
  - current path-based AI create form, lines 69-98
- `typespec/namespaces/shared.tsp`
  - current flat registration/submission wire models, lines 119-156

## Code Change Map

- `docs/specs/platform-service-general-submission.md` (MODIFY)
  - game release/scope/bot/submission revision/ownership/quota/retirement behavior
- `docs/specs/platform-product-auth.md` (MODIFY)
  - operator-only game activation and authenticated own-bot operations
- `typespec/namespaces/shared.tsp` (MODIFY)
- `typespec/namespaces/operator/api.tsp` (MODIFY)
- generated OpenAPI/operator client (MODIFY)
  - game release/scope, bot/revision, filtered list, retire/activate routes
- `internal/platform/service/postgres/schema/` (NEW/MODIFY)
- `internal/platform/service/postgres/migrations/` (NEW)
- `internal/platform/service/postgres/query.sql` and generated sqlc (MODIFY)
  - artifacts, game releases/scopes, bots, AI revisions, active revision and ownership relations
- `internal/platform/service/general.go` and dedicated stores/services (MODIFY/NEW)
  - replace online in-memory durability, enforce ownership/quota/lifecycle
- `internal/platform/service/http.go` and auth middleware (MODIFY)
  - principal-aware service calls and operation-specific authorization
- `cmd/arena-service/main.go` (MODIFY)
  - Postgres store wiring; no process-local source of truth in Postgres mode
- `operator-ui/src/routes/operator/GamesPage.tsx` (MODIFY)
- `operator-ui/src/routes/operator/SubmissionsPage.tsx` (MODIFY)
- shared UI/API adapter and Playwright fixtures/tests (MODIFY)
  - upload selection, scope/bot management, validation/retirement states

## Spec Changes

- `game_registration_id = game + major` の current ambiguity を解消し、ruleset を含む stable scope と
  exact game release を分離する。
- `display_name` だけの flat AI submission を stable named bot + immutable revision へ置き換える。
- concurrent create/revision/retire の quota and uniqueness semantics を固定する。
- Postgres mode で registration/bot/submission が restart 後も残ることを外形契約にする。

## Sub-tasks

- [ ] game release/scope と bot/submission revision schema を spec-first で固定する。
- [ ] Atlas migration、SQL source、sqlc query、Postgres stores を追加する。
- [ ] principal-aware authorization と transactional N/name uniqueness enforcement を追加する。
- [ ] game activation、bot create/revise/retire/list の TypeSpec/handlers を実装する。
- [ ] Games/Submissions pages を artifact upload result と domain entities へ接続する。
- [ ] restart durability、multi-account quota、revision-not-consuming-slot、retired visibility を検証する。

## Dependencies and Parallelism

- depends on: `0100-phase7-submission-operations-01-artifact-bundle-and-wasm-game-runtime.md`
- [parallel] Postgres schema/store と frontend interaction model は TypeSpec/domain contract 固定後に並行できる。
- blocks: `0102-phase7-submission-operations-03-match-composer-and-queue.md`

## Verification

- schema/migration/sqlc parity and Postgres integration tests
- two accounts and N-boundary concurrency tests
- same bot revision does not consume another slot; new bot N+1 is rejected
- retired bot is excluded from active list but remains addressable historically
- service restart preserves game/scope/bot/revision state
- auth-enabled Playwright covers game activation and named bot create/revise/retire
- TypeSpec/generation, Go tests, operator-ui build/browser lanes, workflow lint

## Risks and Mitigations

- bot と submission を同一 entity にすると quota/ranking が artifact update ごとに分裂する
  - mitigation: stable bot と immutable revision を分ける
- concurrent submit で N を超える
  - mitigation: scope/account lock または同等の serializable transaction と boundary tests
- scope と exact game version を同一 key にすると compatible patch rollout が困難になる
  - mitigation: stable major+ruleset scope と active exact release/digest を分ける

## Design Decisions

- N は active bot identity を数え、artifact revision 数は数えない。
- Reversi 初回 scope の N は game manifest 側で 3 とし、platform に hardcode しない。
- historical entities は immutable identity を維持し、retire で new selection だけを停止する。
