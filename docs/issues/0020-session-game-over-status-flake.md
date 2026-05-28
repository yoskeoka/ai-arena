# session-game-over-status-flake

## Summary

PR `#213` の CI follow-up 中、`go-ci / go-test` が 2 回とも
`internal/platform/session.TestSessionInitTurnTimeoutGameOverAndLateResponse`
で落ちた。今回の `0056` durable store 差分は `internal/platform/session`
を触っておらず、local stress rerun でも再現していないため、現時点では
unrelated flaky failure とみなす。

## Failure Output

- observed on: `2026-05-28`
- failing workflow: `go-ci / go-test`
- failing job URL:
  `https://github.com/yoskeoka/ai-arena/actions/runs/26542236482/job/78186160631`
- rerun job URL:
  `https://github.com/yoskeoka/ai-arena/actions/runs/26542236482/job/78186441292`
- representative assertion:

```text
--- FAIL: TestSessionInitTurnTimeoutGameOverAndLateResponse (0.23s)
    session_test.go:85: GameOver status = "no_action", want accepted
FAIL
FAIL    github.com/yoskeoka/ai-arena/internal/platform/session    10.303s
```

## Local Recheck

- command:
  `GOPATH=/tmp/ai-arena-go-quality-gates/go GOMODCACHE=/tmp/ai-arena-go-quality-gates/go/pkg/mod GOCACHE=/tmp/ai-arena-go-quality-gates/go-build go test ./internal/platform/session -run TestSessionInitTurnTimeoutGameOverAndLateResponse -count=20`
- result:
  local rerun は 20 回連続で通過し、同じ failure は再現しなかった

## Proposed Solution

- `internal/platform/session/session_test.go` の
  `TestSessionInitTurnTimeoutGameOverAndLateResponse` で、
  `game_over` 通知と late response 観測の順序が timing-sensitive になっていないか確認する
- test helper 側で `GameOver` status 観測前に別 event が割り込む余地があるなら、
  synchronization point を追加して assertion を timing race から切り離す
- 再現性を上げるため、必要なら targeted stress loop や
  failure 時の event dump を追加して fix 前に flaky mode を固定する

## Priority

`execute-task` PR の required CI を unrelated failure で赤くし続けるため、
workflow 上の landing を阻害する。session 周辺の変更がなくても再発するなら、
今後の実装 PR 全般のノイズ源になる。
