# Architecture.md

# Go + HTMX Web Framework

## 1. Overview

本フレームワークは、Go と HTMX を利用して、シンプルで保守性の高い Web アプリケーションを構築するための軽量フレームワークである。

Rails のような開発体験を目指しながら、複雑な抽象化や大量の依存ライブラリを持たず、Go の標準ライブラリを中心に構成する。

### Goals

- HTML を中心とした Web アプリケーションを構築できる
- MPA を基本とし、HTMX による部分更新をサポートする
- Go の標準ライブラリを最大限活用する
- 明確なディレクトリ構成と責務を提供する
- テストしやすく、AI によるコード生成・変更にも適した構造にする
- フレームワーク自体の依存関係と保守コストを小さくする

### Initial Scope

初期バージョンは、サーバーサイドレンダリングを中心とした単一プロセスの Web アプリケーションに焦点を当てる。

- MPA と HTMX によるサーバー主導の画面更新
- Go 標準ライブラリを中心とする HTTP・データベース処理
- コンストラクタによる明示的な依存関係の組み立て
- 必要な画面機能に絞った JavaScript
- 単一バイナリとしてのビルドとデプロイ

## 2. Core Principles

### 2.1 Server-side Rendering First

画面の HTML はサーバー側で生成する。

通常の HTTP リクエストには HTML ドキュメントを返す。

HTMX リクエストには、更新対象となる HTML フラグメントを返す。

画面状態の中心をサーバー側に置き、クライアントには必要な UI 状態と操作を担当させる。

### 2.2 HTML over JSON

画面表示を目的とした通信では JSON ではなく HTML を返す。

```text
Browser
  |
  | GET /tasks
  v
Go Handler
  |
  v
Usecase
  |
  v
Repository
  |
  v
Go Template
  |
  v
HTML Response
```

JSON API が必要な場合は、画面表示用の HTML エンドポイントとは別に提供する。

### 2.3 Explicit over Magic

暗黙的な振る舞いを減らす。

- ルーティングは明示する
- 依存関係はコンストラクタで渡す
- テンプレートの呼び出しを明示する
- エラー処理を明示する
- データベースアクセスを明示する

適切な Convention over Configuration を採用しつつ、依存関係はコンストラクタで明示的に組み立てる。

### 2.4 Standard Library First

Go 標準ライブラリを優先する。

| 領域 | 基本方針 |
|---|---|
| HTTP | net/http |
| Routing | net/http.ServeMux |
| HTML | html/template |
| JSON | encoding/json |
| Logging | log/slog |
| Testing | testing / httptest |
| Database | database/sql |
| Configuration | os / flag |
| Context | context |

外部ライブラリは、標準ライブラリで実装するよりも明確なメリットがある場合に限り採用する。

## 3. Architecture

### 3.1 Modular Monolith

アプリケーションは単一のプロセス・デプロイ単位を維持し、ビジネス機能ごとのモジュールに分割する。各モジュールの内部ではレイヤーを分け、モジュール境界を越える依存は公開 API に限定する。

```text
cmd/server
    |
    v
Application Composition --wires--> Module A
              |                       |
              +--wires--> Module B    +-- Handler / Usecase / Domain / Repository
                          |            +-- API
                          +--depends on--> Module A API
              |
              +-- Module A / B use --> Shared Platform
                                        (HTTP, Database, Templates, Logging)
```

各モジュール内の責務は以下のとおり。

| Layer | Responsibility |
|---|---|
| Handler | HTTP Request / Response |
| Usecase | アプリケーションのユースケース |
| Domain | ドメインルール・ビジネスロジック |
| Repository | データ永続化 |
| Template | HTML 表示 |

モジュール間の依存ルール：

- 各モジュールは、自身のドメインとユースケースの所有者となる。
- 他モジュールの機能は、そのモジュールの `api` パッケージを通じて利用し、Handler、Domain、Repository などの内部実装をカプセル化する。
- モジュール依存を一方向に保つ。複数モジュールにまたがる処理は Application Composition またはユースケース層で調整する。
- `platform` は業務ルールから独立した技術的な共通機能を提供する。ドメインロジックは、それを所有するモジュール内に配置する。
- 全モジュールを同じアプリケーション内で実行し、論理的なモジュール境界によって将来の分割可能性と変更の局所性を保つ。

### 3.2 Handler

Handler は HTTP に関する処理だけを担当する。

責務：

- Request の受け取り
- Path / Query / Form の解析
- 入力値のバリデーション
- Usecase の呼び出し
- HTTP Status の決定
- HTML / JSON のレスポンス生成

Handler は HTTP の入出力を扱い、ビジネスロジックは Usecase と Domain に委ねる。

### 3.3 Usecase

Usecase はアプリケーションの操作単位を表現する。

例：

- CreateTask
- UpdateTask
- CompleteTask
- ListTasks

責務：

- 業務処理の実行
- Domain の呼び出し
- Repository を通じたデータ取得・保存
- トランザクション境界の管理

Usecase は HTTP や HTMX から独立させ、アプリケーションの操作単位を表現する。

### 3.4 Domain

Domain はビジネスルールを表現する。

- Entity
- Value Object
- Domain Service

Domain は HTTP、HTML、DB の実装詳細から独立させる。

### 3.5 Repository

Repository は永続化の責務を持つ。

初期バージョンでは `database/sql` を基本とし、ORM は必要性と採用理由を評価したうえで選択する。

## 4. HTMX Integration

HTMX はサーバー主導の画面更新を実現するために利用する。

### 4.1 Request Handling

HTMX リクエストは `HX-Request` ヘッダーで判定する。

```go
func IsHTMXRequest(r *http.Request) bool {
    return r.Header.Get("HX-Request") == "true"
}
```

### 4.2 Response Strategy

同じ URL でもリクエストの種類によって返却する HTML を変更できる。

```text
GET /tasks

通常リクエスト
  -> layouts/app.html
     + pages/tasks.html

HTMX リクエスト
  -> fragments/tasks-list.html
```

HTMX 固有の判定とレスポンス生成は共通処理に集約し、Handler から共通処理を利用する。

### 4.3 HTML Fragment

HTML フラグメントは独立したテンプレートとして管理する。

例：

```html
<div id="task-list">
  {{ range .Tasks }}
    <div class="task">
      {{ .Title }}
    </div>
  {{ end }}
</div>
```

HTMX の `hx-target` と `hx-swap` によって、更新対象を制御する。

### 4.4 Form Submission

フォーム送信は通常の HTML Form を基本とする。

HTMX を利用する場合も、通常のフォーム送信にフォールバックできる設計を優先する。

### 4.5 JavaScript Policy

JavaScript は必要な画面機能に絞って使う。

- 基本的な画面更新は HTMX で行う
- 独自 JavaScript は必要な機能に限定する
- UI コンポーネントは軽量な実装を基本とし、大規模なフロントエンド基盤は明確な必要性がある場合に選択する
- JavaScript の依存ライブラリを追加する場合は採用理由を記録する

## 5. Directory Structure

アプリケーションの基本構成は以下とする。`tasks` と `identity` は機能モジュールの例であり、実際のモジュールは要件に応じて定義する。

```text
.
├── cmd/
│   ├── server/
│   │   └── main.go
│   └── modulegen/                 # モジュール雛形を生成する開発用 CLI
│       └── main.go
│
├── internal/
│   ├── app/
│   │   └── app.go                 # モジュールの生成と依存関係の組み立て
│   ├── generator/
│   │   └── module/                # モジュール生成ロジックと雛形
│   ├── platform/                  # 業務ルールから独立した技術的共通機能
│   │   ├── database/
│   │   ├── http/
│   │   ├── logging/
│   │   └── templates/
│   └── modules/
│       ├── tasks/
│       │   ├── api/               # 他モジュールに公開する契約
│       │   ├── domain/
│       │   ├── usecase/
│       │   ├── repository/
│       │   ├── handler/
│       │   ├── templates/
│       │   └── module.go          # モジュールの生成・HTTP 登録
│       └── identity/
│           ├── api/
│           ├── domain/
│           ├── usecase/
│           ├── repository/
│           ├── handler/
│           ├── templates/
│           └── module.go
│
├── web/                            # 全モジュールで共有する静的ファイル
│   └── static/
│       ├── css/
│       └── js/
│
├── migrations/                     # 適用順を一元管理する DB マイグレーション
│
├── tests/                          # モジュール横断の統合テスト
│
├── go.mod
├── go.sum
└── docs/
    └── architecture.md
```

モジュール固有の単体テストは各モジュールの `*_test.go` に置き、モジュール横断の統合テストは `tests/` に置く。テンプレートも原則として所有するモジュール内で管理する。

`app` は Composition Root として、モジュールの生成、依存の注入、ルート登録を明示する。初期段階では必要な境界に沿って構成し、細かなパッケージ分割よりも明確な責務を優先する。

### 5.1 Module Generator

`cmd/modulegen` は新しい機能モジュールの初期雛形を生成する開発用 CLI とする。生成ロジックと雛形は `internal/generator/module` に置き、アプリケーションの実行時コードから分離する。

- 生成する構成は本書のモジュール境界と依存ルールに従う。
- モジュール名などの入力を検証し、既存ファイルを保護しながら生成する。
- Composition Root への登録方法を利用者に案内し、依存関係を明示的に組み立てる。
- 生成物は通常のソースコードとして編集・保守できる形にし、実行時はアプリケーション単体で動作させる。

## 6. Routing

ルーティングは `net/http.ServeMux` を基本とする。

```go
mux := http.NewServeMux()

mux.HandleFunc("GET /tasks", taskHandler.Index)
mux.HandleFunc("POST /tasks", taskHandler.Create)
mux.HandleFunc("POST /tasks/{id}/complete", taskHandler.Complete)
```

ルーティング定義はアプリケーションの起動時に明示的に登録する。

## 7. Template Management

HTML テンプレートは `html/template` を利用する。

テンプレートは以下の単位で管理する。

- Layout
- Page
- Fragment

テンプレートは表示に専念させ、ビジネスロジックは Usecase と Domain に置く。

HTML のエスケープは `html/template` の標準動作に任せる。

## 8. Middleware

Middleware は HTTP 横断処理を担当する。

初期状態で想定する Middleware：

- Request ID
- Access Log
- Panic Recovery
- Security Headers
- CSRF Protection
- Authentication の共通処理

Middleware は HTTP 層の関心事に限定する。

Security Headers と CSRF Protection は共通 Middleware として提供し、アプリケーションの標準 HTTP 構成に組み込む。認証の共通処理は Middleware で提供し、認可ポリシーは各モジュールの Usecase / Domain が所有する。

## 9. Error Handling

エラーはレイヤーごとに責務を分ける。

- Domain: 業務上のエラー
- Usecase: ユースケース実行上のエラー
- Repository: 永続化に関するエラー
- Handler: HTTP Status と画面表示への変換

エラーは発生レイヤーに応じて明示的に伝達し、必要な文脈とともに記録する。

ユーザーには適切なエラーメッセージを返し、内部詳細はログに記録して運用者が調査できるようにする。

## 10. Testing Strategy

テストはレイヤーごとに実施する。

| Layer | Test |
|---|---|
| Domain | Unit Test |
| Usecase | Unit Test |
| Repository | Integration Test |
| Handler | HTTP Test |
| HTMX | Browser / E2E Test |

### 10.1 Handler Test

`net/http/httptest` を利用する。

HTTP Status、Header、Response Body を検証する。

### 10.2 HTMX Test

以下を重点的に検証する。

- 通常リクエストで完全な HTML が返ること
- HTMX リクエストで適切な Fragment が返ること
- `HX-Request` の判定が正しいこと
- Form Submission が正しく動作すること
- エラー時のレスポンスが適切であること
- DOM の更新対象が期待どおりであること

ブラウザ上での動作は Playwright 等による E2E テストで検証する。

## 11. Dependency Policy

フレームワークの長期保守性を重視し、標準ライブラリを基盤に採用価値の明確な依存ライブラリを選定する。

### Principles

1. Go 標準ライブラリで実現できる場合は標準ライブラリを使う
2. 外部ライブラリを追加する場合は採用理由を記録する
3. 小さな機能は標準ライブラリや既存の仕組みで実装し、大きなフレームワークは明確な利点がある場合に選択する
4. JavaScript の依存関係は、機能上の利点を確認して必要なものを選択する
5. 依存ライブラリの更新を自動検知できるようにする
6. バージョン更新時はテストで振る舞いを検証する

### HTMX

HTMX は外部 JavaScript ライブラリとして扱う。

バージョンは `package.json` / `package-lock.json` の devDependency で固定し、`scripts/sync-htmx.mjs` で `web/static/js/htmx.min.js` に複製する。npm は更新時の開発ツールとし、アプリの実行時や通常の Go ビルドには使わない。Dependabot の npm 更新 PR では同期スクリプトを実行して同梱ファイルと ADR を更新し、CI で差分がないことを検証する。更新時には変更内容と E2E テスト結果を確認する。

## 12. Security

フレームワークは、安全な実装を標準経路として提供し、各アプリケーションが一貫した対策を実装できるよう支援する。

### 12.1 Secure Defaults and Shared Components

- HTTP サーバーの標準構成に Security Headers、CSRF Protection、リクエストタイムアウト、リクエストボディサイズ制限を含める。
- CSRF Middleware とフォーム用ヘルパーを提供し、通常のフォームと HTMX リクエストの両方でトークンを検証する。HTMX クライアントには共通ヘルパーを通じてトークンを送信する。
- Session Cookie は `HttpOnly`、`SameSite`、HTTPS 環境での `Secure` を安全な初期値とし、環境に応じた設定方法を提供する。セッションの生成・更新・失効は共通の仕組みに集約する。
- 認証済みユーザーの識別情報を HTTP 層から Usecase に明示的に渡す。操作ごとの認可判断は各モジュールの Usecase / Domain に置く。
- 入力値は Handler で形式・サイズ・範囲を検証し、Domain で業務上の不変条件を検証する。
- SQL はプレースホルダーを使う Repository の実装例を標準パターンとして示す。HTML は `html/template` の自動エスケープを標準経路とする。
- 内部エラーの詳細はログに記録し、HTTP レスポンスには利用者向けの安全なメッセージを返す。

### 12.2 Generator and Security Tests

Module Generator は、安全な HTTP 登録方法、CSRF 対応フォーム、入力検証、認証情報を受け取る Usecase の例を生成する。各生成例には、成功ケースに加えて未認証・権限不足・CSRF トークン不正などのセキュリティテストを含める。

フレームワーク共通の Middleware とヘルパーには、次の振る舞いを検証するテストを用意する：

- CSRF トークンのない、または不正な変更系リクエストを拒否する
- Session Cookie に設定したセキュリティ属性がレスポンスに反映される
- Security Headers とリクエスト制限が標準 HTTP 構成で適用される
- 未認証・認可失敗時に適切な HTTP レスポンスを返す

### 12.3 Security Fitness Functions

セキュリティ上のアーキテクチャ特性を Fitness Function として CI で継続検証する：

- モジュール間依存が公開 API を通ること、および認可の所有者が Usecase / Domain であること
- SQL 呼び出しがパラメーター化され、HTML 出力が標準テンプレート経由でエスケープされること
- セキュリティ Middleware がアプリケーションの標準 HTTP 構成に含まれること
- CSRF、Cookie 属性、認証・認可、Security Headers の振る舞いテストが通ること

静的解析で判定できるルールは Linter に、リクエストやレスポンスの実際の振る舞いはテストに置き、検査結果を CI で確認できるようにする。

## 12.4 Quint for State-based Security Properties

Quint は、状態遷移を伴うセキュリティ特性を仕様化し、モデル検査・シミュレーションする補完的な手段として利用する。Linter はコード構造、通常のテストは実装の振る舞い、Quint は状態機械の仕様と不変条件をそれぞれ検証する。

適用候補：

- セッションの作成、更新、失効と認証状態の遷移
- 認証・認可における許可条件と権限の不変条件
- CSRF トークンの発行、送信、検証の状態遷移
- 重要な操作における順序、重複実行、失敗時の状態維持

モデルは重要な状態、操作、前提条件、不変条件に絞り、明示した範囲で検査する。Quint の検証結果はモデルとその前提に対する結果として扱い、Go 実装の正しさはそれだけで結論づけない。

実装との対応は、モデルから生成したトレースを Go のテストドライバーで実行し、各操作の結果や状態を期待値と照合して確認する。モデル、テストドライバー、実装の変更を同じ変更単位でレビューし、仕様と実装のずれを継続的に検知する。CI では Quint の検証と対応する Go テストを実行し、反例が得られた場合は再現可能なテストケースとして扱う。

## 13. Operational Requirements

- Graceful Shutdown
- Structured Logging
- Health Check
- Request Timeout
- Context Propagation
- Configuration via Environment Variables

アプリケーションは単一の Go バイナリとしてビルド・実行できる構成を基本とする。

---

## 14. Design Decisions

以下は初期設計における決定事項である。

| Decision | Choice |
|---|---|
| Backend Language | Go |
| HTTP Server | net/http |
| Router | net/http.ServeMux |
| View | html/template |
| UI Interaction | HTMX |
| Rendering | Server-side Rendering |
| Database Access | database/sql |
| Logging | log/slog |
| Frontend Build | 原則不要 |
| JavaScript | 最小限 |
| Architecture | Modular Monolith with internal layers |
| Deployment | Single Binary |

今後、重要な設計変更は ADR として記録する。

## 15. AI-Assisted Development and Quality Harness

AI エージェントが既存の設計意図を理解し、自律的に実装できることと、変更の品質を機械的に検証できることを両立する。

### Responsibilities

| Mechanism | Responsibility |
|---|---|
| `AGENTS.md` | リポジトリ共通の設計方針、作業上の制約、完了条件を伝える |
| Skills | 特定の作業で必要となる専門知識や再利用可能な手順を提供する |
| Linter | 明確な規約違反や要改善パターンを機械的に検出し、アーキテクチャの Fitness Function を形成する |
| Tests | アプリケーションの振る舞いが要件を満たすことを検証する |
| Human review | 要件の解釈やトレードオフなど、機械判定できない設計判断を確認する |

### Operating Principles

1. `AGENTS.md` は目的、境界、制約、検証方法を示し、それらを満たす実装方法の自律的な選択を支援する。
2. Skills はタスクに応じた作業手順や専門的な判断材料を提供し、共通ルールは `AGENTS.md` に集約する。
3. Linter は決定的で説明可能なルールを検査し、違反箇所と修正の手掛かりを明確にする。設計上の好みや文脈依存の判断は人によるレビューで扱う。
4. Linter とテストは相互補完の品質確認手段として使い、規約の静的検査には Linter、実行時の振る舞いの検証にはテストを適用する。
5. 変更の完了時には、該当する Linter とテストを実行し、未実行の確認項目や失敗を明示する。
6. 新しい規則は、繰り返し発生する問題や重要な境界に対応する範囲で追加する。機械検査できる規則は Linter を唯一の定義元とし、指示文から参照する。

### Fitness Functions

Fitness Function は、アーキテクチャが意図した性質を保っているかを継続的に判定する自動検査である。Linter と静的解析を CI で実行し、設計方針をコード変更のたびに検証できる状態にする。

優先して検査する性質：

- モジュール間の依存が公開 `api` を通り、一方向に保たれていること
- モジュール間に循環依存がなく、各モジュールの内部実装が境界内に保たれていること
- Handler、Usecase、Domain、Repository の依存方向と責務が保たれていること
- 依存ライブラリが定めた採用方針と承認済みの依存関係に沿っていること
- CI が使うアクションが lockfile に記録されたコミットで実行されること

CI の依存は `.github/workflows/actions.lock` で固定し、workflow の変更と同じ PR で更新する。lockfile と workflow の整合性だけでなく、上流のタグが移動していないことも pull request ごとに検証する（[ADR-0005](decisions/0005-actions-lockfile.md)）。

各 Fitness Function は、検査対象、合格条件、違反時の説明を明確に定義する。静的解析で表現できる性質は Linter のルールにし、実行時の振る舞いを伴う性質はテストで検証する。検査結果は CI 上で確認可能にし、アーキテクチャの変更に合わせてルールも更新する。