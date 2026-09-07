# unactivated-game-bundle-retention

## Summary

Games 画面から game bundle を upload すると、admission が成功した bytes は immutable artifact
store に保存される。一方で operator が competition scope の activation を完了せずに画面を離れる、
または registration が失敗する場合、その artifact は active release から参照されないまま残る。

## Why It Remains Open

`0110-games-inline-bundle-upload` は、既存の admission と activation を browser 上で連続して
実行できるようにする UX 改善を対象とする。artifact の保持期間、参照状態の durable な追跡、delete
authorization、active / historical run が参照する digest の保護、R2 と filesystem store 間の一貫した
削除を同じ変更に含めると lifecycle contract が広がるため、この PR では cleanup を実装しない。

## Follow-up Boundary

- activation に到達しなかった artifact を識別できる durable lifecycle state と、保留期間を定義する。
- active release、competition scope、AI revision、match / rerun の snapshot が参照する digest を絶対に
  削除しない。
- operator が abandon を明示する場合と、期限切れ artifact を回収する場合の authorization、監査、
  idempotency を定める。
- filesystem / R2 bundle store の両方で、metadata と bytes の partial failure を回復可能にする。

