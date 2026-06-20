# oauth-library-adoption
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`docs/issues/0032-replace-hand-rolled-oauth-oidc-with-supported-libraries.md`
を解決するため、current GitHub login 実装の hand-rolled OAuth 部分を
supported library ベースへ置き換える execution plan を定義する。

この plan のゴールは、current browser route / session cookie / invite gate の
外部 contract を保ったまま、backend auth 実装を

- GitHub login は `golang.org/x/oauth2`
- 将来の Google login を含む OIDC provider は
  `github.com/coreos/go-oidc/v3/oidc` と provider sub package

で素直に扱える内部境界へ寄せることにある。

Google login 自体はこの plan の対象外だが、導入時に callback / token exchange /
claim normalization / store bind を大改造しなくてよい構造をこの段階で固定する。

Addresses: `docs/issues/0032-replace-hand-rolled-oauth-oidc-with-supported-libraries.md`

## Context

- `docs/specs/platform-product-auth.md` は current public auth contract を
  GitHub login first で定義している
- `docs/exec-plan/done/0087-product-auth-and-gated-signup-01-github-login-and-account-linking-foundation.md`
  により、account / identity / invite / session の durable contract はすでに固定されている
- current backend は `internal/platform/service/auth.go` と
  `internal/platform/service/auth_github.go` で GitHub authorize URL 組み立て、
  token exchange、user profile fetch を hand-rolled に持っている
- current `account_identity` schema は `provider` と `provider_subject` を分離しており、
  provider-agnostic identity 正本としては十分なので、今回の主眼は
  transport / provider integration boundary の整理にある

## Option Snapshot

### Option A: GitHub だけ `oauth2` 化し、OIDC 対応は後で別設計にする

- 利点:
  変更量が最小
- 欠点:
  Google login 追加時に provider abstraction、callback state、
  token / claim validation の責務をもう一度剥がす必要がある

### Option B: route contract は GitHub first のまま、backend 内部を
`oauth2` + optional OIDC verifier の provider boundary に整理する

- 利点:
  current GitHub behavior を保ちつつ、OIDC provider 追加時の差分を
  verifier / claim mapping / config 追加へ限定できる
- 欠点:
  今回の GitHub-only 変更としては abstraction が 1 段増える

### Option C: 今から auth 全体を OIDC 中心へ再設計する

- 利点:
  OIDC provider 追加は最も素直
- 欠点:
  GitHub は OIDC provider ではないため不自然で、
  current GitHub first contract に対して過剰な再設計になる

## Recommendation

- Option B を採用する
- public route は current の `/auth/github/login` と
  `/auth/github/callback` を維持する
- backend 内部では provider ごとの authorize URL generation と code exchange を
  `oauth2.Config` ベースへ寄せる
- OIDC provider 向けには `oidc.Provider` / `oidc.Verifier` /
  provider sub package を差し込める verifier seam を先に定義する
- GitHub provider は ID token を前提にせず access token + userinfo fetch のまま扱い、
  OIDC provider だけが ID token validation を要求する capability split を明記する

## Scope

- GitHub login backend の supported library 移行方針を定義する
- provider-generic auth flow と provider-specific claim resolution の境界を定義する
- 将来の Google login 追加で再利用する OIDC verifier seam を定義する
- current session / invite / role / account_identity contract を壊さない移行順序を定義する
- provider bootstrap / env inventory を library-backed auth 前提へ更新する

この plan では以下を扱わない。

- Google login の実装
- frontend login UX の再設計
- role model や invite policy の再設計
- same-origin 化や custom domain 化そのもの

## Spec Changes

- `docs/specs/platform-product-auth.md`
  - current GitHub-first route contract を維持したまま、
    backend auth provider boundary を追記する
  - supported library 利用方針として、
    OAuth provider は `oauth2` で authorization URL / token exchange を扱い、
    OIDC provider は追加で ID token / claim validation を行う責務を明記する
  - provider-agnostic identity claim の最小 shape
    (`provider`, `subject`, `login/display name`, `email`) を定義する
- `docs/development/platform-service-online-deploy.md`
  - GitHub OAuth app bootstrap を `oauth2` 前提の secret / callback inventory として整理する
  - 将来の Google login 用に、OIDC issuer / client credentials /
    callback path の置き場を reserved inventory として追記する
- 必要なら `docs/specs/platform-frontend-architecture.md`
  - frontend が provider library を持たず backend redirect contract に依存する前提を補強する

## Expected Code Changes

- `internal/platform/service/auth.go`
  - GitHub 固有の pending state cookie 名や config field を
    provider-aware auth flow へ寄せる
  - `GitHubLogin` / `GitHubCallback` を provider boundary 越しに組み直す
- `internal/platform/service/auth_github.go`
  - hand-rolled token exchange を `oauth2` client へ置き換える
  - GitHub user profile fetch だけを provider-specific responsibility として残す
- 新規 provider boundary file 群
  - `oauth2.Config` を持つ provider descriptor
  - optional OIDC verifier / claim extractor interface
  - provider-normalized identity claim type
- `internal/platform/service/auth_postgres.go`
  - `provider='github'` 直書き前提を見直し、
    generic identity bind / lookup helper へ寄せる
- `cmd/arena-service/main.go`
  - auth config 組み立てを provider-aware env loading へ寄せる
- auth test 群
  - GitHub login redirect / callback / invite claim の behavior を維持する回帰テスト
  - OIDC verifier seam の unit test

## Sub-tasks

- [ ] current hand-rolled OAuth responsibility を棚卸しし、
  `oauth2` へ置き換える範囲と app responsibility を分ける
- [ ] provider-normalized identity claim と optional OIDC verifier seam を定義する
- [ ] GitHub provider を `oauth2` ベースへ置き換え、callback behavior を維持する
- [ ] auth store を provider-aware helper へ寄せ、`account_identity` bind の generic path を作る
- [ ] spec / deploy runbook に supported library 方針と future Google inventory を反映する
- [ ] redirect, invite, session, operator auth の回帰テストを追加・更新する

## Parallelism

- [parallel] spec / runbook 更新は、provider boundary 設計と並行で進められる
- [parallel] GitHub `oauth2` 化と auth store helper の整理は、
  identity claim shape が固まれば並行できる

## Dependencies

- depends on:
  `docs/exec-plan/done/0087-product-auth-and-gated-signup-01-github-login-and-account-linking-foundation.md`
- depends on:
  `docs/specs/platform-product-auth.md`
- depends on:
  `docs/issues/0032-replace-hand-rolled-oauth-oidc-with-supported-libraries.md`

## Verification

- GitHub login start が引き続き `/auth/github/login` から GitHub authorize へ redirect する
- GitHub callback が current invite / session behavior を壊さない
- auth-disabled mode と auth-enabled mode の session status contract が維持される
- provider abstraction 導入後も `account_identity(provider, provider_subject)` の正本 contract が維持される
- OIDC provider 向け test seam があり、Google login 実装前でも
  ID token verification の差し込み点を unit test で確認できる

## Risks and Mitigations

- GitHub 向け abstraction を作り込みすぎると current change が過剰になる
  - mitigation:
    route は GitHub first のまま固定し、generic 化は backend provider boundary に限定する
- OIDC verifier seam を今から広げすぎると未使用 config が増える
  - mitigation:
    この plan では optional interface と env inventory の予約に留め、
    Google login 実装そのものは deferred と明記する
- current Postgres lookup が `provider='github'` に依存しているため、
  provider 追加時に session principal 解決が歪む
  - mitigation:
    lookup helper を generic identity selection へ寄せ、
    session principal が linked identity を正しく返す contract をテストで固定する

## Design Decisions

- OAuth と OIDC を同一 protocol として雑に統一せず、
  `oauth2` transport と optional OIDC identity verification を分けて扱う
- GitHub login は `oauth2` のみを使う first provider とし、
  Google login は後続で OIDC verifier を伴う provider として追加する
- provider route は current public contract を壊さず、
  backend 内部の supported-library adoption で将来拡張を吸収する
