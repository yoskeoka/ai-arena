# staging-owned-bot-list-persistence

## Summary

staging の `/operator/requests` では、`reversi-v1-standard` に対して登録済みと認識して
いる2つの bot が eligible seat として表示される。一方、同じアカウントで確認した
`/operator/submissions` の `Your bots` には bot が表示されず、次の文言が表示される。

```text
Enter a competition scope to list bots.
```

これは、scope-wide の composition projection と、owner-scoped の bot list が同じ
永続化された bot/revision 情報を別の条件で参照している現象として起票する。現時点では
artifact bytes の消失や特定の lookup failure までは断定しない。

## Observed Evidence

- 2026-09-10、`/operator/requests` で `reversi-v1-standard` を選ぶと、2席に
  `reversi-bot-2` と `普通リバーシbot` を割り当てられた。
- `/operator/submissions` の `Your bots` では、同じアカウントであるにもかかわらず、
  上記 bot が表示されなかった。
- requests 側の eligible projection は、scope 全体から `active` bot と `ready` な
  active revision を取得する。owner account の条件はない。
- Your bots 側の list は、認証 principal の account ID、入力された scope ID、
  `include_retired=true` を条件に owner-scoped query を実行する。
- 現在の表示文言は、scope が空のときにも表示されるため、画面表示だけでは API が空の
  response を返したのか、scope が未入力だったのかを区別できない。

## Current Boundary and Cause Candidates

`SubmissionsPage` は初期状態で competition scope を空にし、空の場合は bot list API を
呼ばずに上記の文言を表示する。scope が入力されている場合は、owner-scoped の
`/api/v1/bots` を呼ぶ。

requests 側の `/api/v1/eligible-bots` は scope-wide であり、DB の bot と active
revision が存在し、revision が `ready` であれば候補に含める。match 作成時には別の
owner 条件なしの scope query で選択された bot の active revision と `artifact_id` を
解決する。

そのため、今回の現象に対する候補は次のとおりである。

1. submissions 画面の scope input が空、または登録時と異なる scope ID である。
2. `/api/v1/bots` が参照する authenticated principal の account ID と、bot登録時に
   `owner_account_id` として保存された値が一致していない。
3. owner-scoped list と scope-wide eligible projection が、異なる DB/schema、または
   異なる永続化状態を参照している。
4. owner-scoped list API は bot を返しているが、frontend の response transform または
   render で失われている。

同じアカウントであることが確認済みなら、候補1の scope state、候補2の principal と
`owner_account_id` の対応、候補3の永続 store lookup を優先して切り分ける。ただし、
requests に bot が表示されたことだけでは、owner-scoped list が同じアカウントの bot を
返すことまでは証明できない。

この一覧表示の経路は bot artifact bytes の存在確認を行わないため、今回の表示欠落だけ
から R2 上の bot artifact 消失とは判断しない。また、同時に発生した match request error
はゲーム側 artifact registry lookup で先に失敗しており、bot artifact lookup の結果を
示すものではない。

## Follow-up Boundary

この issue では修正を実装しない。次の follow-up で、同じ認証セッションについて
owner-scoped list の request scope、principal account ID、response items、DB の
`owner_account_id`/`scope_id`/active revision relation を相関させ、現象が UI 表示だけか、
永続情報の lookup 不整合かを確定する。

