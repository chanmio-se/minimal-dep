# ADR-0003: セキュリティを標準経路と Fitness Function で担保する

- 状態: Accepted
- 日付: 2026-11-27

## Context

`docs/architecture.md` 12.1〜12.3 で、セキュリティ対策をアプリケーションごとの任意実装にせず、
フレームワークの標準経路として提供することが求められた。外部 Go 依存を増やさず、
静的検査と実行時テストの役割を分けて継続的に検証する必要がある。

## Decision

- `internal/app` の標準 HTTP チェーンに `SecurityHeaders` / `MaxBody` / `CSRF` / `Timeout` / `Auth` を明示的に含める。
- CSRF は double-submit token とし、Cookie は `HttpOnly` を維持する。HTMX にはレイアウトの meta タグと
  `web/static/js/csrf.js` 経由で `X-CSRF-Token` を付与し、通常フォームは hidden field で送信する。
- セッションの生成・更新・失効は `internal/platform/session.Store` に集約し、ログイン時は
  `LoginWithSession` で旧トークンを Rotate してセッション固定攻撃を防ぐ。Cookie 属性は
  `internal/platform/http/cookie` に集約し、`HttpOnly` / `SameSite=Lax` / HTTPS 時の `Secure` を既定とする。
- 認証情報は Middleware が解決して Usecase に明示的に渡し、認可は各モジュールの Usecase / Domain が所有する。
  Handler は形式・サイズを検証し、Domain は業務上の不変条件を検証する。
- SQL はプレースホルダー、HTML は `html/template` を標準経路とし、内部エラーの詳細はログにのみ記録する。
- `cmd/archlint` に `sql-parameterized` / `html-template-only` / `security-middleware` を追加する。
  振る舞いは `tests/` の統合テストと各パッケージの単体テストで検証し、Generator も同じパターンを生成する。

## Consequences

- 新しいモジュールは追加設定なしでセキュリティ Middleware とテストパターンを継承できる。
- Cookie を JavaScript から読まないため、HTMX 用のトークン注入を meta タグ + 共通スクリプトに限定する。
- 静的ルールでは認可や CSRF の実行時挙動を証明できないため、統合テストを CI の必須条件として維持する。
- 設計変更時は Linter ルールとテストの両方を同じ変更単位で更新する。
