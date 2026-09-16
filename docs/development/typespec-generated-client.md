# TypeSpec generated client の運用

TypeSpec HTTP contract の source of truth は `typespec/` である。OpenAPI と operator UI client は
その build artifact であり、generated client を直接編集してはならない。

## Build-owned postprocess

`pnpm --dir typespec run build` は compile の直後、format の前に唯一の defined postprocess を実行する。
これは upstream の `@typespec/http-client-js` が operation を持たない client に出力する strict
TypeScript 非互換の context を取り除く暫定措置である。wire contract、公開 operation、runtime behavior、
TypeScript strictness を変更するものではない。

postprocess は emitted `aiArenaClient.ts` の `AiArenaClient` と `SharedClient` にある既知の empty-client
shape だけを受理する。対象 context の import、private field、initializer は一組として除去する。
`OperatorClient`、`PublicClient`、operation、model import、endpoint/options の利用形態は対象外である。

入力が既知 shape と一致しない、必要 token がないか重複する、または処理後に target token が残る場合は
nonzero で停止する。既に upstream により修正済みの output も no-op 成功にはしない。upstream 修正を採用する
ときは clean regeneration と strict build を確認し、別の execution plan で postprocess の撤去を判断する。

## Verification と release gate

`make verify-typespec-generated-client` は次を同一の順序で実行する。

1. TypeSpec workspace の frozen install と build
2. OpenAPI と generated client の tracked/untracked drift check
3. operator UI の frozen install と strict build

TypeSpec source、postprocessor、generated artifact、TypeSpec/runtime dependency、またはこの verifier/gate を
変える PR は dedicated CI gate の対象である。staging release も同じ変更集合では target SHA の successful
`typespec-generated-client` push run を確認してから Pages build を開始する。比較または run lookup ができない、
missing、pending、failure のいずれも deploy を開始しない。workflow dispatch はこの prerequisite を bypass しない。
