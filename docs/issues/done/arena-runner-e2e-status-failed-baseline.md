# arena-runner e2e で `status = "failed"` が返る

## Summary

`pinact` rollout branch の検証中に、workflow/docs しか変更していないにもかかわらず
`make test` が `e2e/arena_runner_test.go` で失敗した。

確認した failure は以下:

- `TestArenaRunnerHappyPaths/simultaneous`
- `TestArenaRunnerHappyPaths/sequential`
- multiple `TestArenaRunnerFailurePaths/*` cases
- `TestArenaRunnerStartFromSnapshot`
- `TestArenaRunnerResumeFromHistoryAndContinue`

いずれも `status = "failed", want completed` を報告した。

## Impact

- この `pinact` rollout branch では `make test` を green にできていない。
- この failure が `main` baseline でも再現するか、branch 固有かはまだ未確認。
- arena-runner の実行結果まわりに別修正が必要な可能性があるため、この branch は fully passing な `make test` の証跡を出せていない。

## Follow-up

- `main` で再現して baseline 由来か branch 固有かを切り分ける。
- happy-path 1 件と failure-path 1 件について artifact/status output を取り、runner がどこで `failed` に遷移するかを確認する。
- test 側の期待値が current runner contract から drift したのか、runner 自体が regression したのかを判断する。

## Resolution

2026-05-12 に `main` で再確認したところ、この failure は再現しなかった。

- `go test ./e2e -run 'TestArenaRunnerHappyPaths|TestArenaRunnerFailurePaths|TestArenaRunnerStartFromSnapshot|TestArenaRunnerResumeFromHistoryAndContinue' -count=1` は成功した。
- `make test` も成功し、issue に記載した `e2e/arena_runner_test.go` の `status = "failed", want completed` は出なかった。

このため、pinact rollout branch 固有の一時的な failure と判断して issue を close する。
