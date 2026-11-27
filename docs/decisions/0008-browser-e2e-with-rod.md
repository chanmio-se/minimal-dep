# ADR-0008: ブラウザ E2E は rod で最小限に検証する

- 状態: Accepted
- 日付: 2026-11-27

## Context

`tests/integration_test.go` は `HX-Request` 付き HTTP リクエストとフラグメントを検証できるが、
ブラウザが HTMX を実際に読み込み、クリックでスワップする経路は検証できない。
HTMX のバージョン更新時にブラウザ上の動作を確認する必要がある一方、Node と大きな
フレームワークを追加して E2E 用の別環境を持つことは、標準ライブラリ中心の構成と矛盾する。

## Decision

- ブラウザ E2E には `github.com/go-rod/rod` v0.116.2 を使う。ChromeDriver と Node は不要で、
  DevTools Protocol を直接操作する。バージョンは `go.mod` で固定する。
- テストは `tests/e2e/` に置き、`e2e` ビルドタグで有効化する。通常の `go test ./...` は
  ブラウザを起動せず、`make e2e` / `go test -tags=e2e ./tests/e2e/...` で実行する。
- 検証対象は HTMX のロード、フォーム送信後のフラグメント・スワップ、完了操作、
  `htmx:afterSwap` の発火、ページ遷移が起きないことに絞る。HTTP レベルの網羅的な
  振る舞いは既存の統合テストが担当する。
- ブラウザはインストール済みの Chrome / Chromium を優先し、見つからない場合のみ rod が
  固定リビジョンの Chromium をダウンロードする。CI の `e2e` job と `htmx` job が
  タグ付きテストを実行し、HTMX 更新時にスワップの回帰を検出する。
- rod は本番バイナリに含まれない開発用依存として扱い、`dependency-policy` の承認リストには
  追加しない。代わりに `internal/linter/architecture` の開発ツール対象に `tests/` を加え、
  ADR-0007 の例外を `tools/` と `tests/` に広げる。lint は `e2e` タグ付きで実行する。

## Consequences

- ブラウザ不要の `go test ./...` はこれまでどおり軽快で、オフラインでも動く。
- E2E の実行には Chrome / Chromium または初回ダウンロードが必要になる。CI では
  GitHub Actions の ubuntu runner にインストール済みの Chrome を使うため、毎回の
  ダウンロードは発生しない。
- システムの Chrome バージョンに依存するため、ブラウザ固有の挙動を厳密に固定したい場合は
  rod のダウンロードする Chromium に切り替える必要がある。今回の検証範囲では差は生じない。
- `tests/` 配下の外部 import が承認リスト検査の対象外になるため、本番コードに
  依存を持ち込まないことをレビューと `cmd/archlint` の残りのルールで担保する。
