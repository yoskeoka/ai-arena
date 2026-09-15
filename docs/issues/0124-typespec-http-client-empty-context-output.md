# TypeSpec HTTP client emitter が空 client の未使用 context を出力する

## 状況

TypeSpec-generated operator client は、operation を持たない `AiArenaClient` と `SharedClient` に private
context field を出力する。この field は参照されないため、operator UI の strict TypeScript build が `TS6133`
で失敗する。

0134 の staging release 復旧では、current pin の generated artifact から未使用 field と初期化だけを除く
最小 correction を適用した。`noUnusedLocals` は変更せず、operator/public HTTP wire contract および runtime
behavior は変更していない。

## 再現と version evidence

2026-09-16 に npm registry で確認し、次の互換 line を新規 install と compile に使用した。

- `@typespec/compiler` / `@typespec/http` / `@typespec/openapi3`: `1.16.0`
- `@typespec/rest`: `0.86.0`
- `@typespec/http-client-js`: `0.16.2`
- generated runtime: `@typespec/ts-http-runtime` `0.3.9`

emitter の peer dependency は compiler/http/rest と上記 line に整合していた。しかし
`pnpm --dir typespec run build` の後も `AiArenaClient.#context` と `SharedClient.#context` が再出力され、
`pnpm --dir operator-ui run build` は各 field の `TS6133` で失敗した。

## Upstream 関連

- [microsoft/typespec#8479](https://github.com/microsoft/typespec/issues/8479) は empty client/options の
  generation を扱う open issue であり、empty client の出力方針という根本領域が関連する。
- 同 issue は `emitter:client:csharp` の issue で、JavaScript emitter が unused context を出力して
  `TS6133` になる本件を直接は扱わない。2026-09-16 に title/body で
  `http-client-js unused context`、`TS6133`、`unused private context`、`client context` を検索したが、
  同一の公開 issue は確認できなかった。

## 次の判断

upstream emitter が empty client を strict compile 可能に出力する version、または TypeSpec source で
generated client を空にしない contract-level の設定が確認できるまで、emitter/runtime upgrade、generated
drift gate、staging prerequisite の導入は行わない。generated file への恒久手編集、emitter 単独更新、
TypeScript strictness の緩和は採らない。

採用候補が得られたら、clean install → TypeSpec compile → generated tracked/untracked drift check → operator UI
strict build の順で再評価し、成功時に専用 CI / staging release gate を別の実行 plan で導入する。
