# gamemaster-manifest-relative-command-resolution
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`arena-runner --game-master-manifest` で `runtime.command[0]` に relative path を
書いたとき、spec では manifest file 配置ディレクトリ基準で解決すると定義しているが、
consumer repo 実地検証では `fork/exec ... no such file or directory` で失敗する。

この plan では、game master manifest overlay の local-subprocess command 解決を
consumer repo でも実際に起動可能な形へ揃え、spec と実装を再一致させる。

## Context

- `docs/exec-plan/done/0052-external-gamemaster-manifest-registration-01-dev-runner-overlay.md`
  では、manifest file 基準の relative path 解決を追加済みとして扱っている
- `docs/specs/game-master.md` と `reversi-ai-arena/docs/specs/tagged-runner-consumption.md`
  は、`runtime.command[0]` の relative path は manifest file ディレクトリ基準で
  解決すると明記している
- しかし現行 `cmd/arena-runner/gamemaster_manifest.go` は relative path を
  `filepath.Join(filepath.Dir(manifestPath), command[0])` しただけで、
  `exec` 実行時に確実に startable な absolute path へ正規化していない
- `reversi-ai-arena` の local manual verification では、README を absolute path
  workaround に変えると動き、manifest-relative command のままだと失敗した

## Scope

- game master manifest overlay の relative command 解決を、実行時に startable な
  path へ正規化する
- spec wording を「manifest 基準で解決される」だけでなく、
  runner がその解決済み entrypoint を実行する責務まで明確化する
- consumer-style manifest 配置での regression test を追加する

この plan では以下を扱わない。

- AI sidecar manifest の runtime resolution 仕様変更
- game master manifest の fresh-run-only 制約緩和
- official registration / persisted registry の admission policy
- `reversi-ai-arena` 側 README や issue tracking 以外の consumer-repo docs cleanup

## Spec Changes

### `docs/specs/game-master.md`

- external game master manifest 節で、relative `runtime.command[0]` は
  manifest file ディレクトリ基準で解決した後、runner が startable な command path
  として local-subprocess adapter へ渡すことを明記する
- path 解決失敗は match loop 開始前に fail-fast することを明記する

### `docs/specs/platform.md`

- dev-only game master manifest overlay の責務として、
  consumer-supplied manifest から startable local-subprocess command を構築する
  ことを追記する

## Expected Code Changes

- `cmd/arena-runner/gamemaster_manifest.go`
  - manifest-relative command resolution の結果を、`exec` 実行で曖昧さが残らない
    canonical path に正規化する
  - command path 解決 helper の責務を明確化する
- `cmd/arena-runner/main_test.go`
  - relative command resolution が absolute / startable path になることを確認する
  - consumer-style manifest 配置で fresh run が通る regression coverage を追加する
- 必要なら `cmd/arena-runner/testdata/game-master/external-echo/` または
  近傍 helper を拡張し、relative command 実行失敗を再現しやすい fixture shape を
  足す

## Design Decisions

- fix は consumer repo workaround ではなく runner 側で吸収する
- game master manifest の relative command は shell 展開ではなく、
  runner が manifest location 基準で deterministic に解決する
- path resolution の正しさは unit assertion だけでなく、実起動まで含む regression
  test で固定する

## Sub-tasks

- [ ] spec wording を spec/code parity が取れる表現へ更新する
- [ ] `resolveGameMasterRuntime` の path normalization を見直し、
      startable command path を返すようにする
- [ ] [parallel] relative command resolution の unit expectation を更新する
- [ ] [parallel] consumer-style manifest layout での fresh-run regression test を追加する
- [ ] [depends on: path normalization, regression test] fail-fast error surface が
      relative-path miss ではなく実際の missing target を報告することを確認する

## Parallelism

- spec wording の更新と regression test fixture の準備は並行で進められる
- code fix と unit expectation 更新は同じ surface だが、regression test 追加は
  fixture 準備が済めば独立して進められる

## Dependencies

- informed by: `docs/exec-plan/done/0052-external-gamemaster-manifest-registration-01-dev-runner-overlay.md`
- informs: `docs/exec-plan/todo/0054-external-gamemaster-manifest-registration-03-resume-replay-follow-up.md`

## Verification

- `resolveGameMasterRuntime` の test が、manifest-relative input から startable path を
  返すことを確認する
- `arena-runner` fresh-run test が、consumer-style manifest 配置と relative
  `runtime.command[0]` で game master を正常起動できることを確認する
- missing target の failure test が、path normalization 後も期待どおり
  `no such file or directory` を返すことを確認する

## Risks and Mitigations

- relative path を absolute 化することで既存 test expectation が崩れる
  - mitigation: spec wording と test expectation を同じ change set で揃える
- `command[0]` だけ直して args や Dir との整合が崩れる
  - mitigation: regression test で実起動まで通し、`Dir` を含む起動 contract を固定する
- consumer repo workaround の README だけが先行して長く残る
  - mitigation: consumer repo 側 issue/README から ai-arena plan を参照し、
    実装後に absolute-path workaround を外せる前提を明記する
