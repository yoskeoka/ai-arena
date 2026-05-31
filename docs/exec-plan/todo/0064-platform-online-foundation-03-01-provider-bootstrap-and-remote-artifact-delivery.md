# platform-online-foundation-03-01-provider-bootstrap-and-remote-artifact-delivery
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Phase 6 の online confirmation を始める前提として、`Render + Neon Postgres + Cloudflare R2 + Cloudflare Pages`
の provider bootstrap と secret/config 契約を固める。
最初のゴールは、signup / checkout / project 作成 / secret 登録の手順を独立に整理し、
artifact download を backend 中継ではなく remote object storage provider へ委譲する契約を固定することに置く。

## Context

- user decision として、backend process の first landing はまず `Render` で実際に動かして判断する
- `Neon Postgres` と `Cloudflare R2` へ state と artifact を分離しているのは、
  将来 `runner` だけ別 compute へ逃がす余地を残すためでもある
- UI は Phase 6 から外さず、まずは simple polling と最低限の artifact access を成立させる
- completed match の artifact access は必要だが、object bytes を `arena-service` が毎回中継すると
  remote object storage の強みと backend 軽量性を損ねる

## Scope

- `Render` / `Neon` / `Cloudflare R2` / `Cloudflare Pages` の最小 bootstrap 手順を定義する
- signup / checkout / plan 契約が必要な provider を洗い出し、独立した human task として明記する
- `arena-service` が持つ secret / config / environment variable contract を定義する
- completed artifact access は、storage provider が発行する短寿命 download URL または同等の delegated download token を
  backend が要求して返す形に固定する
- backend は artifact locator と delegated download metadata を返すだけに留め、artifact 本体をダウンロード時に通過させない

この plan では以下を扱わない。

- remote queue/run 自体の実装
- Pages UI 実装
- matchmaking / ranking / rerun policy
- provider 固有 SDK への恒久ロックイン

## Spec Changes

### `docs/specs/platform-service-persistence.md`

- artifact locator metadata と delegated download metadata の責務境界を追記する
- write model は download URL 本体を永続化せず、stable locator だけを保持することを明記する

### `docs/specs/platform-service-read-model.md`

- completed match detail が返してよい artifact access surface を追記する
- delegated download URL / token は短寿命の derived field であり、object bytes は backend を経由しないことを明記する

### 新規または更新 spec / development doc

- provider bootstrap と env/secret 契約をまとめた online service development/deploy doc を追加する
- Pages / Render / Neon / R2 それぞれで人手作業が必要な bootstrap 項目を列挙する

## Expected Code Changes

- remote object storage config loader
- delegated artifact download URL issuance interface
- provider bootstrap / local override configuration
- bootstrap 手順の verification script または checklist

## Sub-tasks

- [ ] provider ごとの signup / checkout / project 作成 / secret 登録を棚卸しする
- [ ] `Render` / `Neon` / `R2` / `Pages` に渡す env/secret contract を spec に落とす
- [ ] artifact access を locator 永続化 + delegated download issuance に分ける契約を定義する
- [ ] delegated download で object bytes が backend を通過しないことを spec に明記する
- [ ] human bootstrap が必要な項目を execution 前提の checklist として整理する

## Parallelism

- provider bootstrap 棚卸しと artifact access contract 整理は並行できる
- bootstrap 契約が固まった後は Render / R2 / Pages 側の設定作業を分担できる

## Dependencies

- depends on: parent/base item `0047-platform-online-foundation-03-operator-flow-and-matchmaking.md` (now retired to `docs/exec-plan/done/` after split)
- informs: `0065-platform-online-foundation-03-02-remote-service-topology-and-polling-api.md`
- informs: `0066-platform-online-foundation-03-03-minimal-operator-ui-and-artifact-access.md`

## Risks and Mitigations

- artifact access を backend proxy 前提で始めると outbound / bandwidth / CPU が余計に膨らむ
  - mitigation: provider-generated short-lived download URL または同等 token を返す契約に寄せ、object bytes は provider に配信させる
- provider bootstrap が implementation plan に埋もれると、人手作業の抜けで execution が止まる
  - mitigation: signup / checkout / secret 登録を独立 child plan の責務として先に明文化する
- provider 固有 URL shape を read model 正本にすると差し替えが難しくなる
  - mitigation: write model には stable locator だけを残し、download URL は request 時に派生させる

## Design Decisions

- first landing provider set は `Render + Neon Postgres + Cloudflare R2 + Cloudflare Pages` のまま維持する
- artifact delivery は backend 中継ではなく remote object storage provider へ委譲する
- delegated artifact access は provider-specific implementation を持ちうるが、
  read model contract は locator + derived delegated access metadata の二層に分ける
