# ADR-0004: Quint で状態遷移セキュリティ特性を仕様化する

- 状態: Accepted
- 日付: 2026-11-27

## Context

Linter はコード構造、通常のテストは実装の振る舞いを検査できるが、セッション更新や認可の
状態遷移・不変条件を仕様として独立に検証する手段が必要だった。状態機械の仕様と実装の対応を
継続的に確認したい。

## Decision

- `spec/quint/` に session / authz / csrf のモデルを置き、順序・重複・失敗時の状態を明示する。
- Quint の simulator が出力した ITF トレースを `tests/quint/traces/` に保持し、
  `tests/quint/` の Go テストドライバーで各ステップの結果と状態を照合する。
- Quint の CLI バージョンを固定し、`parse` / `typecheck` / `run` / `verify` と Go ドライバーを
  `.github/workflows/ci.yml` で実行する。CI のランタイムは Node 26.13.1 と Eclipse Temurin 25.0.5 に固定し、
`verify`（Apalache）は Java 25 で実行する（Apalache の推奨ランタイム）。
- モデルは重要な状態・操作・前提・不変条件に絞る。TTL 満了や乱数トークンなどモデル外の詳細は
  実装側の単体テストで検証し、Quint の結果はモデルと前提に対する結果として扱う。

## Consequences

- 仕様と実装のずれは、トレースと Go ドライバーの失敗として具体的に検出できる。
- 反例が得られた場合は再現可能なテストケースとして扱い、モデル・ドライバー・実装を同じ変更単位でレビューする。
- Apalache 実行には Java が必要で、ローカル環境に Java がない場合は `verify` を実行できない。
  その場合も `parse` / `typecheck` / `run` と Go ドライバーで継続的に確認する。
