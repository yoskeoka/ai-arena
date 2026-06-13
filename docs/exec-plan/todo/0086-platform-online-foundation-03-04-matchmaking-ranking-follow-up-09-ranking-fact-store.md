# platform-online-foundation-03-04-matchmaking-ranking-follow-up-09-ranking-fact-store
**Execution**: Use `/execute-task` to implement this plan.

## Objective

ranking 集計の将来拡張先を、queryable な DB fact store 基盤として固定する。
最初のゴールは、Phase 7 中の current ranking lifecycle を無理に一般集計基盤へ広げず、
Phase 7 後に着手できる deferred child plan として、
期間条件付き leaderboard や再集計の土台を整理しておくことに置く。

## Context

- current ranking lifecycle は、completed match の `result-summary` から
  durable snapshot を増分更新し、recompute / verify 時だけ compact summary を走査する
- この方式は current 想定規模の online operator cycle では十分軽い一方、
  「直近 1 か月」「season 単位」「competitor / game filter」などの query flexibility には弱い
- `0079-platform-online-foundation-03-04-matchmaking-ranking-follow-up-06-rerun-retry-cancellation.md`
  が整理する correction / rerun semantics は、将来の ranking facts の正本定義に直結する
- project-plan 上の current priority はまだ Phase 7 であり、この plan は
  Phase 7 の child items を優先した後でも迷わず着手できる deferred follow-up として残したい

## Option Snapshot

### Option A: object storage snapshot を拡張し続ける

- 利点:
  current 実装の延長で済み、追加 infra が少ない
- 欠点:
  時間窓や filter 条件ごとの再集計が重くなり、
  ranking を durable cache ではなく擬似 analytical store として扱うことになる

### Option B: ranking fact を DB へ落とし、snapshot は DB 由来に寄せる

- 利点:
  bounded time window、correction、backfill、verification を扱いやすい
- 欠点:
  schema と recompute path の責務整理が増える

### Option C: Redis など live cache を先に導入する

- 利点:
  low-latency leaderboard read には向く
- 欠点:
  authoritative fact store を解決しないまま cache invalidation を先に抱える

## Recommendation

- Option B を採用する
- ranking の authoritative aggregation substrate は Postgres など queryable DB に寄せる
- current object-storage / local snapshot は Phase 7 の durable lifecycle として維持しつつ、
  将来は export または cache 扱いへ下げる
- Redis 相当の live cache は、この plan では扱わず、real read-latency requirement が
  出た後の別 follow-up に分離する

## Scope

- ranking 集計用の durable fact row をどの粒度で DB に保存するか定義する
- all-time と bounded time window の両方を支えられる read model 方向を定義する
- current snapshot path と future DB-backed aggregate path の責務分離を定義する
- correction / rerun / recompute が DB facts にどう反映されるかの原則を定義する

この plan では以下を扱わない。

- public leaderboard UI
- season reward / division system
- Redis や dedicated cache layer
- Phase 7 完了前に current ranking lifecycle を置き換える作業

## Spec Changes

- ranking lifecycle spec を更新し、authoritative fact store と durable snapshot の役割を分離する
- `docs/specs/platform-service-read-model.md` に、current operator read model と
  future leaderboard query surface の境界を補足する
- rerun / retry / cancellation spec と cross-reference し、
  correction が ranking facts へ与える影響を明示する

## Expected Code Changes

- ranking fact 用の Postgres schema / migration
- per-match `result-summary` から fact row を materialize する persistence path
- DB facts から all-time / bounded-window ranking を再構築する query / recompute path
- verification helper と fixture 拡張

## Sub-tasks

- [ ] ranking fact row の最小 schema を定義する
- [ ] all-time / bounded-window leaderboard の query contract を定義する
- [ ] current snapshot と future DB aggregate の source-of-truth 境界を定義する
- [ ] rerun / correction / backfill 時の fact mutation rule を定義する
- [ ] recompute / verify が object storage scan から DB facts 中心へ移る手順を整理する

## Parallelism

- [parallel] fact schema 整理と leaderboard query contract の叩き台作成は並行できる

## Dependencies

- depends on: `0078-platform-online-foundation-03-04-matchmaking-ranking-follow-up-05-ranking-lifecycle.md`
- depends on: `0079-platform-online-foundation-03-04-matchmaking-ranking-follow-up-06-rerun-retry-cancellation.md`
- depends on: `0082-platform-service-db-migration-release-flow.md`
- depends on: parent/base item `0067-platform-online-foundation-03-04-matchmaking-ranking-follow-up.md` (retired after split)

## Suggested Timing

- Phase 7 の current child items を優先し、この plan は Phase 7 後または
  Phase 8 以降へ送ってよい
- ただし、time-window leaderboard や ranking correction の要求が先に強まった場合は、
  Phase 7 の後半で前倒ししてよい

## Risks and Mitigations

- current snapshot を一般集計基盤として延命すると、windowed query と correction が複雑化する
  - mitigation: authoritative facts は DB に寄せ、snapshot を analytical source にしない
- rerun / cancellation semantics が先に固まっていないと fact correction rule がぶれる
  - mitigation: `0079` の lifecycle vocabulary を先に踏まえる
- live cache を早まって入れると source-of-truth が増え、運用が重くなる
  - mitigation: cache 導入は別 plan に分離する

## Design Decisions

- ranking の将来拡張先は object storage aggregate ではなく queryable DB fact store とする
- current Phase 7 ranking lifecycle は短期の運営基盤として維持し、この plan は deferred follow-up とする
