# local-invite-result-summary
**実行**: `/execute-task` でこの plan を実装する。

## Objective

local operator auth / verification 手順で completed detail を開いたとき、
`result-summary.json` の object 欠損が API 全体の error になり、
operator UI が結果確認まで進めなくなる問題を解消する。

同時に、invite token 発行しかしていない helper を
`local-dummy-fixture` という誤解しやすい名前で exposed している状態を整理し、
manual local auth 導線の command 名と実挙動を一致させる。

## Context

- local manual auth check では `make up` / `make migrate` / `make local-dummy-fixture` のあと
  backend / frontend を別起動して確認する運用がある
- current `local-dummy-fixture` は completed run fixture を投入せず、
  operator signup invite URL を返すだけである
- current S3 artifact reader は `NoSuchKey` を `os.ErrNotExist` へ正規化しておらず、
  `result_summary_path` だけ残って object が欠けた detail request が hard failure になる
- current operator UI は decoded `result_summary` を必須前提で表示しており、
  API が欠損許容になっても panel 側で graceful degradation が必要になる

## Scope

- local object storage の `NoSuchKey` を detail request failure ではなく
  missing summary として扱う contract を fixed する
- operator UI detail panel が decoded summary 欠損時も artifact / replay inputs を表示できるようにする
- local invite command を `local-invite-url` として分離し、
  `local-dummy-fixture` は legacy alias へ落とす
- local auth / operator verification の doc と spec を現実の導線へ合わせる

この plan では以下を扱わない。

- completed run fixture を新たに自動投入する local helper の追加
- SeaweedFS object 欠損そのものの root-cause 調査
- broader operator verification lane の再設計

## Spec Changes

- `docs/specs/platform-service-read-model.md`
  - `result_summary_path` が残っていて object backend 側で object が欠損しても、
    detail view 全体は失敗せず decoded `result_summary` だけ optional 欠損にできることを明記する
- `docs/specs/platform-service-operator-ui.md`
  - summary unavailable でも detail panel 全体は error にせず、
    artifact access / replay inputs を継続表示する contract を明記する
- `docs/development/operator-ui-local-verification.md`
  - local invite 発行の正本 command を `make local-invite-url` に更新し、
    `local-dummy-fixture` は legacy alias として扱う

## Expected Code Changes

- `internal/platform/service/artifact_s3.go`
  - S3 `NoSuchKey` を `os.ErrNotExist` へ正規化する
- `internal/platform/service/artifact_s3_test.go`
  - missing object が `os.ErrNotExist` になる回帰テストを追加する
- `operator-ui/src/routes/operator/CompletedDetailPanel.tsx`
  - decoded summary 欠損時の fallback 表示を追加する
- `tools/dev/local-invite-url.sh`
  - local invite URL 発行専用 helper を新設する
- `tools/dev/local-dummy-fixture.sh`
  - invite helper への legacy alias に置き換える
- `Makefile`
  - `local-invite-url` target を専用 helper へ向ける

## Sub-tasks

- [x] missing S3 object を summary unavailable へ正規化する backend change を入れる
- [x] decoded summary 欠損時の operator UI fallback を追加する
- [x] local invite helper 名を実挙動に合わせて分離する
- [x] spec / development doc を更新する

## Verification

- `go test ./internal/platform/service -run 'TestS3ArtifactStorePutReadAndPresign|TestDefaultArtifactReaderSupportsS3Locator|TestS3ArtifactStoreReadLocatorMissingObjectReturnsNotExist' -count=1`
- `make lint`
- `make test`
- `pnpm_config_store_dir=/tmp/.pnpm-store pnpm build` in `operator-ui/`

## Risks and Mitigations

- object 欠損を黙殺しすぎると本当の persistence 失敗を見逃しやすい
  - mitigation:
    detail request 全体ではなく decoded summary だけ optional にし、
    locator と artifact access metadata は返し続ける
- local helper 名変更で既存手順が壊れる
  - mitigation:
    `local-dummy-fixture` は alias として残し、doc だけ正本を差し替える

## Design Decisions

- local manual auth 導線では invite helper と completed fixture helper を同一視しない
- summary object の欠損は operator read model の degraded-but-readable case として扱い、
  API / UI の両方で graceful degradation させる
