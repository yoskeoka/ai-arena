# Platform service object storage harness

online service skeleton の deploy-shaped artifact lane は、
local では `SeaweedFS` を S3-compatible object storage harness として検証する。
production target は `Cloudflare R2` であり、
この文書は repo-local contributor workflow を定義する。

## Command Surface

```sh
make seaweed-up
make seaweed-bootstrap
make seaweed-down
```

`make seaweed-bootstrap` は reset-first で次を行う。

- `SeaweedFS` container 起動
- default bucket `ai-arena-local` の作成
- optional seed object upload の入口準備

default data dir は repo-local ignored path の `.local/seaweed/` とする。
stale object を引き継がない local verification を主導線にするため、
通常は `make seaweed-bootstrap` を毎回使う。

## Default Env Contract

local `SeaweedFS` lane は remote artifact lane と同じ env key を使う。

```text
ARENA_SERVICE_ARTIFACT_BACKEND=r2
ARENA_SERVICE_ARTIFACT_R2_BUCKET=ai-arena-local
ARENA_SERVICE_ARTIFACT_R2_S3_ENDPOINT=http://127.0.0.1:8333
ARENA_SERVICE_ARTIFACT_R2_ACCESS_KEY_ID=admin
ARENA_SERVICE_ARTIFACT_R2_SECRET_ACCESS_KEY=secret
```

`arena-service` は local harness でも `r2` backend 名を使う。
この backend 名は provider 固有挙動ではなく、
S3-compatible object storage lane を表す deploy-shaped contract として扱う。

## Local Verification

起動:

```sh
make postgres-up
make postgres-schema-apply
make seaweed-bootstrap
make render-build
ARENA_SERVICE_POSTGRES_DSN=postgres://arena:arena@127.0.0.1:55432/arena_service?sslmode=disable \
ARENA_SERVICE_ARTIFACT_BACKEND=r2 \
ARENA_SERVICE_ARTIFACT_R2_BUCKET=ai-arena-local \
ARENA_SERVICE_ARTIFACT_R2_S3_ENDPOINT=http://127.0.0.1:8333 \
ARENA_SERVICE_ARTIFACT_R2_ACCESS_KEY_ID=admin \
ARENA_SERVICE_ARTIFACT_R2_SECRET_ACCESS_KEY=secret \
PORT=10000 \
make render-start
```

general bundle regression は registered game と admitted bot を指定した match request を使い、
completed detail の `s3://...` stable locator と delegated `result-summary.json` download を確認する。

停止:

```sh
make seaweed-down
```

## Seed Object Rule

seed object が必要な場合も、
`.local/seaweed/` や container 内 filesystem へ直接 `cp` しない。
必ず S3-compatible API client 経由で投入する。

repo-local helper は `tools/dev/seaweed-bootstrap.sh` を正本にする。
bootstrap helper が Dockerized AWS CLI を使う場合は、AWS 公式が推奨する
`public.ecr.aws/aws-cli/aws-cli:latest` を default image にしてよい。
