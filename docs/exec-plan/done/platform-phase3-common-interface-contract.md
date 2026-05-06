# platform-phase3-common-interface-contract

## Objective

AI player、platform、game master の間で将来のゲーム追加やダンジョンゲーム実装に共通で使う interface 契約を固定する。あわせて、game master が platform 内で動く場合と trusted external game backend 上で動く場合の両方に対応しやすい、汎用的な境界を明文化する。

## Goal

- AI player が従う platform 共通契約を、個別ゲーム仕様から切り出して固定する
- game master と platform の責務境界を、platform 内実行と trusted external game backend 実行の両方を見据えて固定する
- action の受理結果や failure reason の分類を platform 共通の契約として固定する
- 複数ゲームを追加していく前提で、登録済み game を識別・選択できる共通的な game registry の方向を固める
- ダンジョンゲーム実装がこの phase 完了後に大きな interface 手戻りなしで進められる状態にする

## Out of Scope

- 実装方式の確定
- 具体的な API / package 構成の確定
- 個別 game の詳細仕様策定
- 実装着手

## Follow-up

- この plan は phase goal を固定するための skeleton であり、実装や spec 詳細化の HOW は後続の詳細 plan に分割して扱う
- 特に platform 共通 spec の切り出し、external game backend 境界、game registry、failure reason の整理は別 plan に分けて具体化する
