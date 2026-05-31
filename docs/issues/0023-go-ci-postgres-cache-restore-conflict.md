# go-ci-postgres-cache-restore-conflict

## Summary

`2026-05-31` の `go-ci` run で、`go-test-postgres` job の Postgres 起動や
`make ... test-postgres` 本体は通過した一方、事前の cache restore で
`/usr/bin/tar: ... Cannot open: File exists` が大量発生した。

これは `internal/platform/runtime` や `internal/platform/session` の test flaky ではなく、
`actions/setup-go` と追加の `actions/cache` が同じ
`/home/runner/go/pkg/mod` / `/home/runner/.cache/go-build` を復元して衝突している
CI workflow 起因の不安定要素とみなす。

## Failure Output

- observed on:
  `2026-05-31`
- workflow run:
  `https://github.com/yoskeoka/ai-arena/actions/runs/26707309959/attempts/1`
- affected job:
  `https://github.com/yoskeoka/ai-arena/actions/runs/26707309959/job/78711063226`
- representative log sequence:

```text
Setup Go
  cache: true
  Cache restored from key: setup-go-Linux-x64-ubuntu24-go-1.26.3-...

Cache Go module and build state
  Cache hit for: Linux-go-...-test-postgres
  /usr/bin/tar -xf .../cache.tzst -P -C /home/runner/work/ai-arena/ai-arena ...
  /usr/bin/tar: ../../../go/pkg/mod/gopkg.in/yaml.v3@v3.0.1/emitterc.go: Cannot open: File exists
  ...
  Failed to restore: "/usr/bin/tar" failed with error: The process '/usr/bin/tar' failed with exit code 2
```

## Why This Is Separate From 0019 / 0020

- `0019-runtime-stream-capture-flake` は
  `internal/platform/runtime.TestStartStreamsAndCapturesStderr` の stream close race
- `0020-session-game-over-status-flake` は
  `internal/platform/session.TestSessionInitTurnTimeoutGameOverAndLateResponse` の
  status assertion race
- 今回は test body 自体は成功しており、失敗点は Go cache 展開時の tar 衝突である

つまり failure domain が test timing ではなく CI cache layer にあるため、
既存 flaky への追記ではなく別 issue で扱うほうが切り分けしやすい。

## Suspected Cause

`.github/workflows/go-ci.yml` の `go-test-postgres` job では、
`actions/setup-go` が `cache: true` のまま Go cache を restore したあとで、
同じ path に対して `actions/cache` が再度 restore を試みている。

その結果、後段の `actions/cache` が既存ファイルを上書きできず
`Cannot open: File exists` を返し、warning annotation を出している可能性が高い。

## Proposed Solution

- `go-ci.yml` の Go cache 管理を 1 系統に寄せる
- 具体案は次のどちらか
  - `actions/setup-go` の cache を無効化し、明示 `actions/cache` だけを使う
  - 追加の `actions/cache` を削除し、`actions/setup-go` 側の cache に寄せる
- 変更後、`go-test-file-backed` / `go-test-postgres` / `go-lint` の各 job で
  cache restore warning が消えることを確認する

## Priority

現時点では workflow は green だが、PR timeline に warning/noise を残し、
本物の flaky test と見分けづらくする。Go CI cache 方針の整理として早めに潰したい。
