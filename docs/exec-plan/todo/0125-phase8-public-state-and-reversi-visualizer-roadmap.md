# phase8-public-state-and-reversi-visualizer-roadmap
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.
>
> **Parent boundary**: Do not run `/execute-task` on this roadmap until it has been split into reviewed child plans. Execute each merged child plan instead.

Addresses: N/A

## 目的と完了境界

Phase 8 を「公開してよい game state を platform から取得し、game repo の viewer で観戦できる」経路として、
実行可能な child plan に分解する。terminal artifact replay、in-progress snapshot polling、将来の event stream を
同じ delivery mechanism として扱わず、private artifact と public spectator resource の境界を先に固定する。

この親 plan の完了は Phase 8 の実装完了ではない。public contract、Reversi の artifact-first viewer、platform 接続、
event stream の順序と依存を child plan に分け、各 child が独立した black-box contract と検証を持つ状態を完了境界とする。
その分割は ai-arena の `0126`、reversi-ai-arena の `0001`、ai-arena の `0128`--`0129` で完了した。`0126` と `0001` は実装へ渡す詳細 plan、`0128` と `0129` は前者の実測と
versioned contract を受けてから詳細化する intentional parent plan である。

`N/A - detail required before execution`: 本文は cross-repository roadmap である。API field、公開対象の選択 policy、
state cadence、viewer UX、stream transport は child plan で具体化し、人間の review を経てから実装する。
親 plan 自体は code / API mutation を行わない。

## Phase 8 の開始点

- Phase 6 / Phase 7 の milestone 本体は `docs/project-plan.md` で完了済みである。残る metadata cache、ranking fact
  store、cleanup、legacy retirement、test harness、外部 log retention は運用・保守上の deferred follow-up であり、
  public spectator contract の開始条件にはしない。
- terminal artifact の生成、persist-before-complete、durable locator、operator read model は
  `docs/specs/platform-service-persistence.md:28-121`、`docs/specs/platform-service-read-model.md:138-186`、
  `internal/platform/service/worker.go:83-115`、`internal/platform/service/replay_inputs.go:38-170` で成立している。
  これらは public delivery の入力基盤であって、既存 operator route を public にしてよい根拠ではない。
- `record.json` / `history.json` は replay / audit の source of truth だが public artifact ではない。
  `docs/specs/platform-common-contract.md:312-335` が定める exported snapshot を latest public state の起点とし、
  terminal replay には game が別に生成する public replay payload を使う。
- `typespec/namespaces/public/api.tsp:1-3` は予約 namespace のみであり、public spectator API は未実装である。
- Reversi は `reversi-ai-arena/docs/specs/visualizer-architecture.md:5-32` と
  `reversi-ai-arena/docs/specs/artifact-kifu-export.md:20-80` に artifact-first replay と accepted turn / explicit pass の
  contract を持つが、`reversi-ai-arena/visualizer/src/main.ts:1-20` は scaffold に留まる。

## child plan の構成と実行状態

| 区分 | child plan | 現在の粒度 | 実行条件 |
| --- | --- | --- | --- |
| A | `0126-phase8-public-state-and-reversi-visualizer.md` | 詳細な実装 plan | 全 spectator resource を認証なしで読む public API、official-run selection、public cross-origin read を固定済み |
| B | `reversi-ai-arena/docs/exec-plan/todo/0001-phase8-public-state-and-reversi-visualizer-reversi-replay-viewer.md` | 詳細な実装 plan | Reversi 提供者が外部ホストする公式 TypeScript/Phaser reference viewer。A の public API / fixture を入力にする |
| C | `0128-phase8-public-state-and-reversi-visualizer-platform-connection.md` | intentional parent | A/B が public anonymous delivery を完了した後に、external visualizer 向け access protection を詳細化する。Phase 8 の完了条件にはしない |
| D | `0129-phase8-public-state-and-reversi-visualizer-event-stream-evaluation.md` | intentional parent | A/B の public polling load / latency / reconnect evidence を評価後に詳細 plan を新規作成する。Phase 8 の完了条件にはしない |

以下は cross-repository dependency の roadmap summary である。各区分の implementation scope、spec-first change、
verification はリンク先 child plan を正本とする。

### A. Public exported-state contract と delivery

詳細 plan: `docs/exec-plan/todo/0126-phase8-public-state-and-reversi-visualizer.md`

- `(MODIFY) docs/specs/`: public spectator resource の discover / list / detail、terminal public replay、
  in-progress latest state、terminal / unavailable / retention の observable behavior を定義する。private
  `record` / `snapshot` / `history` / stderr / bundle bytes を public path から除外する。
- `(MODIFY) typespec/namespaces/public/api.tsp` と shared TypeSpec: public wire contract の唯一の field-level source を
  定義する。match identity、game metadata、monotonic public-state version / turn、lifecycle、opaque な game-specific
  `public_state`、terminal replay format / version / payload、cache / retry hint を扱う。
- `(MODIFY) game master output / artifact persistence contract`: game が public として生成した versioned replay payload を
  private event log とは別 artifact として保存し、terminal public replay resource から読める stable locator を保持する。
  platform が `record.json` / `history.json` を後から filter して public transcript を生成してはならない。payload は
  game-specific かつ opaque とし、platform は共通 envelope、size/version、locator、retention だけを扱う。
- `(MODIFY) internal/platform/service/*`: durable locator から latest exported state と terminal public replay だけを読む
  anonymous public read adapter を追加する。browser を含む external visualizer はこの API を読めるが、ai-arena は game 固有
  visualizer の asset、runtime、rendering を host しない。operator authorization の緩和や signed operator artifact URL の
  再利用で代用しない。
- 初回 delivery は snapshot polling を候補として、cadence、cache、stale response、terminal stop を契約化する。
  SSE / WebSocket、per-event feed、reconnect cursor は stable polling contract と load / latency requirement の確認後に
  別 child plan とする。
- verification は filesystem / S3-compatible backend の双方で public/private boundary、in-progress version の単調性、
  public replay の format/version/access/retention、stale response、terminal / unavailable を black-box test する。remote
  staging は provider deploy、exact `/version`、`/healthz` readiness、public API response を別々の証跡として残す。

### B. Reversi artifact-first replay viewer

詳細 plan: `reversi-ai-arena/docs/exec-plan/todo/0001-phase8-public-state-and-reversi-visualizer-reversi-replay-viewer.md`

- `(MODIFY) reversi-ai-arena/docs/specs/visualizer-architecture.md` と
  `(MODIFY) reversi-ai-arena/docs/specs/artifact-kifu-export.md`: browser replay input を A の terminal public replay payload と
  final exported snapshot から再構成する contract として固定する。Reversi の public replay format は lossless な
  accepted placement / explicit pass の transcript を持ち、private `record` / `history` を viewer input にしない。
  malformed input、failed / canceled terminal result を shared fixture で検証する。
- `(NEW) reversi-ai-arena/visualizer/src/replay/*`: browser-side normalizer と immutable replay model を実装する。
  Reversi reference viewer は versioned neutral DTO と shared public fixture を使う TypeScript normalizer を選び、game
  提供者が別 host で提供する static web application とする。ai-arena への static link、plugin asset upload、browser filesystem
  assumption を持ち込まない。
- `(MODIFY) reversi-ai-arena/visualizer/src/main.ts` と `(NEW) renderer / control modules`: Phaser board scene、turn
  stepping、play / pause、seek、pass、score / current player、terminal result、input / network error を lightweight な
  web-standard shell に実装する。React や Reversi 固有 backend bypass は追加しない。
- verification は parser / reconstruction golden、pass-bearing fixture、malformed / early-failure input、
  Phaser-independent state test、browser load から terminal までの replay E2E、`npm run typecheck` / `npm run build`、
  accessibility に必要な non-canvas text summary を含める。

### C. External visualizer の access protection

Intentional parent: `docs/exec-plan/todo/0128-phase8-public-state-and-reversi-visualizer-platform-connection.md`。
ai-arena の `0126` と reversi-ai-arena の `0001` を完了するまで、この section を implementation plan として実行してはならない。

- A/B は anonymous public API と external-hosted reference viewer を end-to-end で成立させる。C はその後に、利用者や
  external visualizer をどの認可 model で保護するかを決める follow-up である。web browser、native application、CLI の
  credential / redirect / token / origin policy を A の anonymous contract に遡って混在させない。
- access protection を導入する場合、game provider が host する browser visualizer、non-browser client、API consumer の
  互換性、migration / rollback、公開対象 selection、cross-origin behavior を human review で決める。A の public response から
  private artifact を露出しない境界は維持する。

### D. Event stream の評価と追加

Intentional parent: `docs/exec-plan/todo/0129-phase8-public-state-and-reversi-visualizer-event-stream-evaluation.md`。
`0128` の polling evidence を得るまで、この section を implementation plan として実行してはならない。

- snapshot polling の load / latency / reconnect evidence を取得し、event stream が必要な場合だけ transport、cursor、
  ordering、retention、terminal close、backpressure を child plan で定義する。
- stream は public exported-state family の追加 transport とする。`record.json.event_log` の直接公開、Reversi 固有
  platform endpoint、private snapshot の露出は禁止する。
- stream を追加しない判断をする場合は、Phase 8 の spectator latency を polling が満たす証拠と、再評価条件を
  `docs/project-plan.md` または該当 spec に残す。

## 依存関係と並行性

```text
A anonymous public exported-state contract
  ├──> versioned public fixture ──> B external-hosted Reversi reference viewer
  └──> public polling semantics ──> B ──> C access protection (post-Phase 8)
                                        └──> D stream evaluation (post-Phase 8)
```

- A の wire / visibility contract と anonymous HTTP resource を最初に固定する。B の実装は A の merged fixture / resource を
  入力にして開始し、未 merge branch を dependency にしない。
- B は A の stable contract と fixture に依存し、anonymous public resource へ直接接続する。C は A/B の実装に依存する
  post-Phase 8 follow-up、D は A/B の public polling evidence に依存する post-Phase 8 follow-up である。
- child plan はそれぞれ該当 repo の最新 `main` から作成し、spec-first update、非 AI quality gate、workflow lint、
  latest-head follow-up を独立して完了する。

## child plan ごとの受入条件

- A: anonymous client が discoverable public match の latest exported state と
  game-produced terminal public replay だけを取得でき、private artifact、stderr、AI / game bundle、internal snapshot を
  取得できない。in-flight と terminal の version / lifecycle / retention が TypeSpec と black-box test で一致する。
- B: runner-derived public fixture を browser で読み、開始から terminal まで合法な Reversi board progression と pass / score /
  winner を再生できる。private engine field なしで入力を再構成できる。
- C: authenticated / policy-controlled delivery を追加する場合、外部 host の web viewer と native / CLI consumer が
  documented access contract に従い、anonymous Phase 8 client から安全に移行できる。
- D: event stream を追加する場合は public state の ordering / reconnect / close contract を満たす。追加しない場合は polling が
  spectator latency を満たす測定結果と再評価条件が残る。

## 非目標

- 最初の child で full event streaming、chat、AI evaluation overlay、analysis workstation、leaderboard redesign を実装しない。
- public API を operator API の認証緩和や signed operator artifact URL の再利用で代用しない。
- game 提供者の visualizer asset を ai-arena が登録・host・実行する経路は本 Phase 8 に含めない。将来検討は
  `docs/issues/0120-platform-hosted-visualizer-registration.md` に残す。
- access protection 後の event / tournament 用 anonymous opt-in は本 Phase 8 に含めない。将来検討は
  `docs/issues/0121-public-spectator-access-protection-and-anonymous-opt-in.md` に残す。
- R2 / Postgres に artifact bytes を二重保存しない。metadata / locator と object artifact の責務分離を維持する。
- `record.json`、internal `snapshot.json`、`history.json`、stderr を replay のために公開しない。

## 親 plan の検証と handoff

- 各 child が目的、外部完了境界、exact reference、`(NEW)/(MODIFY)/(DELETE)` map、black-box spec change、依存、
  verification を持ち、そのまま `/execute-task` に渡せることを review する。
- public wire / visibility の未決事項を A、browser normalizer / UX の未決事項を B、cross-repo versioning を C、
  transport 採否を D に閉じ込め、親 plan のまま実装を開始しない。
