# reversi-game-manual-registration-verification
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: N/A

## Objective

人間の operator が local または staging 環境で Reversi game bundle を upload し、admission 済みの
artifact から game registration を作成できることを確認する。この確認は特定の公開 release を CI の
staging verification に組み込むことを目的にしない。

完了境界は、選んだ環境で operator UI から Reversi game の upload、admission、registration が成功し、
作成された competition scope と artifact digest を人間が確認できることである。AI bundle、bot 作成、
Shuffle、queue、ranking、rerun/promote、service restart/recovery はこの手順の対象外とする。

## Human Verification Checklist

- [ ] local または staging のどちらか一方を選び、operator として sign in する。staging を選ぶ場合は、
  shared data と衝突しない registration ID を使う。
- [ ] Reversi game bundle を operator UI から upload し、admission 成功後に表示される artifact digest を
  控える。bundle の取得元と checksum を確認したい場合は
  `docs/development/reversi-release-artifact-verification.md` を参照する。
- [ ] Games 画面で admission 済み artifact を指定し、bundle から導出される game metadata と ruleset を
  使って game registration を作成する。
- [ ] 作成後の competition scope に game identity、artifact digest、ruleset が表示されることを確認する。
- [ ] 環境、時刻、registration ID、artifact digest、画面または API response の locator を手元の
  handoff evidence として残す。失敗時は、再現手順と error を添えて原因を限定した follow-up plan / issue を
  作る。

## Existing Implementation References

- `operator-ui/src/routes/operator/GamesPage.tsx`
  - admission 済み game artifact から registration を作る operator UI
- `docs/development/operator-ui-local-verification.md`
  - local operator UI と service の起動・観察方法
- `docs/development/reversi-release-artifact-verification.md`
  - Reversi release artifact を任意に取得・checksum 確認する手順
- `.github/workflows/online-release-staging-verify.yml`
  - diagnostic preset による deploy 後の自動確認。Reversi bundle は入力にしない

## Code Change Map

- `docs/issues/<next>-reversi-game-manual-registration-*.md` (NEW, conditional)
  - 手動登録で observable contract を満たさない失敗があった場合だけ記録する
- runtime / operator UI source (MODIFY, conditional)
  - failure が既存 contract を満たさないことを示す場合だけ、別 execution plan を先に作る

## Dependencies and Parallelism

- local を選ぶ場合は operator UI と artifact backend を起動できる開発環境が必要
- staging を選ぶ場合は対象 deploy、operator 権限、隔離した registration ID が必要
- automated staging verification はこの確認の dependency ではない

## Verification

- human operator が local または staging で Reversi game bundle を upload / admit / register できる
- 表示された competition scope が game identity、ruleset、artifact digest を保持する
- `online-release-staging-verify` は Reversi game または AI release asset を download、upload、register
  しない

## Risks and Mitigations

- shared staging の既存 scope と registration ID が衝突する
  - mitigation: 人間が一意な registration ID を選び、既存の scope を変更しない
- 公開 release asset の完全性も同時に確認したくなる
  - mitigation: 任意の手動確認として artifact verification 文書を参照し、CI gate には昇格させない
