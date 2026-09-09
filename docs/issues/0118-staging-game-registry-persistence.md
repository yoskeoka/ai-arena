# staging-game-registry-persistence

## Summary

staging の `/operator/games` では、admitted game release と competition scope が
`ready` として表示される。一方で同じ scope から `/operator/requests` で match
request を作成すると、ゲーム側の artifact digest に対して次の admission error が
返る。

```text
service: bad request: registry: unsupported artifact "sha256:98bd46609016dc763bcbfff747c6705c7f1608a86d164d77b38db208d0d1c0df"
```

`/operator/games` に表示された `builder` と `artifact` の digest は、この error の
digest と一致する。したがって、ゲーム登録情報の一覧表示と、match admission 時の
実行可能 descriptor 解決が同じ durable state を参照できていない可能性がある。

## Observed Evidence

- 2026-09-10、`reversi-v1-standard` が `/operator/games` に表示された。
- 表示内容は `reversi@1.0.0 / standard`、`build: wasm-wasi`、`builder: artifact/<digest>`、
  `artifact: <digest>` だった。
- 同じ scope の `/operator/requests` では、2つの eligible bot を選択できたが、match
  request 作成時に上記の `registry: unsupported artifact` が返った。
- staging backend の `/version` は `55c837706a0e11ed811702b56e8f09a85e27c8a2` で、
  現在の `origin/main` と一致していた。
- Render の staging service は単一インスタンスである。したがって、複数インスタンス間の
  routing 差だけでは説明しない。ただし、単一インスタンスでも restart/redeploy により
  プロセス内 state は失われる。

## Current Boundary and Cause Candidates

現在の実装では、game bundle admission は bundle bytes を artifact store に保存した後、
admitted descriptor record を process-local registry に登録する。起動時には新しい
WASI overlay registry が作られるが、Postgres に保存された game release をその registry
へ再構築する処理は確認できない。

一方、game list/get は Postgres の `game_releases` と `competition_scopes` から
`artifact_id` を取得するだけで、registry での実行解決を検証しない。match request は
保存された game の artifact identity を submission に設定し、queue admission 前に
registry の exact artifact lookup を行う。この順序により、registry record が失われて
いてもゲーム一覧は `ready` のままになり得る。

最有力の候補は、game registration 後の staging service restart/redeploy によって
process-local registry record だけが失われた状態である。単一インスタンスであることは
この候補を除外しない。

R2 上の bundle bytes が消失している可能性も残るが、今回の error は bundle store の
読み込み・materialize より前の registry lookup で発生している。そのため、今回の観測
だけから R2 上の bytes 消失を直接の原因とは判断できない。

## Follow-up Boundary

この issue では修正を実装しない。registry lookup は永続 store を source of truth として
扱い、メモリは必要に応じた cache として使う方向を望ましい follow-up の前提とする。
具体的な plan では、少なくとも次を決める。

- 永続 game release から descriptor を解決する責務と lookup 境界
- descriptor のうち DB に保存する情報と、artifact manifest から読む情報
- メモリ cache の key、失効、immutable release との整合性
- restart/redeploy 後、既存 release と過去 match の digest を解決できること
- R2 の bytes、Postgres の release record、registry cache の partial failure 観測

