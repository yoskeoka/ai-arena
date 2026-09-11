# operator-registration-and-bot-cleanup
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: `docs/issues/0042-operator-registration-and-bot-cleanup.md`

## Objective

registered game、bot、revision、bundle bytes を安全に整理する運営 menu を設計・実装する。これは `admin` role を
定義する task ではない。既存 operator surface 内に置き、role-specific authorization、account role schema、
operator/admin の権限分離は scope 外とする。

本 plan は destructive operation の実装開始前に、referential-integrity inventory と delete/recovery contract を
review する parent plan である。hard delete の可否、referenced artifact の保存期間、DB/R2 partial failure recovery
を未確定のまま実装してはならない。

## Existing References

- `docs/issues/0038-unactivated-game-bundle-retention.md` と `docs/issues/0042-operator-registration-and-bot-cleanup.md`:
  unactivated-only retention と broader operator cleanup の境界。
- `internal/platform/service/postgres/schema/06_game_scopes_bots.sql:1-60`:
  release/scope/bot/revision foreign-key graph。
- `internal/platform/service/bot_ownership_postgres.go:97-104`:
  existing owner-scoped logical bot retirement。
- `internal/platform/service/artifact_bundle_store.go:14-96` と
  `artifact_bundle_s3.go:18-96`: immutable filesystem/R2 bundle storage。現状 delete interface はない。
- `internal/platform/service/request.go`、`worker_local.go`、`ranking.go`、replay packages:
  match/rerun/replay/ranking が digest/registration/bot identity を参照する consumer set。

## Required Design Gate

execution child plan を作る前に、次を evidence 付きで決定する。

1. DB schema/query と artifact locator を横断して、deletable candidate、protected reference、owner/account boundary、
   historical run/replay/ranking dependency を列挙する。
2. UI action と temporary maintenance command の比較を行い、operator menu を canonical execution surface とするか、
   command を first safe migration seam とするかを human と合意する。
3. preflight/dry-run response、explicit confirmation identity、idempotency、DB-first/R2-first ordering、partial failure
   repair/retry、audit evidence、rollback impossibilityを behavioral spec に固定する。
4. hard delete 対象と logical retire 対象を分ける。active/referenced record/blob は候補に含めない。

`N/A - detail required before execution`: 上記 gate が未完了のため、この parent plan 自体は implementation を開始しない。
結果は UI cleanup child plan と必要なら maintenance migration child plan に分割し、それぞれが exact DB/R2 targets と
verification を持つ。

## Black-box Invariants for Child Plans

- cleanup candidate には preflight で reference/protection reason を表示し、protected candidate を mutation しない。
- destructive confirmation が成功した場合だけ、unreferenced DB metadata と corresponding immutable bundle bytes を
  一貫して整理する。filesystem と R2 のどちらでも orphan/partial failure を検知・repair できる。
- normal operator game registration、bot revision、match queue、ranking、replay の behavior を変えない。
- role `admin`、role-based access control、new privilege grants は追加しない。

## Dependencies

- 0121--0123 と独立に discovery は可能だが、preset-related legacy data を cleanup candidate に含める判断は 0123
  の historical-read contract 確定後に行う。
- `0038` の unactivated-only retention は duplicate にせず、candidate/reference model が重なる場合は一つの child
  contract へ統合する。

