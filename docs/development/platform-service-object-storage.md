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
make verify-local-object-storage
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
ARENA_SERVICE_PRESET_CONFIG=./config/platform-service/presets.remote-bootstrap.json \
ARENA_SERVICE_ARTIFACT_BACKEND=r2 \
ARENA_SERVICE_ARTIFACT_R2_BUCKET=ai-arena-local \
ARENA_SERVICE_ARTIFACT_R2_S3_ENDPOINT=http://127.0.0.1:8333 \
ARENA_SERVICE_ARTIFACT_R2_ACCESS_KEY_ID=admin \
ARENA_SERVICE_ARTIFACT_R2_SECRET_ACCESS_KEY=secret \
PORT=10000 \
make render-start
```

この lane では `presets.remote-bootstrap.json` を canonical とし、
`make render-build` が生成する prepared preset executable を使って remote bootstrap shape を再現する。
`presets.example.json` は lightweight contributor example であり、この deploy-shaped lane の canonical config ではない。

verification helper:

```sh
make verify-local-object-storage
```

この helper は少なくとも次を確認する。

- preset queue request が受理される
- completed detail が `s3://...` stable locator を返す
- delegated download URL を取得できる
- その URL から `result-summary.json` を取得できる

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
