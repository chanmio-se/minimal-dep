# ADR-0002: Modular Monolith と明示的な Composition Root

- 状態: Accepted
- 日付: 2026-11-27

## Context

機能追加の局所性と将来の分割可能性を確保しつつ、単一プロセスの運用を維持する必要がある。暗黙的な DI やリフレクションは AI による変更の安全性を下げる。

## Decision

- 機能ごとにモジュールを分け、内部を `api` / `domain` / `usecase` / `repository` / `handler` / `templates` に層分割する。
- モジュール間の依存は `api` 経由の一方向に限定する。
- `internal/app` を Composition Root とし、依存関係とルート登録をコードで明示する。
- 依存方向と境界は `internal/linter/architecture` の Fitness Function を golangci-lint カスタム linter として機械的に検査する。

## Consequences

- 新しい機能はモジュール境界を守る限り、他モジュールに影響せず追加できる。
- 境界を越える必要が生じた場合は、公開 `api` を扩充するか Composition Root で調整する。
- `internal/linter/architecture` が機械検査できるルールの唯一の定義元となり、設計方針の変更時はルールも更新する。`tools/archlintplugin` は golangci-lint へのアダプタに留める。
