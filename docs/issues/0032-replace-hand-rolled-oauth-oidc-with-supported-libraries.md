# hand-rolled OAuth/OIDC 実装を supported library へ置き換える

## Summary

current backend auth 実装は GitHub login flow の一部を
repo 内で hand-rolled に組んでいる。

OAuth / OIDC は security-sensitive な protocol なので、
`golang.org/x/oauth2` や `github.com/coreos/go-oidc` のような
広く使われている library を前提に組み直すべき。

## Scope

- backend 側の OAuth / OIDC flow 実装を対象とする
- current 対象ファイル例:
  - `internal/platform/service/auth.go`
  - `internal/platform/service/auth_github.go`
- frontend は当面対象外とする
  - current frontend は session cookie の中身を扱わず、
    session status を問い合わせて route guard しているだけなので、
    backend 側の auth/session contract が正しければ追加 library 導入は不要

## Why this matters

- OAuth / OIDC の protocol detail を app 側で持つほど
  state / redirect / token handling の security bug を埋め込みやすい
- provider 拡張時に hand-rolled logic を増やすと、
  GitHub 専用の分岐や provider 差分吸収が散らばりやすい
- supported library に寄せた方が review しやすく、
  future maintenance でも protocol 更新を追いやすい

## Follow-up direction

- GitHub login の authorization URL generation と token exchange を
  `golang.org/x/oauth2` ベースへ置き換える
- OIDC provider を今後追加する可能性を考え、
  GitHub 固有 logic と provider-generic auth flow を切り分ける
- OIDC 対応 provider を入れるタイミングで
  `github.com/coreos/go-oidc` を使った ID token / claim validation へ寄せる
- current cookie/session ownership, invite gating, operator role check は
  backend app responsibility として維持する

## Non-goals for this follow-up

- frontend auth architecture の再設計
- session cookie contract の frontend-side library 化
- current PR の auth behavior 変更を伴う immediate rewrite
