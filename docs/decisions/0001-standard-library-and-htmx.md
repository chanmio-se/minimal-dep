# ADR-0001: 標準ライブラリと HTMX の採用

- 状態: Accepted
- 日付: 2026-11-27

## Context

軽量で長期保守しやすいフレームワークとし、AI によるコード生成・変更にも適した明確な構成が必要である。フロントエンドのビルドを増やさず、サーバー主導の画面更新を実現する必要がある。

## Decision

- HTTP / ルーティング / HTML / DB / ログ / テストには Go 標準ライブラリを使う。本番コードで認証・認可・暗号化など自前実装のリスクが高い分野に限り、ADR に採用理由とリスク評価を記録し、`internal/linter/architecture` の承認リストに追加した外部ライブラリを使う。
- 画面更新には HTMX を利用し、`web/static/js/htmx.min.js` に同梱する。バージョンは npm のロックファイルで固定する。
- HTMX は外部 JavaScript ライブラリとして扱い、Go モジュールの依存には追加しない。
- `package.json` と `package-lock.json` に `htmx.org` を devDependency として固定する。npm は同梱ファイルの更新時だけ使う開発ツールとし、アプリの実行時や通常の Go ビルドには使わない。
- sqlc / golangci-lint / Quint / npm などの開発ツールは実行バイナリに含まれないため承認リストの対象外とし、バージョン固定と CI 検証で管理する。HTMX は同梱 JS を生成するため、この ADR で個別に扱う。
- `.github/dependabot.yml` の npm エコシステムで更新を確認する。`htmx.org` を含む更新 PR では、`.github/workflows/sync-dependabot-htmx.yml` が CI 完了後に `npm ci --ignore-scripts` と `npm run sync:htmx` を実行し、同梱 JS と ADR のハッシュを PR ブランチへコミットする。
- 自動更新は `workflow_run` でデフォルトブランチのスクリプトを使い、PR のコードは checkout しない。`pull_request_target` で PR のコードを実行して書き込む方式は採らない。
- 同期 workflow は `GITHUB_TOKEN` で push するため、続けて `workflow_dispatch` で CI を実行し、差分がないことと統合テストを確認する。
- CI では `npm ci` → `npm run sync:htmx` → `git diff --exit-code` で同梱 JS と ADR の差分がないことを確認し、`go test ./...`（`tests/` の統合テストを含む）を必須条件とする。

SHA-256 (htmx.min.js):
`d6fdc75f204e6bdefa99b69bf1e6d4ac69b8a364f77929f45c13476b4000f717`

## Consequences

- 生成済み JS をコミットするため、単一バイナリで配布でき、フロントエンドのビルドは不要である。
- 本番コードに外部 Go 依存を追加する場合は、認証・認可・暗号化の例外分野であっても ADR に採用理由とリスク評価を記録し、`internal/linter/architecture` の承認リストに追加する必要がある。
- 開発ツールは承認リストを変更せず、バージョン固定と CI 検証で管理する。
- Dependabot の npm 更新 PR だけでは同梱 JS と ADR は更新されない。同期 workflow が生成差分を PR へコミットする。手動で更新する場合も同じコマンドを実行して差分をコミットする。
- `node_modules/` はコミットしない。HTMX を更新する場合は、同梱 JS と ADR のハッシュを同じ変更に含め、統合テスト（`tests/`）で通常 HTML とフラグメントの両方を確認する。
