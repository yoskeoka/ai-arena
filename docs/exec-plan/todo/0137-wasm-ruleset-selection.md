# WASM game master への大会 ruleset 伝達

> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## 目的と完了境界

運営者が同一 game release の別 ruleset を持つ大会を別々に登録し、各大会の試合を実行できる。各大会で固定した ruleset はその試合の WASM game master 起動時に伝わり、game master の metadata、状態、結果および再開時の識別と一致する。試合中に ruleset を変更する機能は不要。単一 bundle から複数 ruleset を利用でき、既存の単一 ruleset bundle と過去の run は互換を保つ。Reversi XOT は利用例であり game 固有実装は対象外。Addresses: none.

## 背景と方針

`docs/specs/platform-game-registry.md` は `game_id + major` で release を解決し、ruleset を build 入力として渡す。exact version と immutable digest の一意性は再現と監査を支えるため維持する。`internal/platform/registry/wasm_resolver.go` は選択 ruleset を期待 metadata に入れる一方、WASM へは固定 runtime args しか伝えない。`internal/platform/gamemaster/gamemaster.go` は initialize 前に metadata を照合するため、initialize request にだけ値を追加しても不足する。起動時伝達を欠くこと自体の意図的な制限理由は確認できていない。

WASI game master に platform が予約する環境変数 `AI_ARENA_RULESET_VERSION` を起動時に渡す。値は検証済みの大会 / run の ruleset をそのまま用い、metadata 問い合わせ前に利用可能とする。未対応の既存 game は無視でき、既存 runtime args の解釈は変えない。環境変数は game master 専用で、player AI や host 環境を継承しない。対応 game は未知の値を拒否し、選択した ruleset を metadata に返す。platform は従来どおり不一致を拒否する。初期化時のみの伝達や全 bundle に新しい必須引数を追加する方法は、metadata の順序または既存 bundle 互換を満たさない。

## 変更対象と契約

- (MODIFY) `docs/specs/game-master.md`、`docs/specs/platform-game-registry.md`、`docs/specs/platform-artifact-bundle.md`: 上記起動契約、選択元、未対応・不一致、過去の run と互換性を先に明記する。field の正本が必要な場合は `gamemaster/types.go` 等の typed contract を更新し、Markdown に重複させない。
- (MODIFY) `internal/platform/registry/wasm_resolver.go` の `BuildSession` / `BuildSessionFromSnapshot`、`internal/platform/gamemaster/gamemaster.go` の `WASIConfig` / `StartWASMWASI`、`internal/platform/runtime/wasm_wasi.go`: 検証された build input を起動時環境へ伝える。空・未対応 ruleset は起動前に拒否し、bundle の固定 args と sandbox 制約を保つ。
- (MODIFY) `internal/platform/service/general.go`、`internal/platform/service/worker_local.go`、queue / retry 経路、および `cmd/arena-runner` の bundle 実行経路: 大会登録から実行、resume、retry まで選択と artifact digest が固定されることを確認し、必要箇所を修正する。複数 ruleset bundle の直接 runner 実行では明示選択でき、単一 ruleset の既存呼び出しは従来どおり動くようにする。未知・曖昧な選択は失敗させる。
- (MODIFY/NEW) 上記の registry、runtime、service、runner のテストと WASI fixture: １つの bundle が２ ruleset で異なる metadata / observable behavior を返す fixture を用いる。固定 args の既存 fixture も維持する。

## 検証と依存

1. 同一 bundle を使う２大会でそれぞれ選択された値が起動前から利用でき、metadata と実際の挙動に一致することを operator 登録、queue、WASI worker、成果物まで検証する。異なる規則の大会を同時に扱い、ruleset や環境が漏れないことを確認する。
2. fresh、snapshot resume、retry / rerun で元の artifact digest と ruleset が維持されること、未対応・不一致・欠落・曖昧な選択が明示的に失敗することを検証する。既存単一 ruleset bundle、固定 args、直接 runner、既存 record の回帰を確認する。
3. 適用される Go / WASI / service のテストと workflow lint を実行し、実行可能な bundle を使った E2E 証拠を残す。XOT 側はこの実装のマージ後、対応 revision を pin して別途標準・large・small の実行を検証する。

この計画は `https://github.com/yoskeoka/reversi-ai-arena/pull/56` の XOT 計画に先行する。実装はこの計画のレビュー・マージ後に `/execute-task` で行う。
