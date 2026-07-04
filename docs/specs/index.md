# Spec Index

`docs/specs/` の本文は、`ai-arena` の observable behavior、責務境界、topology を説明する。
通信仕様の field inventory や wire format の正本は、ここで案内する型付き source を読む。

## 読み方の原則

- process 間、server 間、frontend-backend 間、CLI-backend 間の通信仕様を確認するときは、
  Markdown の列挙より先に typed source of truth を読む
- `docs/specs/*.md` は振る舞い、責務境界、運用上の前提を補足する companion 文書として読む
- 同じ wire contract が TypeSpec または型付き Go code で定義されている場合、
  request / response field の正本はそちらに置く

## HTTP / Browser / CLI API

operator-facing HTTP contract は TypeSpec を正本とする。

- project entrypoint
  - `typespec/main.tsp`
- operator route definitions
  - `typespec/namespaces/operator/api.tsp`
  - `typespec/namespaces/operator/auth.tsp`
  - `typespec/namespaces/operator/health.tsp`
- shared schema
  - `typespec/namespaces/shared.tsp`
- emitted artifacts
  - `typespec/generated/openapi/operator/openapi.json`
  - `operator-ui/src/generated/operator-api/`

関連する behavioral companion:

- `docs/specs/platform-service-operator-ui.md`
- `docs/specs/platform-product-auth.md`
- `docs/specs/platform-service-general-submission.md`
- `docs/specs/platform-service-match-request-scheduling.md`
- `docs/specs/platform-service-read-model.md`

## AI Runtime / Game Master JSON-RPC

`ai-arena` 固有の process 間通信は、Markdown に field 一覧を再掲せず、
次の型付き code を正本とする。

- JSON-RPC 2.0 envelope と NDJSON framing helper
  - `gamemaster/protocol.go`
  - `internal/platform/protocol/protocol.go`
- method 名と request / response payload 型
  - `gamemaster/types.go`
- platform 内で共有する alias contract
  - `internal/platform/contract/types.go`
  - `internal/platform/contract/snapshots.go`

関連する behavioral companion:

- `docs/specs/platform.md`
- `docs/specs/platform-common-contract.md`
- `docs/specs/ai-runtime.md`
- `docs/specs/game-master.md`

## Future Rule

新しい transport や API family を追加するときも、
`docs/specs/*.md` に field inventory を増やすのではなく、
まず typed source of truth を決めてこの index に追加する。
