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

- [microsoft/typespec#11978](https://github.com/microsoft/typespec/issues/11978) は JavaScript client
  emitter が empty client の unused context を出力して `TS6133` になる本件の upstream bug である。
- [microsoft/typespec#8479](https://github.com/microsoft/typespec/issues/8479) は empty client/options の
  generation を扱う open issue であり、empty client の出力方針という根本領域が関連する。
- 同 issue は `emitter:client:csharp` の issue で、JavaScript emitter が unused context を出力して
  `TS6133` になる本件を直接は扱わない。2026-09-16 に title/body で
  `http-client-js unused context`、`TS6133`、`unused private context`、`client context` を検索したが、
  同一の公開 issue は確認できなかったため #11978 を起票した。

## 採用判断

generated file への恒久手編集、emitter 単独更新、TypeScript strictness の緩和は採らない。一方で、0136 は
TypeSpec build に所有された決定的 postprocess を導入する。これは `AiArenaClient` と `SharedClient` の既知の
empty-client context/import/initializer にだけ限定し、shape の変化・不足・重複・既修正 output は fail-closed
にする。従って contributor、CI、staging release は手作業ではなく同じ regeneration output を検証する。

#11978 は upstream blocker として継続監視する。upstream emitter が strict-compile-safe な output を提供した
時点で、clean install → TypeSpec build → generated tracked/untracked drift check → operator UI strict build を
行い、postprocess を撤去するかは別 execution plan で判断する。この postprocess は upstream fix を代替または
偽装するものではない。
