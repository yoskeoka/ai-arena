# operator-registration-and-bot-cleanup

## Summary

operator が unneeded game registration、bot revision、および content-addressed game/AI bundle を安全に整理する
運営 surface がない。現在は game/bot admission が仕様適合なら増やせる一方、bot の `retire` は identity/revision/
artifact bytes を削除せず、bundle store も delete capability を公開していない。

## Required Boundary

- `competition_scopes`、`game_releases`、`ai_bots`、`ai_submission_revisions`、queue/match/ranking/replay と
  immutable artifact digest の参照グラフを調査する。
- active scope、bot revision、queued/running/completed run、retry/rerun/replay、ranking/audit が参照する DB row または
  bundle bytes は削除しない。
- 安全に unreferenced と証明できる registration/bot/revision/bundle だけを、dry-run/preflight と idempotent な
  実行結果を伴って整理できるようにする。filesystem と R2 の partial failure は復旧可能にする。
- operator UI の運営 menu に置くが、`admin` role、role schema/migration、role-specific authorization policy は
  導入しない。既存 operator surface 以外のアクセス制限を本 issue の理由に追加しない。

## Open Design Decision

hard delete と logical retirement の対象を、referential-integrity audit なしに決めてはならない。既存の
`retire` は owner-owned bot の logical state transition であり、運営による cross-owner cleanup や bundle deletion
の代替ではない。先に dependency inventory と deletion/recovery contract を設計し、実装 plan をその結果へ分割する。

