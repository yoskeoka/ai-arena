# phase7-staging-reversi-acceptance-and-closeout
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective

PR #313 が main へ merge された後、staging の実 deploy 上で Reversi v0.1.0 の公開 release asset を
operator flow から upload / register / bot 作成 / Shuffle / queue / completed / ranking / rerun / promote
まで通し、Phase 7 の残る acceptance を evidence として固定する。

完了境界は workflow green だけではない。人間が staging UI と GitHub Actions artifact を確認し、
exact release URL / SHA-256、3 named bots、scope、request、run、ranking、rerun/promote/recompute の
identity を記録したうえで、`docs/project-plan.md` の Phase 7 と全 child item を完了に更新する。

## Human Staging Verification Checklist

この plan の実行担当者は以下を順番に実施し、各項目の URL、run ID、時刻、結果を PR 本文または
workflow artifact に残す。失敗時は Phase 7 を完了扱いにせず、失敗 artifact と latest deploy SHA を
添えて原因ごとの follow-up issue / plan を作る。

- [ ] PR #313 merge SHA に対する `online-release-staging` が success であり、migration が Render deploy
  より先に apply されたことを workflow summary で確認する。
- [ ] 同じ SHA の `online-release-staging-verify` を確認し、Reversi game / AI URL、SHA-256、
  `SHA256SUMS` 検証が success であることを確認する。`echo-reference` preset は acceptance evidence に
  使用しない。
- [ ] staging operator UI に operator として sign in し、Reversi game bundle と AI bundle が release asset
  由来の digest で upload / admission されたことを確認する。
- [ ] Reversi scope に所有者が識別できる 3 named active bots を作成し、各 bot の immutable active revision
  が同じ検証済み AI artifact を参照することを確認する。
- [ ] UI から Shuffle selection を実行し、bot limit と game player count を満たす request が作成され、
  queue -> leased/running -> completed と遷移することを確認する。
- [ ] completed run の `result-summary`、ranking、worker heartbeat / queue lag を確認し、service restart 後も
  scope、bots、request、run、ranking が残ることを確認する。期限切れ in-flight record を意図的に残す
  テストが必要な場合は staging data を破壊せず、隔離した verification scope を使う。
- [ ] completed run を rerun し、candidate run を promote する。ranking recompute / verify が success であり、
  official run と ranking snapshot が一致することを確認する。
- [ ] screenshots、Playwright trace、workflow summary、result-summary locator、ranking response を保存し、
  project-plan 更新に引用する evidence index を作る。

## Existing Implementation References

- `.github/workflows/online-release-staging.yml`
  - migration -> Pages / Render deploy ordering
- `.github/workflows/online-release-staging-verify.yml`
  - exact Reversi URL / digest / SHA256SUMS gate と remote verification artifact
- `docs/development/platform-service-online-deploy.md`
  - provider inventory、recovery runbook、Reversi v0.1.0 acceptance contract
- `docs/specs/platform-service-persistence.md`
  - lease / heartbeat / expired in-flight recovery observable contract
- `docs/specs/platform-service-single-worker-assumptions.md`
  - colocated single worker guard と graceful shutdown boundary
- `docs/project-plan.md`
  - Phase 7 と 3 child item の completion marker

## Code Change Map

- `docs/project-plan.md` (MODIFY)
  - 全 evidence が揃った場合だけ Phase 7 と 3 child item を complete にする
- `docs/development/platform-service-online-deploy.md` (MODIFY)
  - 実施した staging verification の evidence index と restart/recovery の結果を追記する
- `docs/issues/<next>-phase7-staging-acceptance-*.md` (NEW, conditional)
  - acceptance failure を発見した場合だけ原因を限定して記録する
- runtime / workflow / operator UI source (MODIFY, conditional)
  - failure が既存の observable contract を満たさないことを示す場合だけ、別 execution plan を先に作る

## Sub-tasks

- [ ] merge SHA と staging deploy / verify workflow run URL を取得し、対象 SHA が一致することを確認する。
- [ ] release URL / SHA-256 / `SHA256SUMS`、artifact admission、3 bot identity を evidence index へ記録する。
- [ ] Shuffle から request / queue / completed / ranking までの remote result を確認する。
- [ ] restart / recovery、rerun / promote / recompute / verify を隔離 scope で確認する。
- [ ] 失敗があれば evidence を添えて issue / follow-up plan に切り分け、成功時だけ project plan を更新する。

## Dependencies and Parallelism

- depends on: PR #313 merged to `main` and its staging deployment completed
- depends on: `online-release-staging-verify` uses the same merged commit SHA
- [sequential] release integrity -> admission/bots -> match/ranking -> restart/recovery -> project-plan closeout
- [parallel] workflow artifact collection と human UI screenshots は admission 後に並行可能

## Verification

- staging deploy / verify workflow URLs name the same merged SHA and both succeed
- published Reversi v0.1.0 ZIP bytes match pinned SHA-256 and release `SHA256SUMS`
- remote evidence identifies release artifacts, scope, three bots, request, initial/rerun/official run, and ranking snapshot
- restart preserves durable entities; expired in-flight recovery leaves no record permanently leased/running/persisting
- ranking recompute / verify succeeds after promotion
- `docs/project-plan.md` is updated only after every preceding evidence item is available

## Risks and Mitigations

- staging UI/auth/deploy drift prevents the remote scenario from starting
  - mitigation: retain workflow artifact and deploy SHA; create a narrow failure plan instead of declaring Phase 7 complete
- verification accidentally falls back to a preset or repository-local bundle
  - mitigation: require release URL, SHA-256, `SHA256SUMS`, and admitted artifact identity in evidence
- restart/recovery check mutates shared staging data
  - mitigation: use a uniquely named verification scope and bots; do not use production or another operator's scope

## Design Decisions

- Phase 7 closeout is an evidence-gathering execution after deployment, not an assertion inferred from a green deploy.
- A human staging checklist is part of the acceptance contract because authenticated UI and provider state cannot be fully established by a branch-local test.
