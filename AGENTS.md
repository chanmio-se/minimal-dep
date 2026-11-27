# AGENTS.md

本リポジトリは [docs/architecture.md](docs/architecture.md) に定義された Go + HTMX フレームワークの実装である。設計意図・境界・検証方法をここに集約し、変更はこの方針に従う。

## Purpose

- サーバーサイドレンダリングを中心とした MPA を、HTMX による部分更新で構築する。
- 単一バイナリ・単一プロセスの Modular Monolith を維持する。
- Go 標準ライブラリを優先し、依存と保守コストを小さく保つ。

## Boundaries

- `internal/app` は Composition Root であり、モジュールの生成・依存注入・ルート登録を明示する。
- `internal/platform` は業務ルールを持たない技術的共通機能のみを提供し、`internal/modules` を import しない。
- モジュールは `api` / `domain` / `usecase` / `repository` / `handler` / `templates` を所有する。
- モジュール間の依存は相手モジュールの `api` パッケージ経由のみとし、一方向に保つ。
- レイヤーの依存方向は `handler -> usecase -> domain` および `usecase -> repository -> domain` とする。`domain` は他レイヤーに依存しない。
- テンプレートは所有するモジュール内に置き、レイアウトのみ `internal/platform/templates` で共有する。

## Constraints

- HTTP は `net/http.ServeMux`、HTML は `html/template`、DB は `database/sql`、ログは `log/slog` を使う。
- ルーティング・依存関係・テンプレート呼び出し・エラー処理は暗黙にせず明示する。
- HTMX 判定とレスポンス選択は `internal/platform/http/htmx` に集約し、Handler から利用する。
- 通常リクエストには完全な HTML、HTMX リクエストにはフラグメントを返す。フォームは通常送信にフォールバックできる形を保つ。
- POST / PUT / DELETE 相当のリクエストには常に CSRF トークンを要求する。
- 外部 Go ライブラリは追加しない。追加する場合は ADR に採用理由を記録し、Linter の承認リストを更新する。
- HTMX は `package.json` / `package-lock.json` でバージョン固定し、`npm run sync:htmx` で `web/static/js` に同梱する。更新時は E2E テスト結果を確認する。Dependabot の npm 更新 PR では `.github/workflows/sync-dependabot-htmx.yml` が同期結果を PR ブランチへコミットする。
- GitHub Actions の依存は `.github/workflows/actions.lock` で固定する。`uses:` を変更した場合は `gh actions-lock` を再実行し、lockfile を同じ変更に含める。lockfile は手編集しない。

## Verification

変更の完了前に、以下をすべて実行する。

```sh
go build ./...
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...
go vet ./...
go run ./cmd/archlint .
go test ./...
```

`archlint` はモジュール境界・レイヤー方向・依存方針・循環依存の Fitness Function を実装する。ルールの定義元は `internal/linter/architecture` であり、指示文ではなく Linter を更新する。実行時の振る舞い（通常 HTML / HTMX フラグメント / フォーム送信 / CSRF）は `tests/` の統合テストで検証する。

workflow を変更した場合は `gh actions-lock --verify` も実行する。CI では `.github/workflows/verify-lockfile.yml` が同じ検査を pull request ごとに実行する。

HTMX を更新した場合は `npm ci --ignore-scripts && npm run sync:htmx` を実行し、同梱 JS と ADR の差分を確認する。CI の `htmx` job が同じ検査と `go test ./tests/...` を実行する。Dependabot の htmx 更新 PR では `sync-dependabot-htmx.yml` が CI 完了後に同期・コミットし、`workflow_dispatch` で CI を再実行する。

## Completion Criteria

- 上記の検証コマンドがすべて成功すること。
- 新しいモジュールは `go run ./cmd/modulegen --name=<name>` で生成し、Composition Root に明示的に登録すること。
- 設計判断の変更は `docs/decisions/` に ADR として記録すること。
- 未実行の確認項目や既知の失敗がある場合は、完了報告で明示すること。
