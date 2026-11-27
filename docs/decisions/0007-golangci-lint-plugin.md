# ADR-0007: アーキテクチャ検査を golangci-lint カスタム linter に統合する

- 状態: Accepted
- 日付: 2026-11-27

## Context

`cmd/archlint` は独立 CLI としてモジュール境界・レイヤー方向・依存方針・静的セキュリティルールを検査していた。一方 golangci-lint は設定ファイルなしで別途実行しており、lint コマンドが二重に存在した。エディタ診断や CI のアノテーションでも同じルールを適用し、検証手順を一つにまとめる必要がある。

golangci-lint v2 には module plugin と Go plugin（`.so`）の二つの拡張機構がある。Go plugin は CGO と golangci-lint 本体・プラグイン間の環境／依存バージョンの完全一致が必要で、運用コストが高い。

## Decision

- `internal/linter/architecture` をルールの唯一の定義元として残し、検査関数を再利用可能にする。
- `tools/archlintplugin` に golangci-lint module plugin のアダプタを置き、Fitness Function を `go/analysis.Analyzer` として公開する。Go plugin は使わない。
- `.golangci.yml` で `archlint` を有効化し、`.custom-gcl.yml` でローカルモジュールをプラグインとして組み込む。`go tool golangci-lint custom` で `bin/custom-gcl` をビルドし、`./bin/custom-gcl run ./...` を標準の lint コマンドとする。
- golangci-lint のバージョンは `go.mod` の tool ディレクティブと `.custom-gcl.yml` の `version` で同じ値に固定し、更新時は両方を同時に変更する。
- 開発ツール専用の `tools/` 配下は `dependency-policy` の承認リスト対象外とし、プラグイン SDK の import を許可する。`tests/` 配下も開発用として同じ例外を適用する（ADR-0008）。本番コードの承認リストは変更しない。
- `cmd/archlint` はカスタムバイナリを使わない場合の standalone 実行手段として残すが、CI と `make lint` からは外す。

## Consequences

- 通常の lint 実行一回でコード品質とアーキテクチャ／セキュリティの Fitness Function の両方が検査され、エディタ統合でも同じルールが使える。
- カスタムバイナリのビルドに `git clone` と `go build` が必要になり、CI とローカルの両方でその一手順が増える。
- golangci-lint の `//nolint` や除外設定が `archlint` の指摘にも効くため、設計ルールの例外はコードレビューと ADR で管理する。
- import cycle は Go の typecheck / `go build` が先に検出するため、`module-cycle` ルールは standalone 実行と補完関係にある。カスタム linter では typecheck 失敗時に解析されないことがある。
- プラグイン SDK と `golang.org/x/tools` は `tools/` 配下だけが import し、本番バイナリの依存には含まれない。ブラウザドライバも `tests/e2e/` だけが import する。
