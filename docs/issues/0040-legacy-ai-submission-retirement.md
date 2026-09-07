# legacy-ai-submission-retirement

## Summary

`POST/GET /api/v1/ai-submissions` と `artifact_ref` を受ける legacy AI submission model は、bot/revision と
artifact-backed AI bundle admission より前の match request を扱うために残っている。`/operator/submissions` の
新規 registration UI は AI bundle ZIP admission と bot create/revise へ移行するため、legacy create/list panel を
利用しない。

現時点では local/CI browser workflow が legacy endpoint を作成経路として利用する evidence はなく、
artifact-backed service E2E は `RegisterAIBundle` を使う。一方、match request service は legacy
`AISubmissionID` を読み、stored `ArtifactRef` / `ArtifactID` を submitted player に解決できる。そのため UI から
消えることだけを根拠に HTTP endpoint、TypeSpec models、generated client、service/store を削除してはならない。

## Why It Remains Open

`0112-reversi-submission-zip-upload` は、released Reversi AI ZIP を operator UI から admission し、stable bot
identity の create/revision へ進める UX を対象とする。legacy API retirement を同じ change に含めると、historical
match request の read/retry/rerun/replay、production/staging に残る legacy rows、direct service/fixture tests、
TypeSpec consumer compatibility の migration と data-retention contract が加わり、完了境界が拡大する。

## Follow-up Boundary

- staging/production と local durable fixtures に legacy AI submission、legacy match request、legacy artifact ref が
  残るかを、read-only query と exported historical records で確認する。存在する場合は retention 期間、migration
  owner、rollback strategy を定める。
- `MatchRequestService` の legacy participant resolution、retry/rerun/replay、preset/materialization と durable
  Postgres/filesystem/S3 stores を監査し、legacy record への read dependency を列挙する。
- Go unit/integration/E2E、operator browser fixture/CI workflow、CLI/development scripts、external API consumers が
  `/api/v1/ai-submissions`、`AISubmissionRequest`、`artifact_ref` を create/list input として使うかを確認する。
- removal が可能なら、先に black-box migration/retention contract と compatibility sunset date を plan に固定する。
  historical match result/replay を壊さず、bot/revision artifact identity へ安全に backfill または explicit read-only
  compatibility adapter を置く。
- usage が残るなら、legacy endpoint を hidden internal migration-only surface として authorization、observability、
  sunset criterion を定義する。new operator UI を再導入しない。
- TypeSpec endpoint/model、generated client exports、HTTP handler、service/store、tests/fixtures を削除するのは、
  上記 evidence と migration completion を PR で確認した後だけとする。
