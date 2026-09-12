# public-spectator-access-protection-and-anonymous-opt-in

## Summary

Phase 8 A の anonymous public spectator API を protected access へ移行する設計と、その後に event / tournament
等だけ anonymous viewing を明示 opt-in できる policy を検討する。

## Why It Remains Open

first delivery は game provider が host する web visualizer、native application、CLI、直接 API consumer が同一の exported-state-only
API を読めるよう、session を要求しない。これにより Reversi reference viewer を含む visualizer を public API だけで完結できる。

将来、通常の観戦を ai-arena identity で保護する場合は、外部 host の browser client を session cookie に暗黙依存させず、non-browser
consumer も含めた authorization / migration / rollback を決める必要がある。保護後も、運営公式が十分な capacity / cache / load plan を
用意する event や tournament では匿名観戦を広げたい場合があるため、match 作成時の明示 opt-in が候補になる。これは現在の anonymous
first contract に visibility flag を混ぜる理由にはならない。

## Follow-up Boundary

- protected access の対象、external web / native / CLI / direct API consumer の credential model、redirect / token / origin policy、
  CORS、revocation、migration / rollback を human review で決める。
- access protection を導入した後だけ、match creation 時の anonymous opt-in、default、audit、retry/rerun/promotion semantics、
  existing match migration を定義する。
- event / tournament の anonymous opt-in には capacity、cache、rate limit、observability、abuse response、operator accountability
  の operational contract を伴わせる。auto-scaling / procurement の実装をこの issue の完了条件にはしない。
- private `record.json`、internal snapshot、history、stderr、bundle bytes を公開しない exported-state-only boundary は、anonymous / protected
  のどちらでも維持する。具体的な access model を持つ execution plan と black-box authorization tests が review されるまで実装しない。
