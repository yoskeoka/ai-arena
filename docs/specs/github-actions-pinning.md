# GitHub Actions Pinning

この repository の GitHub Actions workflow file と composite action では、
`uses:` 参照を `pinact` で管理する。

## Operator Contract

- `.github/workflows/*.yml` または `.github/actions/**` を編集するときは、新規 `uses:` 参照の pin と既存 pinned 参照の更新に `pinact` を使う。
- `@v6` や `@stable` のような floating version tag を、action 依存更新のために手で書き換えてはいけない。`pinact` を実行して、その diff を review する。
- repository 全体ではなく一部だけ更新したいときは、`pinact run` に対象 file path を明示してよい。
- `.pinact.yaml` は必須ではない。明示 file 引数だけでは out-of-scope workflow file を避けられないときだけ追加する。
