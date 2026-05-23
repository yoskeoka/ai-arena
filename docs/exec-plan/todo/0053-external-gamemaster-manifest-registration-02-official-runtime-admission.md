# external-gamemaster-manifest-registration-02-official-runtime-admission
**Execution**: Use `/execute-task` to implement this plan.

## Objective

official registered game として受け付ける game master runtime kind と admission contract を拡張し、
開発用 local subprocess 入口と正式 registration policy を分離して定義する。
この plan の到達点は、どの runtime kind を正式登録で許可・拒否するかを security / operability /
portability の観点で比較し、platform contract と registration policy を spec に固定することに置く。

## Context

- `0052` は dev-only overlay として local subprocess fresh run を成立させる
- project-plan と既存 ADR の WASM 方針は AI player の提出形式に関するものであり、
  game master runtime kind にそのまま自動適用されるわけではない
- external game master の正式 registration では、単純な executable 実行を許すと host 権限の隔離が弱くなりやすい
- user decision:
  - local subprocess executable は開発用 runner entry としては許可する
  - official registered game の registration policy として executable をそのまま許可するかは別途判断する
  - 採用 runtime kind の最終判断はこの plan の execution で改めて比較する

## Scope

- official registered game で許可する runtime kind 候補を比較する
- runtime kind ごとの isolation / deployment / observability / portability / local-dev cost を整理する
- formal registration policy と dev-only overlay policy の境界を spec に明記する
- registry / registration contract に必要な metadata surface を整理する
- 必要なら future external adapter との関係を整理する

この plan では以下を扱わない。

- `0052` の dev-only manifest overlay 実装
- 実際の Docker runtime adapter や WASM/WASI runtime adapter の本実装
- online registration API / approval workflow / DB schema
- replay / resume entrypoint の追加

## Spec Changes

### `docs/specs/platform-game-registry.md`

- official registered game の runtime kind / builder metadata surface を補足する
- dev-only overlay path と official registration path が別 policy であることを明記する
- official registration では admission policy に通った runtime kind だけが persisted record 化されることを明記する

### `docs/specs/game-master.md`

- game master runtime kind の候補と required contract surface を整理する
- dev-only local subprocess と official runtime kind の責務差を明記する
- security boundary 上、単純 executable を official registration で許可しない場合の rationale を記録する

### `docs/specs/platform.md`

- platform が扱う game master runtime topology を整理する
- local development path と official operation path の違いを明記する

### `docs/design-decisions/adr.md`

- official registered game に許可する runtime kind と、その理由を記録する

## Expected Code Changes

- この plan の主成果物は spec / ADR であり、コード変更は最小に留めてよい
- 必要なら runtime kind enum / validation comments / placeholder contract tests を追加する

## Sub-tasks

- [ ] official game master runtime kind の候補を列挙する
- [ ] executable / Docker / WASM-WASI / future external adapter を比較する
- [ ] dev-only overlay と official registration policy の境界を定義する
- [ ] official registration に必要な metadata / admission surface を定義する
- [ ] ADR に採用方針と却下理由を記録する

## Parallelism

- runtime kind の比較整理と spec 文面の下書きは並行で進められる
- ADR 記録は採用方針が固まった後に進める

## Dependencies

- informed by: `0052-external-gamemaster-manifest-registration-01-dev-runner-overlay.md`
- informs: `0054-external-gamemaster-manifest-registration-03-resume-replay-follow-up.md`

## Risks and Mitigations

- AI player 用の WASM decision を game master へそのまま誤適用する
  - mitigation: player runtime と game master runtime の threat model と責務差を明示して比較する
- policy 比較なしで executable 拒否だけを先に決めて、将来の運用制約を見落とす
  - mitigation: isolation だけでなく deploy cost、debuggability、consumer repo での開発コストも比較表に含める
- spec で runtime kind を先走って固定し、実装裏付けがないまま運用前提が増える
  - mitigation: この plan は contract と admission policy に絞り、runtime adapter 実装は別 execution に分離する

## Design Decisions

- dev-only runner overlay と official registration policy は分離して扱う
- official registered game の runtime kind は security と運用負荷の比較を踏まえて ADR で決定する
- simple executable / local subprocess を official registration でどう扱うかは、この plan の execution で明示判断する
