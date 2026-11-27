# minimal-dep

Go と HTMX による軽量 Web フレームワークの雛形。[docs/architecture.md](docs/architecture.md) の設計に従った Modular Monolith 構成です。

## 特徴

- `net/http.ServeMux` による明示的なルーティング
- `html/template` によるサーバーサイドレンダリング（レイアウト / ページ / フラグメント）
- `HX-Request` に基づく HTML フラグメントの返却
- `database/sql` と `log/slog`（外部 Go 依存なし）
- モジュールごとの境界と、Composition Root による明示的な依存注入
- アーキテクチャとセキュリティの Fitness Function を検査する `archlint`
- CSRF / Session / 入力検証 / 認可を標準経路とし、セキュリティテストを同梱
- Quint による状態遷移セキュリティ特性の仕様化とトレース駆動テスト
- GitHub Actions の依存を lockfile で固定し、CI で検証
- HTMX を npm の開発依存から同梱ファイルへ同期し、CI で整合性を検証

## ディレクトリ

```text
cmd/server        アプリケーション起動
cmd/migrate       DB マイグレーション適用
cmd/modulegen     モジュール雛形生成
cmd/archlint      アーキテクチャ検査
internal/app      Composition Root
internal/platform 共通基盤（config / database / http / logging / templates）
internal/modules 機能モジュール（identity / tasks）
web/static        静的ファイル（HTMX 同梱）
migrations        DB マイグレーション
tests             モジュール横断の統合テスト / Quint トレースドライバー
spec/quint        Quint の状態遷移モデル
docs/decisions    設計判断の ADR
scripts           HTMX 同梱ファイルの同期スクリプト
package.json      HTMX 更新用の npm 開発依存
.github/workflows CI / lockfile 検証 / HTMX 同期の workflow
```

## 起動

```sh
go run ./cmd/server
curl -i http://localhost:8080/healthz
```

初回は `http://localhost:8080/login` でメールアドレスと名前を入力すると、ユーザーが作成され `/tasks` にリダイレクトされます。DB を設定しない場合はメモリリポジトリで動作します。

### 環境変数

| 変数 | 既定値 | 説明 |
|---|---|---|
| `ADDR` | `:8080` | リッスンアドレス |
| `APP_ENV` | `development` | `production` で JSON ログ |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `DB_DRIVER` | 空 | 空ならメモリリポジトリ |
| `DB_DSN` | 空 | `DB_DRIVER` と同時に指定 |
| `SESSION_SECRET` | 起動時に生成 | 32 文字以上を推奨 |
| `SESSION_TTL` | `8h` | セッションの有効期間 |
| `MAX_BODY_BYTES` | `1048576` | リクエストボディの上限（バイト） |
| `SECURE_COOKIES` | `false` | HTTPS 環境で `true` |
| `REQUEST_TIMEOUT` | `10s` | リクエスト全体の上限 |
| `SHUTDOWN_TIMEOUT` | `10s` | Graceful Shutdown の上限 |

DB を使う場合はドライバをアプリケーションに登録し、`go run ./cmd/migrate` でマイグレーションを適用します。リポジトリの SQL は `?` プレースホルダーを使うため、SQLite / MySQL 互換のドライバを想定しています（PostgreSQL を使う場合は `$1` 形式への変更が必要です）。

## 検証

```sh
make build
make test
make lint
```

`lint` は golangci-lint（実行ごとに最新バージョンを取得）、`go vet`、`go run ./cmd/archlint .` を実行します。golangci-lint は開発ツールとして実行し、Go モジュールの依存には追加しません。`archlint` は次を検査します。

- モジュール間依存が `api` を通ること
- モジュール間の循環依存がないこと
- レイヤーの依存方向（`domain` は他レイヤーに依存しない等）
- 未承認の外部 Go 依存がないこと
- `platform` がモジュールに依存しないこと
- SQL がプレースホルダーを使い、HTML が `html/template` を経由すること
- `SecurityHeaders` / `MaxBody` / `CSRF` / `Auth` が標準 HTTP 構成に含まれること

Quint のモデル検査とトレース再生は `.github/workflows/ci.yml` で実行します。ローカルでのコマンドは [docs/quint.md](docs/quint.md) を参照してください。

GitHub Actions の依存は `.github/workflows/actions.lock` に固定しています。workflow の `uses:` を変更した場合は `gh actions-lock` を再実行し、生成された lockfile をコミットしてください。lockfile は手編集しません。`.github/workflows/verify-lockfile.yml` が push と pull request で整合性を検証します（ローカルでは `gh actions-lock --verify`）。`.github/dependabot.yml` がアクションの更新を確認し、lockfile も同じ PR で更新します。採用理由は [ADR-0005](docs/decisions/0005-actions-lockfile.md) を参照してください。

HTMX は `package.json` / `package-lock.json` でバージョン固定しています。Dependabot が `htmx.org` の更新 PR を作ると、`.github/workflows/sync-dependabot-htmx.yml` が CI 完了後に `npm ci --ignore-scripts` と `npm run sync:htmx` を実行し、`web/static/js/htmx.min.js` と [ADR-0001](docs/decisions/0001-standard-library-and-htmx.md) のハッシュを PR へコミットします。その後 `workflow_dispatch` で CI を再実行します。手動で更新する場合も同じコマンドを実行して差分をコミットしてください。CI の `htmx` job が `npm ci` → `npm run sync:htmx` → `git diff --exit-code` で差分がないことを確認し、`go test ./tests/...` も実行します。

## モジュール追加

```sh
go run ./cmd/modulegen --name=widgets
```

生成後に表示される案内に従い、`internal/app/app.go` でモジュールを生成・登録します。生成物には CSRF 対応フォーム、入力検証、認可を含む Usecase、および未認証・CSRF 不正・他オーナー認可のテストが含まれます。生成物は通常のソースコードとして編集できます。

## 依存関係

Go の依存ライブラリはありません。HTMX は `htmx.org` として npm の devDependency に固定し、`scripts/sync-htmx.mjs` で `web/static/js/htmx.min.js` に複製しています。npm は更新時の開発ツールだけで、アプリの実行時や通常の Go ビルドには不要です（採用理由は [ADR-0001](docs/decisions/0001-standard-library-and-htmx.md)）。
