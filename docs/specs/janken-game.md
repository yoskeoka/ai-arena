# じゃんけんゲーム仕様

## 目的

じゃんけんは、プラットフォームの Phase 2 実証用ゲームである。より複雑なダンジョンゲームへ進む前に、プラットフォームの基本ループを最小構成で検証することを目的とする。

このゲームで検証したい要素は以下。

- 同時行動ターン処理
- 解決まで相手の手が見えない隠し情報
- タイムアウト処理
- 無効アクション処理
- 勝率ベースの複数ラウンド順位付け

## 試合形式

- プレイヤー数: 2人以上
- ターンモデル: 同時行動
- ラウンド数: 事前設定された固定 `N`
- 選択可能な手: `rock`, `paper`, `scissors`
- 試合終了: `N` ラウンド終了時

## ラウンド解決

各アクティブプレイヤーは、各ラウンドで1つの手を提出する。

### 勝敗ルール

- `rock` は `scissors` に勝つ
- `scissors` は `paper` に勝つ
- `paper` は `rock` に勝つ

### 多人数戦での解釈

2人を超える場合の1ラウンド解決は以下とする。

- 全員が同じ手なら全員引き分け
- 3種類すべての手が出たら全員引き分け
- ちょうど2種類の手だけが出たら、勝つ側の手を出したプレイヤー全員を勝ち、負ける側の手を出したプレイヤー全員を負けとする

試合途中でプレイヤーが脱落することはない。

## タイムアウトと無効アクション

- タイムアウト時の行動は `no_action` とみなす
- 無効アクションも `no_action` とみなす
- `no_action` は、そのラウンドに有効な手が1つでも存在すれば負け扱いになる
- 全員が `no_action` の場合は全員引き分けとする

この方針により、別の罰則システムを増やさずに、タイムアウトや不正行動を最終順位へ反映できる。

## スコアと順位

各プレイヤーは以下を記録する。

- `wins`
- `losses`
- `draws`
- `timeouts`
- `invalid_actions`

主順位指標:

- 勝率 = `wins / N`

同率時のタイブレーク順:

1. `losses` が少ない方
2. `timeouts` が少ない方
3. `invalid_actions` が少ない方
4. それでも同じなら同順位

## 可視情報モデル

### 初期情報

`init` 時に各プレイヤーへ渡す情報:

- `game`: `janken`
- `player_id`
- `players`: プレイヤー ID の順序付き配列
- `rounds`: 総ラウンド数
- `deadline_ms`

### 各ラウンド情報

各 `turn` の `visible_state` には以下を含める。

- `round`
- `rounds`
- `self_history`
- `public_history`

`self_history` は、そのプレイヤー自身の過去の手と結果を含む。

`public_history` は、すでに解決済みの過去ラウンドのみを含む。現在ラウンドの未解決提出内容は含めず、同時行動の隠し手を維持する。

`legal_action_hint` は `visible_state` の外側に置き、常に以下とする。

```json
["rock", "paper", "scissors"]
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

それ以外は無効アクションとする。

## JSON-RPC 例

### `init`

リクエスト:

```json
{
  "jsonrpc": "2.0",
  "id": "init",
  "method": "init",
  "params": {
    "match_id": "match-001",
    "player_id": "p1",
    "game": "janken",
    "ruleset_version": "phase1",
    "deadline_ms": 100,
    "state": {
      "players": ["p1", "p2"],
      "rounds": 5
    }
  }
}
```

レスポンス:

```json
{
  "jsonrpc": "2.0",
  "id": "init",
  "result": {
    "ready": true
  }
}
```

### `turn`

リクエスト:

```json
{
  "jsonrpc": "2.0",
  "id": "round-3",
  "method": "turn",
  "params": {
    "turn": 3,
    "visible_state": {
      "round": 3,
      "rounds": 5,
      "self_history": [
        {"round": 1, "action": "rock", "outcome": "win"},
        {"round": 2, "action": "paper", "outcome": "draw"}
      ],
      "public_history": [
        {
          "round": 1,
          "actions": {"p1": "rock", "p2": "scissors"},
          "outcomes": {"p1": "win", "p2": "loss"}
        },
        {
          "round": 2,
          "actions": {"p1": "paper", "p2": "paper"},
          "outcomes": {"p1": "draw", "p2": "draw"}
        }
      ]
    },
    "legal_action_hint": ["rock", "paper", "scissors"],
    "deadline_ms": 100
  }
}
```

レスポンス:

```json
{
  "jsonrpc": "2.0",
  "id": "round-3",
  "result": {
    "action": "scissors"
  }
}
```

### `result`

リクエスト:

```json
{
  "jsonrpc": "2.0",
  "id": "result",
  "method": "result",
  "params": {
    "placement": 1,
    "score": {
      "wins": 3,
      "losses": 1,
      "draws": 1,
      "win_rate": 0.6
    },
    "summary": {
      "players": 2,
      "rounds": 5,
      "tie_breakers_applied": []
    }
  }
}
```

## 観戦向け全体状態

じゃんけんの全体状態エクスポートには以下を含める。

- 設定ラウンド数
- 現在ラウンド番号
- 解決済みラウンドの全プレイヤー提出内容
- 各ラウンド結果
- 累積スコア表
- 現在ラウンドで未提出のプレイヤー一覧

これで将来の観戦 UI でラウンドごとの reveal 表示が可能になる。

## このゲームを使う理由

じゃんけん自体を最終的な主力ゲームにしたいわけではない。このゲームを選ぶ理由は、共有 AI プロトコルの上で、同時隠し行動、複数ラウンド、順位付け、締切処理を最小のルールで検証できるからである。
