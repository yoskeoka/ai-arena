# spec-responsibility-wording
**Execution**: Use `/execute-task` to implement this plan.

## Objective

`ai-arena` の platform 共通 spec について、現在の package / type / method 名をそのまま写した説明を減らし、
責務・境界・入力・出力・観測可能なふるまいを先に読むだけで contract が分かる wording に揃える。

この plan は以下を成立させる。

- `docs/issues/spec-should-describe-responsibilities-not-code-symbols.md` で指摘した drift を、registry 周辺だけでなく隣接 spec まで監査して修正する
- stable abstract concept として扱う名前だけを残し、単なる current implementation detail への依存を減らす
- 今後の spec/plan review で同種の drift を再発させにくい確認導線を docs 側へ残す

Addresses:

- `docs/issues/spec-should-describe-responsibilities-not-code-symbols.md`

## Context

- `docs/project-plan.md` は platform を複数 game 対応の共通基盤として育てる方針を取り、個々の current implementation detail より durable な contract 記述が重要になる
- `docs/design-decisions/core-beliefs.md` は AI-first と correctness over speed を要求しており、spec の主語も code structure ではなく contract を優先して整理する必要がある
- `docs/design-decisions/adr.md` にはこの issue 専用の新しい architecture decision はなく、今回は設計選択の変更ではなく spec の記述粒度と review 導線の整理として扱う
- `docs/lessons.md` にはすでに `[2026-05-08] spec では実装シンボル名より責務境界を書く` があり、execution ではこの rule を正本として隣接 spec を点検する
- 現状の `docs/specs/platform-game-registry.md`、`docs/specs/platform.md`、`docs/specs/game-master.md` には、責務ベースで書けている部分と、`DescriptorRecord` / `GameDescriptor` / `BuildSession` のような symbol 名へ寄りやすい部分が混在している

## Scope

- platform / runner / registry / game-master 周辺 spec の wording 監査
- concrete symbol 名を使う箇所の妥当性整理
- spec review 時に使う軽量な確認導線の追加
- issue close に必要な docs 更新と issue file move

この plan では以下は扱わない。

- registry / runner / game-master の code 実装変更
- type/interface 名そのものの rename
- architecture policy を追加するための ADR 新設
- game 固有 spec の全面 rewrite

## Design Decision

この plan では追加 ADR は作らない。

- durable rule の正本は既存の `docs/lessons.md` とし、新しい architecture decision には昇格しない
- spec 本文ではまず責務・境界・入力・出力・観測可能なふるまいを書く
- concrete symbol 名は、stable abstract concept として reader がその名で contract を認識する場合にだけ残す
- review 用の確認導線は architecture policy ではなく docs/specs 側の軽量な checklist として置く

## Spec Changes

### `docs/specs/platform-game-registry.md`

- registry key、persisted metadata、lookup responsibilities、build/replay entrypoint の説明を、現行 symbol 名の列挙ではなく責務中心に言い換える
- `DescriptorRecord`、`GameDescriptor`、`BuildSpec` などを残す箇所では、それらが current package layout ではなく contract 上の安定概念として必要な理由を揃える
- symbol 名が不要な文脈では、store / resolver / descriptor / build entrypoint の責務表現へ戻す

### `docs/specs/platform.md`

- runner / registry / game build 入口の責務分離説明から、current implementation に引きずられた言い回しを除く
- artifact / replay/debug / compatibility 記述でも、game 固有 helper や method 名を spec の主語にしない
- platform の責務と非責務が code symbol を知らなくても読める wording に揃える

### `docs/specs/game-master.md`

- game master の論理 API 記述で、stable contract と implementation detail の線引きを揃える
- platform との責務境界、metadata compatibility、snapshot/result 契約を、実装 package 名がなくても解釈できる表現に寄せる

### `docs/specs/README.md`

- spec review 用の lightweight checklist を追加し、少なくとも「その symbol 名は contract 上必須か」「責務・入出力・観測可能なふるまいへ言い換えられないか」を確認できるようにする

### Optional touch if audit finds wording drift elsewhere

- `docs/specs/platform-common-contract.md` や `docs/specs/ai-runtime.md` に同種の drift が見つかった場合だけ、同じ rule に沿って最小限修正する

## Expected Code Changes

なし。execution は docs のみを変更する。

## Sub-tasks

- [ ] Audit `docs/specs/platform-game-registry.md`, `docs/specs/platform.md`, and `docs/specs/game-master.md` for wording that depends on current code symbols more than contract responsibilities
- [ ] Rewrite the identified passages so responsibility, boundaries, inputs/outputs, and observable behavior remain primary
- [ ] Decide which existing symbol names are stable abstract concepts worth keeping in spec prose and remove the rest from the audited passages
- [ ] Add a lightweight spec-review checklist to `docs/specs/README.md`
- [ ] If the audit finds the same drift in `docs/specs/platform-common-contract.md` or `docs/specs/ai-runtime.md`, apply the same wording rule there
- [ ] Move `docs/issues/spec-should-describe-responsibilities-not-code-symbols.md` to `docs/issues/done/` when the audited docs and checklist fully cover the issue intent

## Parallelism

- [parallel] `platform-game-registry.md` の wording audit と `platform.md` / `game-master.md` の wording audit は独立して進められる
- [parallel] `docs/specs/README.md` の checklist 追加は、主要な wording 方針が固まった後なら他の spec 修正と独立して進められる
- issue close の判断は、監査対象 spec と checklist の両方が揃ってから行う

## Risks and Mitigations

- symbol 名を消しすぎると、reader がどの概念を安定 contract として共有しているのか逆に分かりにくくなる
  - mitigation: abstract concept として意味が固定されている名前だけは残し、なぜ残すのかを周辺文脈で明確にする
- wording 監査のつもりが architecture や runtime behavior の再設計へ広がる
  - mitigation: contract 自体は変えず、説明の主語と粒度だけを揃える。behavior change が必要なら別 issue に切り出す
- checklist を重くしすぎると docs review で使われなくなる
  - mitigation: 2-3 個の確認点に絞り、`docs/specs/README.md` の既存案内に自然に置く

## Verification

The execution PR is complete when the following are true.

- `docs/specs/platform-game-registry.md`, `docs/specs/platform.md`, and `docs/specs/game-master.md` の対象箇所が、current implementation symbol を知らなくても contract を読める wording になっている
- concrete symbol 名を残した箇所は、stable abstract concept として必要なものに限定されている
- `docs/specs/README.md` に、同種の drift を review で検出するための lightweight checklist が追加されている
- `docs/lessons.md` の既存 rule と audited spec 群の記述方針が矛盾していない
- `docs/issues/spec-should-describe-responsibilities-not-code-symbols.md` を `docs/issues/done/` へ移せるだけの coverage が PR diff で確認できる
