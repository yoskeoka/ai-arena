# phase8-public-state-and-reversi-visualizer-roadmap
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

Addresses: `docs/issues/0033-local-operator-completed-detail-missing-record-artifact.md` (先行検証。再現時のみ修正 child plan を作成する)。

## 目的と完了境界

Reversi の release 済み game / bot を用いた operator match が terminal まで到達することを出発点に、
Phase 8 を「公開してよい exported state だけを game repo の replay viewer へ届ける」経路として分割する。

この親 plan の完了は、実装完了ではない。次の child plan を実行可能な順序と contract gate に分解し、
artifact replay、公開 state delivery、live watch を混同しないことを完了境界とする。各 child は実装前に
対応する repo の `docs/specs/` と TypeSpec source を更新する。

`N/A - detail required before execution`: 本文は cross-repository roadmap であり、API field、認可方式、
state cadence、viewer UX は child plan で具体化してから実装する。親 plan 自体は code / API mutation を行わない。

## 現状と根拠

- Phase 6 は terminal artifact と durable locator の基盤を既に持つ。`docs/specs/platform-service-persistence.md`
  の保存単位と backend split、`internal/platform/service/worker.go:83-115` の persist-before-complete、
  `internal/platform/service/worker_s3.go:28-99` の `record` / `snapshot` / `exported-snapshot` /
  `history` / summary persist がその根拠である。
- `record.json` は replay/audit の正本だが、public ではない。`docs/specs/platform-common-contract.md:312-335`
  は exported snapshot を観戦用 public shape とし、`docs/specs/platform-service-read-model.md:138-186`
  は既存 read model が operator 向けであることを定める。`typespec/namespaces/public/api.tsp:1-3` は予約 namespace
  のみで、public spectator API は未実装である。
- Reversi は `docs/specs/visualizer-architecture.md:5-32` と `docs/project-plan.md:85-107,145-168` で
  Phaser + lightweight shell の artifact-first replay を求め、live watcher は public platform API 後の Phase 4 に
  明確に分離している。`visualizer/src/main.ts:1-20` は scaffold に留まる。
- Reversi の `docs/specs/artifact-kifu-export.md:20-80` と reusable transcript parser は、`record.json` 優先、
  accepted placement / explicit pass の保持、非 turn / non-accepted event の除外を既に定める。ただし Rust の
  filesystem parser を browser client が直接利用できるわけではない。

## 先行 gate と優先順位

1. **artifact integrity の確認（最優先・短期）**: Reversi の実際の terminal run について `record.json`、
   `exported-snapshot.json`、`history.json`、`result-summary.json` の object と persisted locator を照合する。
   `0033` が再現した場合は、viewer の degraded UI で隠さず、生成・persist・key・completed 遷移のいずれかを
   原因として確定する fix plan を先に実施する。再現しなければ issue に現行 evidence を追記して viewer block
   から外す。
2. **general operator lane の回帰固定（推奨先行）**: active `0122-wasi-bundle-admission-e2e-regression` を実行し、
   immutable ZIP upload -> activation -> bot revision -> request -> worker -> ranking を filesystem / S3-compatible
   lane で自動化する。今回の Reversi manual run は acceptance evidence だが、general lane の回帰テストの代替にはしない。
3. **legacy preset lane の撤去（0122 の後）**: active `0123-preset-match-api-retirement` を実行し、新規 match 作成を
   registered scope + admitted bot の一経路へ収束する。これは public API の仕様化を阻害しないが、運用と fixture の
   identity を単純化する。
4. **public spectator contract（Phase 8 の実装開始 gate）**: operator read model / protected artifact route を流用せず、
   exported state 専用の public API family を先に定義する。Phase 6 の private artifact persistence はこの gate の
   十分条件ではない。

`0119` metadata cache、`0086` ranking fact store、`0124` cleanup design、`0092` / `0096` local browser runtime
alignment は、それぞれ performance、future ranking、destructive operation、test harness の follow-up であり、
artifact replay / public spectator contract の開始 blocker にはしない。`0041` の external log retention は Phase 7
全体を完了と宣言する前の運用 gate として残すが、local replay prototype の blocker にはしない。

## 実行 child plan の構成

### A. Artifact integrity evidence または修正

- ai-arena で Reversi の one-run evidence を compact artifact から順に検証する。`record.json` を含む terminal
  artifact が、queue record の locator、match metadata、player order と一致することを確認する。
- 失敗時は `0033` を Addresses にした fix plan を作り、completed への遷移を partial artifact persist で許容しない
  behavioral contract を spec-first で固定する。detail degradation は source-of-truth integrity の代替にしない。
- 成功時も、その fixture / command を後続の public-delivery と Reversi replay の contract fixture として固定する。

### B. Phase 8 public exported-state delivery

- `(MODIFY) ai-arena/docs/specs/`: public spectator resource の discover / list / detail、terminal replay artifact、
  in-progress latest state、terminal / unavailable / retention の observable behavior を定義する。private
  `record` / `snapshot` / `history` / stderr / bundle bytes を public path から除外する。
- `(MODIFY) ai-arena/typespec/namespaces/public/api.tsp` と shared TypeSpec: public wire contract の唯一の field-level
  source を定義する。match identity、game metadata、monotonic public-state version / turn、lifecycle、public state、
  cache / retry hint を扱うが、game-specific `public_state` payload を platform が解釈しない。
- `(MODIFY) ai-arena/internal/platform/service/*`: durable locator から exported snapshot を安全に読む public read
  adapter を追加する。公開対象の discoverability と anonymous/public access の policy は child plan で human review
  を経て確定する。URL/token を durable write model に保存しない。
- 最初の更新方式は **snapshot polling** を採用候補として評価する。state version と cadence を契約化し、terminal
  replay と in-flight state を同じ resource family で読めるようにする。SSE/WebSocket、per-event public feed、
  reconnect cursor は stable polling contract と load / latency requirement が確認できた後の別 child とする。

### C. Reversi artifact-first replay visualizer

- `(MODIFY) reversi-ai-arena/docs/specs/visualizer-architecture.md` と `(MODIFY) docs/specs/artifact-kifu-export.md`:
  browser replay input を、public exported snapshot と lossless accepted-turn transcript から再構成する contract として
  固定する。`record` / `history` precedence、pass、malformed input、failed/canceled terminal result を shared fixtures
  で検証する。
- `(NEW) reversi-ai-arena/visualizer/src/replay/*`: browser-side normalizer と immutable replay model を実装する。
  Rust filesystem helper の再利用は source copy ではなく、versioned neutral DTO、shared fixture、または verified
  WASM bridge を比較して child plan で選ぶ。private engine state と browser filesystem assumption を持ち込まない。
- `(MODIFY) reversi-ai-arena/visualizer/src/main.ts` と `(NEW) renderer / control modules`: Phaser board scene、
  turn stepping、play / pause、seek、pass、score / current player、terminal result、input / network error を lightweight
  web-standard shell に実装する。React や Reversi-specific backend bypass は追加しない。
- verification は parser/reconstruction golden、pass-bearing fixture、malformed / early-failure input、Phaser-independent
  state test、browser load -> terminal replay e2e、`npm run typecheck` / `npm run build` を含める。

### D. Platform-to-Reversi 接続と live-watch の分離

- C の local artifact loader を B の public terminal resource へ接続し、same match の public metadata、final exported
  snapshot、turn/result が fixture と一致することを contract / browser test で示す。
- in-progress polling は B の cadence/version semantics に従う。viewer は old response を version で破棄し、terminal
  state 後に polling を止める。
- real-time event stream は Reversi Phase 4 / ai-arena follow-up として別 plan にする。stream を先に導入するために
  record event log を public に出すこと、Reversi 固有 endpoint を作ること、private snapshot を露出することを禁止する。

## 依存関係と並行性

```text
A artifact integrity
  ├─ (success) ──────────────┐
  └─ (failure: fix first) ───┤
0122 general-lane regression ─┼─> B public exported-state contract/delivery ─> D platform connection
0123 preset retirement ───────┘                                      └────> C Reversi artifact replay
                                                                                │
                                                                        live stream (separate later plan)
```

- A は直ちに開始する。B の implementation は A が artifact integrity を証明するまで開始しない。
- 0122 は B/C の設計と並行に実施できるが、public delivery の remote acceptance 前には完了させる。0123 は 0122
  後に実施し、B の TypeSpec/spec drafting 自体は妨げない。
- B の stable exported-state wire contract が固定された後、C の browser normalizer / renderer と B の service
  adapter は並行可能である。D は両者に依存する。

## child plan ごとの受入条件

- A: completed record が terminal artifact の実在と一致し、Reversi run を inputs とした record/locator/object
  integrity evidence が残る。
- B: unauthenticated public client が discoverable public match の exported state だけを取得でき、private artifact、
  stderr、AI/game bundle、internal snapshot を取得できない。in-flight と terminal の version/lifecycle/retention が
  TypeSpec と black-box test で一致する。
- C: runner-derived public fixturesを browser で読み、開始から terminal まで合法な Reversi board progression と
  pass / score / winner を再生できる。private engine fields なしで入力を再構成できる。
- D: public platform resource を選ぶ viewer が final replay を再生し、running match では stale state を描画せず、
  terminal で polling を停止する。

## 非目標

- Phase 8 初回で full event streaming、chat、AI evaluation overlay、analysis workstation、leaderboard redesign を
  実装しない。
- public API を operator API の認証緩和や signed operator artifact URL の再利用で代用しない。
- R2 / Postgres に artifact bytes を二重保存しない。Phase 6 の metadata/locator と object artifact の責務分離を
  維持する。
- `record.json`、internal `snapshot.json`、`history.json`、stderr を「replay が便利」という理由で公開しない。

## 検証と handoff

- 各 implementation child はその repo の spec-first update、focused test、full non-AI quality gates、workflow lint、
  PR latest-head follow-up を完了する。
- public delivery child は local filesystem と S3-compatible object-storage lane の双方で private/public boundary と
  stale/terminal behavior を検証し、remote staging では provider deployment、`/version`、API response の実測を
  分けて記録する。
- Reversi child は public terminal fixture の visual regression evidence と accessibility に必要な non-canvas text
  summary を確認する。実ブラウザの完走確認を、typecheck/build の代替にしない。
