# game-release-runtime-default-migration-drift

## Summary

Postgres desired schema と既存 migration chain の diff に、Phase 8 public state と無関係な
`game_releases.runtime_args` / `memory_limit_pages` の `NOT NULL` / default 変更が含まれる。

## Required Boundary

- descriptor runtime metadata の既存 row と admission / registry read path に対する null/default
  compatibility を確認する。
- dedicated migration と regression test を作り、public spectator state migration と混在させない。
- production/staging schema revision と migration history を確認してから apply path を決める。
