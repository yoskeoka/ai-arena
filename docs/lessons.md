# Lessons

## [2026-05-12] 再発調査用 issue には失敗 output を残す

- Mistake: unrelated regression を `docs/issues/` へ切り出したとき、失敗テスト名と推測だけを書き、実際の assertion や event log 抜粋を十分に残さなかった
- Pattern: 「後で main で再現すればよい」と考えて、その場でしか取れない failure output を durable note に落としきらない
- Rule: 偶発かもしれない test/runtime failure を `docs/issues/` へ記録するときは、少なくとも実行コマンド、失敗した test 名、代表 assertion、再調査に効く output 抜粋を同じ issue に残す。完全ログの保存先があるならその path も書く
- Applied: `docs/issues/done/arena-runner-e2e-init-regression.md`、今後の ai-arena verification blocker 切り出し全般

## [2026-05-18] ai-arena spec を consumer 向け開発ガイド化しない

- Mistake: `docs/specs/platform.md` と `docs/specs/game-master.md` に、ai-arena が提供する契約そのものではなく、external repo 側の開発・ownership・verification 運用まで混ぜて書きかけた
- Pattern: platform spec が説明すべき「提供する外形契約」と、別 repo の利用者向けガイドや migration 文脈を同じ文書で扱ってしまう
- Rule: ai-arena の spec は、platform / runner / SDK が何を提供し、どの共通契約を固定するかだけを簡潔に書く。external game repo の開発方法、asset 配置、ownership、verification 運用は別の guide / plan / migration note に分離する
- Applied: `docs/specs/platform.md`、`docs/specs/game-master.md`、今後の external repo 前提の spec wording 全般

## [2026-05-28] docs-only follow-up で issue 追記するだけなら quality gate を回し直さない

- Mistake: CI failure を `docs/issues/` に記録して PR へ含めるだけの follow-up でも、直前の code verification と同じ感覚で local test / lint を再実行し続けかねない
- Pattern: 「PR に追加 commit を積む」ことだけを見て、変更内容が docs-only かどうかを quality gate 判断に反映できていない
- Rule: failure note や issue logging だけの docs-only follow-up では、対象 code を変えていない限り local test / lint を回し直さない。必要なら既存の verification 結果を保持したまま docs diff だけ commit する
- Applied: `0056` PR follow-up での flaky CI issue 追記、今後の docs/issues 追加だけを行う review follow-up 全般

## [2026-05-31] spec には現在の契約だけを書き、plan の達成文脈を混ぜない

## [2026-06-13] docs runbook の cross-reference と command sample は省略しすぎない

- Mistake: internal surface protection runbook で後続 plan 参照を `0080-...` の省略表記にし、staging access runbook の `curl /healthz` もベース URL を省いたまま PR に出した
- Pattern: 自分には文脈で補完できる plan 名や command を docs に書くとき、reviewer や次の実行者がそのまま辿れる粒度まで concrete に書く確認を省きやすい
- Rule: docs runbook で別 plan/file を参照するときは、実在する file path か相対 link で書く。command sample は copy-paste でそのまま実行できる形を基本にし、hostname や path の文脈補完を reader に委ねない
- Applied: `docs/development/platform-service-online-deploy.md` の plan 参照、access runbook、今後の exec-plan/runbook cross-reference と operational command sample 全般

## [2026-07-04] wire contract は markdown ではなく schema source に寄せる

- Mistake: API contract の詳細を Markdown spec に長く残しすぎると、関連 file を全文読みに近い形で追う必要が出てしまった
- Pattern: 仕様書を人間向けの explanation と machine-readable contract の両方にしてしまい、重複と token cost を増やす
- Rule: `docs/specs/` は behavior と boundary に絞り、request/response/route の正本は repo-owned schema source と generated artifact に分離する
- Applied: `docs/specs/platform-service-operator-api.md`、`docs/specs/platform-service-operator-ui.md`、今後の ai-arena API contract 文書全般
