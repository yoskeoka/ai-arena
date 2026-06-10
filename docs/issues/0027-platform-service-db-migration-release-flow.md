# platform-service-db-migration-release-flow

## Summary

staging backend は `service_queue_records` table 未作成のまま起動しようとして失敗し、
release flow 上は Pages deploy が進んでも Render backend が新しい backend code を反映できなかった。
その結果、remote operator UI からは CORS error として観測された。
staging DB への手動 migration 適用で当座の起動失敗は解消したが、
schema change を release workflow へ組み込む canonical lane は未整備のまま残っている。

## Impact

- `online-release-staging` が `success` でも backend 起動失敗を見逃しうる
- schema change を伴う PR が merge/tag されても、stg/prod DB schema apply の canonical lane がない
- production release 前に同じ migration/deploy 順序を rehearsal できない

## Evidence

- staging backend は `OPTIONS /api/v1/preset-matches` に `405 Method Not Allowed` を返し、
  expected CORS preflight behavior と一致しない
- Render deploy log では `service_queue_records` relation が存在せず `make render-start` が exit 2 で失敗している
- current repo contract では runtime startup DDL は禁止され、schema apply は service 起動前提になっている

## Desired Outcome

- schema change PR は desired schema SQL と generated migration SQL を同じ PR に含める
- staging release は DB migration を先に apply してから backend deploy / verify へ進む
- production release も同じ順序を踏み、staging で rehearsal できる
- migration 用 direct/admin DSN と runtime 用 pooled DSN の secret contract を分離する
