# ADR-0001: 標準ライブラリと HTMX の採用

- 状態: Accepted
- 日付: 2026-11-27

## Context

軽量で長期保守しやすいフレームワークとし、AI によるコード生成・変更にも適した明確な構成が必要である。フロントエンドのビルドを増やさず、サーバー主導の画面更新を実現する必要がある。

## Decision

- HTTP / ルーティング / HTML / DB / ログ / テストには Go 標準ライブラリのみを使う。
- 画面更新には HTMX `2.0.11` を利用し、`web/static/js/htmx.min.js` にバージョン固定で同梱する。
- HTMX は外部 JavaScript ライブラリとして扱い、Go モジュールの依存には追加しない。
- `package.json` と `package-lock.json` に `htmx.org` を devDependency として固定する。npm は同梱ファイルの更新時だけ使う開発ツールとし、アプリの実行時や通常の Go ビルドには使わない。
- `.github/dependabot.yml` の npm エコシステムで更新を確認する。更新 PR では `npm ci` と `npm run sync:htmx` を実行し、npm パッケージから同梱 JS を複製して ADR のバージョンとハッシュを更新する。
- CI では `npm ci` → `npm run sync:htmx` → `git diff --exit-code` で同梱 JS と ADR の差分がないことを確認し、`go test ./...`（`tests/` の統合テストを含む）を必須条件とする。

SHA-256 (htmx.min.js 2.0.11):
`d6fdc75f204e6bdefa99b69bf1e6d4ac69b8a364f77929f45c13476b4000f717`

## Consequences

- 生成済み JS をコミットするため、単一バイナリで配布でき、フロントエンドのビルドは不要である。
- 新しい Go 依存を追加する場合は ADR と `internal/linter/architecture` の承認リストを更新する必要がある。
- Dependabot の npm 更新 PR だけでは同梱 JS と ADR は更新されない。PR 内で `npm run sync:htmx` を実行し、生成された差分をコミットする運用が必要である。
- `node_modules/` はコミットしない。HTMX を更新する場合は、同梱 JS と ADR のバージョン・ハッシュを同じ変更に含め、統合テスト（`tests/`）で通常 HTML とフラグメントの両方を確認する。
