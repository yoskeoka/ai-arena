# platform-online-foundation-03-01-provider-bootstrap-and-remote-artifact-delivery
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Phase 6 の online confirmation を始める前提として、`Render + Neon Postgres + Cloudflare R2 + Cloudflare Pages`
の provider bootstrap と secret/config 契約を固める。
最初のゴールは、必要な provider の signup / checkout / project 作成 / secret 発行を実際に完了できる execution scope を固定し、
artifact download を backend 中継ではなく remote object storage provider へ委譲する契約を固定することに置く。

## Context

- user decision として、backend process の first landing はまず `Render` で実際に動かして判断する
- `Neon Postgres` と `Cloudflare R2` へ state と artifact を分離しているのは、
  将来 `runner` だけ別 compute へ逃がす余地を残すためでもある
- UI は Phase 6 から外さず、まずは simple polling と最低限の artifact access を成立させる
- completed match の artifact access は必要だが、object bytes を `arena-service` が毎回中継すると
  remote object storage の強みと backend 軽量性を損ねる

## Provider Targets

### `Render`

- top / pricing:
  - `https://render.com/pricing`
  - `https://render.com/free`
  - `https://render.com/docs/outbound-bandwidth`
- recommended contract for first landing:
  - workspace plan は `Hobby`
  - `arena-service` は free web service ではなく paid web service instance を前提候補にする
  - 理由: free web service は 15 分 idle で spin down し、online confirmation の常時 polling / active match 確認に向かない
- bootstrap outputs to record:
  - Render workspace
  - web service id / service URL
  - deploy method（GitHub repo 連携か image deploy か）
  - environment variable names
- notable constraints as of 2026-06-01:
  - free web service は 15 分 idle で spin down する
  - Render は outbound bandwidth を workspace 単位で課金対象にしうる
  - paid instance は spin down しない

### `Neon Postgres`

- top / pricing:
  - `https://neon.com/pricing`
  - `https://neon.com/docs/introduction/pro-plan`
  - `https://neon.com/docs/introduction/network-transfer`
- recommended contract for first landing:
  - まずは `Free`
  - 連続稼働や転送量で free allowance を超える兆候が出たら `Launch` へ上げる
- bootstrap outputs to record:
  - project id
  - branch id
  - database name
  - pooled connection string
  - direct connection string
  - service role / password secret names
- notable constraints as of 2026-06-01:
  - Free は no credit card required
  - Free は project あたり `100 CU-hours/month`, `0.5 GB storage/project`
  - Free の public network transfer は `5 GB/month` included で、超過時は compute suspend の可能性がある

### `Cloudflare R2`

- top / pricing:
  - `https://developers.cloudflare.com/r2/get-started/`
  - `https://developers.cloudflare.com/r2/pricing/`
  - `https://developers.cloudflare.com/billing/usage-based-billing/`
- recommended contract for first landing:
  - `R2 subscription` checkout を完了する
  - storage class はまず standard
  - access method は first landing では S3-compatible API を優先候補にする
- bootstrap outputs to record:
  - account id
  - bucket name
  - bucket region / endpoint
  - access key id
  - secret access key
  - token / key の secret names
- notable constraints as of 2026-06-01:
  - R2 利用開始には Cloudflare account に対する `R2 subscription` が必要
  - free included usage は `10 GB storage`, `1M Class A`, `10M Class B`
  - egress to Internet は free だが、operation count と storage は監視が必要

### `Cloudflare Pages`

- top / pricing:
  - `https://developers.cloudflare.com/pages/platform/limits/`
  - `https://developers.cloudflare.com/pages/functions/pricing/`
  - `https://developers.cloudflare.com/workers/platform/pricing/`
- recommended contract for first landing:
  - static UI hosting は `Free`
  - Pages Functions は first landing では必須にせず、必要なら Workers Free/Paid の制約で扱う
- bootstrap outputs to record:
  - Pages project name
  - production URL
  - build command
  - output directory
  - environment variable names
- notable constraints as of 2026-06-01:
  - Pages Free は `500 builds/month`
  - Pages Functions は Workers pricing に従い、Free は `100,000 requests/day`, `10 ms CPU/invocation`
  - static asset requests 自体は Free で unlimited 扱い

## Scope

- `Render` / `Neon` / `Cloudflare R2` / `Cloudflare Pages` について、execution 時に実際に必要な signup / checkout / project 作成 / secret 発行の対象を固定する
- provider ごとに、どの plan / contract を first landing に採るかを明記する
- provider ごとに、repo 側へ登録すべき ID / secret / URL / endpoint 名を明記する
- provider ごとに、現時点で online arena 運営が受ける制約
  （spin-down、request limit、bandwidth / transfer、storage / operation 上限）を明記する
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

- online service development/deploy doc に、provider ごとの入口 URL、採用 plan、登録すべき ID / secret / endpoint 名、制約一覧を追加する
- signup 手順の click-by-click 手順ではなく、repo が前提とする provider asset inventory を列挙する

## Expected Code Changes

- provider asset inventory を保持する development/deploy doc
- remote object storage config loader
- delegated artifact download URL issuance interface
- provider bootstrap / local override configuration
- 必要なら bootstrap 後の config 検証 helper

## Sub-tasks

- [ ] `Render` の workspace / web service 契約を確定し、service URL と env var inventory を記録する
- [ ] `Neon` の project / branch / database / connection string inventory を記録する
- [ ] `Cloudflare R2` の subscription checkout、bucket、access credential inventory を記録する
- [ ] `Cloudflare Pages` の project / build / env inventory を記録する
- [ ] `Render` / `Neon` / `R2` / `Pages` に渡す env/secret contract を spec に落とす
- [ ] provider ごとの current limit / pricing constraints を repo-local doc に転記する
- [ ] artifact access を locator 永続化 + delegated download issuance に分ける契約を定義する
- [ ] delegated download で object bytes が backend を通過しないことを spec に明記する

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
  - mitigation: signup / checkout / secret 登録そのものをこの child plan の execution scope とし、repo に必要 asset inventory だけを残す
- provider 固有 URL shape を read model 正本にすると差し替えが難しくなる
  - mitigation: write model には stable locator だけを残し、download URL は request 時に派生させる

## Design Decisions

- first landing provider set は `Render + Neon Postgres + Cloudflare R2 + Cloudflare Pages` のまま維持する
- artifact delivery は backend 中継ではなく remote object storage provider へ委譲する
- delegated artifact access は provider-specific implementation を持ちうるが、
  read model contract は locator + derived delegated access metadata の二層に分ける
