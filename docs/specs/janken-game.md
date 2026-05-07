# じゃんけんゲーム仕様

## 目的

`janken` は `echo-count` fixture 完了後の richer integration game である。
`echo-count` が閉じるのは deterministic payload による platform core の verification までであり、
`janken` はその上で以下を実検証する。

- hidden action reveal
- simultaneous round resolution
- game-specific action schema
- `self_history` / `public_history` の更新
- multi-round ranking / tie-break

したがって、以下は `echo-count` で担保済みの platform 共通責務であり、`janken` の主責務ではない。

- `arena-runner` の match 起動自体
- game / game_version / ruleset selection
- metadata compatibility 判定
- timeout / malformed / mismatched id / late response / shutdown failure の platform 記録分類
- replay/debug entrypoint や persisted artifact 読み出し

`janken` 側では、fixture では薄いゲーム固有の visible state と勝敗解決を担保する。

Phase 4 では `janken` ruleset を共有する `janken-wasm` を、Go 製 WASM/WASI AI runtime の最初の公式検証ゲームとして使う。
この役割では、game ルール自体の検証に加えて、Go で書いた AI を WASM/WASI へ build し、
`arena-runner` が sidecar manifest 経由でその module を正式経路として完走できることも確認対象に含める。

## Metadata

- subprocess verification game_id: `janken`
- WASM verification game_id: `janken-wasm`
- `game_version`: `2.1.0`
- `ruleset_version`: `regular`

`regular` ruleset は 5 ラウンド固定とする。

同時行動であることは metadata 互換性 key ではなく、`DecisionStep.mode = simultaneous` を返す
game master 契約として表現する。

## 試合形式

- プレイヤー数: 2人以上
- ターンモデル: 同時行動
- ラウンド数: 5
- 選択可能な手: `rock`, `paper`, `scissors`
- 試合終了: 5 ラウンド終了時

試合途中でプレイヤーが脱落することはない。

## ラウンド解決

各プレイヤーは、各ラウンドで 1 つの行動を提出する。

### 勝敗ルール

- `rock` は `scissors` に勝つ
- `scissors` は `paper` に勝つ
- `paper` は `rock` に勝つ

### 多人数戦での解釈

有効な手だけを見て 1 ラウンドを解決する。

- 全員が同じ手なら全員引き分け
- 3 種類すべての手が出たら全員引き分け
- ちょうど 2 種類の手だけが出たら、勝つ側の手を出したプレイヤー全員を勝ち、負ける側の手を出したプレイヤー全員を負けとする

## タイムアウトと無効アクション

- タイムアウト時の行動は `no_action` とみなす
- 無効アクションも `no_action` とみなす
- `no_action` を出したプレイヤーは、そのラウンド全体の勝敗がどう解決されたかに関係なく個別に負け扱いとする
- 有効な手を出したプレイヤー同士の勝敗は通常のじゃんけんルールで解決する
- 有効な手を出したプレイヤー同士が引き分けでも、`no_action` を出したプレイヤーだけは負けになる
- 全員が `no_action` の場合は全員負け扱いとする

この方針により、意図的に手を出さないことで引き分けに逃げるメリットを与えない。

## スコアと順位

各プレイヤーは以下を記録する。

- `wins`
- `losses`
- `draws`
- `timeouts`
- `invalid_actions`

主順位指標:

- 勝率 = `wins / rounds`

同率時のタイブレーク順:

1. `losses` が少ない方
2. `timeouts` が少ない方
3. `invalid_actions` が少ない方
4. それでも同じなら同順位

最終 match record の `result.placements` は上記規則で決める。

## 可視情報モデル

### `init.state`

`init` の `params.state` は以下の shape を持つ。

```json
{
  "players": ["p1", "p2"],
  "rounds": 5
}
```

### `turn.visible_state`

各 `turn` request の `visible_state` は以下を持つ。

- `round`: 現在ラウンド番号。1 始まり
- `rounds`: 総ラウンド数
- `self_history`: 自分視点の解決済みラウンド履歴
- `public_history`: 全員に公開される解決済みラウンド履歴

`self_history` の各要素:

```json
{
  "round": 1,
  "action": "rock",
  "result": "win"
}
```

`public_history` の各要素:

```json
{
  "round": 1,
  "actions": {
    "p1": "rock",
    "p2": "scissors"
  },
  "results": {
    "p1": "win",
    "p2": "loss"
  }
}
```

ここでの `action` 値は `rock` / `paper` / `scissors` / `no_action` のいずれかである。

`public_history` は、すでに解決済みの過去ラウンドのみを含む。現在ラウンドの提出内容は解決完了まで含めず、同時行動の隠し手を維持する。

`self_history` は `public_history` と一部情報が重複するが、AI が自分の選択と結果の対応を自己検証しやすい形を先に踏ませるため、意図的に独立フィールドとして持たせる。

### `turn.legal_action_hint`

`legal_action_hint` は `visible_state` の外側に置き、常に以下とする。

```json
{
  "type": "object",
  "required": ["action"],
  "properties": {
    "action": {
      "type": "string",
      "enum": ["rock", "paper", "scissors"]
    }
  }
}
```

## アクションスキーマ

`turn` に対する AI レスポンスの `result` は以下。

```json
{
  "action": "rock"
}
```

許可される `action` 値:

- `rock`
- `paper`
- `scissors`

それ以外は無効アクションとし、platform 上は `invalid-illegal-action` として記録しつつ、ゲーム内では `no_action` として解決する。

## JSON-RPC 例

### `init`

request:

```json
{
  "jsonrpc": "2.0",
  "id": "init",
  "method": "init",
  "params": {
    "match_id": "match-001",
    "player_id": "p1",
    "game_id": "janken",
    "game_version": "2.1.0",
    "ruleset_version": "regular",
    "deadline_ms": 1000,
    "state": {
      "players": ["p1", "p2"],
      "rounds": 5
    }
  }
}
```

response:

```json
{
  "jsonrpc": "2.0",
  "id": "init",
  "result": {
    "ready": true
  }
}
```

## Phase 4 Go-WASM 検証構成

Go 製 WASM/WASI sample AI verification では、少なくとも以下を満たす構成を基準とする。

- game: `janken-wasm`
- game_version: `2.1.0`
- ruleset_version: `regular`
- players:
  - `p1=./testdata/ai/janken/janken-go-wasm-ai`
  - `p2=./testdata/ai/janken/janken-rock-ai-wasm` または同等の比較用 fixture

期待結果:

- match は `completed` で終了する
- standard artifact として `record.json` / `history.json` が `<output-dir>/<match-id>/` に残る
- Go-WASM player の `stderr_bytes` が 0 より大きく、runtime 経由で `stderr` capture が効いている
- mixed runtime 構成でも `public_history` / placement / failure 分類 contract が崩れない

負例として最低限確認するもの:

- sidecar manifest が指す module が存在しない場合、runner は match 開始前に起動失敗を返す
- manifest metadata が match metadata と不整合な場合、runner は compatibility error を返す

ここでの `ready: true` は、AI が `janken` の `init` request を解釈し、初期状態を受け取ったうえでプロトコル応答できたことを示す ACK として扱う。

### `turn`

request:

```json
{
  "jsonrpc": "2.0",
  "id": "turn-3-p1",
  "method": "turn",
  "params": {
    "turn": 3,
        "visible_state": {
      "round": 3,
      "rounds": 5,
      "self_history": [
        {"round": 1, "action": "rock", "result": "win"},
        {"round": 2, "action": "paper", "result": "draw"}
      ],
      "public_history": [
        {
          "round": 1,
          "actions": {"p1": "rock", "p2": "scissors"},
          "results": {"p1": "win", "p2": "loss"}
        },
        {
          "round": 2,
          "actions": {"p1": "paper", "p2": "paper"},
          "results": {"p1": "draw", "p2": "draw"}
        }
      ]
    },
    "legal_action_hint": {
      "type": "object",
      "required": ["action"],
      "properties": {
        "action": {
          "type": "string",
          "enum": ["rock", "paper", "scissors"]
        }
      }
    },
    "deadline_ms": 100
  }
}
```

response:

```json
{
  "jsonrpc": "2.0",
  "id": "turn-3-p1",
  "result": {
    "action": "scissors"
  }
}
```

### `game_over`

request:

```json
{
  "jsonrpc": "2.0",
  "id": "game-over",
  "method": "game_over",
  "params": {
    "match_id": "match-001",
    "final_visible_state": {
      "round": 5,
      "rounds": 5,
      "self_history": [
        {"round": 1, "action": "rock", "result": "win"},
        {"round": 2, "action": "paper", "result": "draw"},
        {"round": 3, "action": "scissors", "result": "loss"},
        {"round": 4, "action": "rock", "result": "win"},
        {"round": 5, "action": "paper", "result": "win"}
      ],
      "public_history": [
        {
          "round": 1,
          "actions": {"p1": "rock", "p2": "scissors"},
          "results": {"p1": "win", "p2": "loss"}
        },
        {
          "round": 2,
          "actions": {"p1": "paper", "p2": "paper"},
          "results": {"p1": "draw", "p2": "draw"}
        },
        {
          "round": 3,
          "actions": {"p1": "scissors", "p2": "rock"},
          "results": {"p1": "loss", "p2": "win"}
        },
        {
          "round": 4,
          "actions": {"p1": "rock", "p2": "scissors"},
          "results": {"p1": "win", "p2": "loss"}
        },
        {
          "round": 5,
          "actions": {"p1": "paper", "p2": "rock"},
          "results": {"p1": "win", "p2": "loss"}
        }
      ]
    },
    "summary": {
      "placements": [
        {"player_id": "p1", "place": 1},
        {"player_id": "p2", "place": 2}
      ]
    },
    "shutdown_after_ms": 3000
  }
}
```

response:

```json
{
  "jsonrpc": "2.0",
  "id": "game-over",
  "result": {
    "ack": true
  }
}
```

`game_over` ACK は、AI が終了前 cleanup を完了したあとに返す。AI はこの request を受けたあとに最終ラウンド結果も含めて自己評価し、必要なら改善用レポートを `stderr` へ出力してよいが、その完了は `shutdown_after_ms` の猶予内でなければならない。

## 観戦向け全体状態

じゃんけんの exported public state には少なくとも以下を含める。

- 設定ラウンド数
- 現在の解決済みラウンド数
- 解決済みラウンドの全プレイヤー提出内容
- 各ラウンド結果
- 累積スコア表

これにより、観戦 UI や replay/debug でラウンドごとの reveal を再構築できる。

## このゲームを使う理由

じゃんけん自体を最終的な主力ゲームにしたいわけではない。このゲームを選ぶ理由は、共有 AI プロトコルの上で、同時隠し行動、複数ラウンド、順位付け、締切処理を最小のルールで検証できるからである。
