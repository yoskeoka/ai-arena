# local-object-storage-harness-for-platform-service

## Summary

`README.md` と現在の local verification 導線は、
`arena-service` の durable metadata backend として Postgres を立てる手順までは持っているが、
artifact backend については local filesystem 依存のままである。

Phase 6 の first landing shape は `Neon Postgres + Cloudflare R2 + Render + Cloudflare Pages` であり、
infra deploy 前の local confirmation でも、artifact access を object storage lane に寄せた再現性を上げたい。
現状の docs では `ARENA_SERVICE_ARTIFACT_BACKEND=r2` 相当の lane を
local でどう立ち上げ、どの S3-compatible endpoint をどう与えるかが未整理である。

## Context

- current local startup guide:
  `README.md`
- deploy/provider inventory:
  `docs/development/platform-service-online-deploy.md`
- current Postgres harness:
  `docs/development/platform-service-postgres.md`
- first landing provider decision:
  `docs/specs/platform-service-skeleton.md`

## Impact

- infra deploy 前の local verification が metadata lane だけ production-shape に寄り、
  artifact lane は file-backed のままになる
- delegated artifact access と locator handling の問題が local で十分に炙れない
- reviewer/operator 向けの「ローカルでも R2 相当を含めて確かめる」導線が弱い

## Proposed Solution

- local S3-compatible harness を 1 つ repo-local の主導線として定める
  - 例:
    MinIO、LocalStack、SeaweedFS S3 gateway など
- 少なくとも次を `docs/development/` と README から辿れる形で固定する
  - 起動コマンド
  - bucket 作成手順
  - `ARENA_SERVICE_ARTIFACT_BACKEND`
  - endpoint / access key / secret key の local default
  - local object storage lane での `arena-service` 起動例
  - teardown 手順
- local filesystem lane は残してよいが、deploy-shaped local confirmation の主導線は
  Postgres + S3-compatible object storage の組み合わせとして明記する

## Priority

中。
現状でも feature 実装自体は進められるが、artifact lane の production-shape 確認が弱く、
R2/object-storage 周りの差分が deploy 直前まで遅延しやすい。
