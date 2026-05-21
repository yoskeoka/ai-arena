# platform-online-foundation-02-persistence-and-read-model
**Execution**: Use `/execute-task` to implement this plan.

## Objective

service skeleton が吐く match artifact と lifecycle state を、online platform の read/write model として扱える形に整理する。
最初のゴールは、result 参照、audit、resume/replay の入口を壊さずに持てる persistence contract を定義することに置く。

## Context

- `record` / `event_log` / `snapshot` / `exported_snapshot` の artifact 契約はすでにあるが、online service の保存モデルとしてはまだ閉じていない
- Phase 6 では公開向け state と source-of-truth artifact の両方を platform 側で保持する前提を置いている
- service skeleton を先に通す方針のため、この plan は 01 の実 flow を受けて保存モデルを固める位置づけになる

## Scope

- match lifecycle state、artifact persistence、result/read model の最小 contract を定義する
- source-of-truth artifact と derived/public read model の責務分離を定義する
- 大規模 DB 設計、長期 retention policy、分析基盤はこの plan では扱わない
- この plan は親 plan として扱い、実装前に storage schema、artifact layout、read API などの child plan へ分解する

## Spec Changes

### `docs/specs/platform.md`

- persisted artifact と service read model の関係を補足する
- resume / replay / audit が依存する source-of-truth を明確化する

### 新規または更新 spec

- persistence model spec を追加し、match state / artifact / public state の保存単位を定義する
- result list / match detail / replay input が読む read model を定義する

## Expected Code Changes

- match lifecycle と artifact 保存を担う persistence layer
- result / match detail / replay input を返す read model builder
- source-of-truth artifact と derived/public state を切り分ける serialization / storage code
- persistence contract を検証する unit / integration / replay verification

## Sub-tasks

- [ ] service skeleton で実際に生成される保存対象を棚卸しする
- [ ] source-of-truth artifact と read model の分離方針を spec に落とす
- [ ] [parallel] resume / replay / audit の各利用者が必要とする保存単位を整理する
- [ ] 実装前に child plan へ分解し、storage-oriented な execution order を確定する

## Parallelism

- artifact 棚卸しと利用者別 read model 要件の整理は並行できる
- child plan 分解後は storage write path、read model、verification path を分けられる可能性がある

## Risks and Mitigations

- 保存モデルを先に細かく決めすぎると、01 の service skeleton 実装で余計な差し戻しが出る
  - mitigation: 01 の実 flow で必要になった保存面を起点にし、抽象度を上げすぎない
- public state と source-of-truth artifact の境界が曖昧だと spectator / audit 両方に悪影響が出る
  - mitigation: `record` 系正本と公開用 read model を別責務として明文化する

## Design Decisions

- persistence は service skeleton の実フローを受けて固める
- この plan 自体は parent/base plan であり、実装着手前に同じ `platform-online-foundation-02` 系の child plan へ分割する前提とする
