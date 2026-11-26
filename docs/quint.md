# Quint による状態遷移セキュリティ特性の検証

## 目的

`docs/architecture.md` の 12.4 節に基づき、状態遷移を伴うセキュリティ特性を Quint で仕様化する。
Linter はコード構造、通常のテストは実装の振る舞い、Quint は状態機械の仕様と不変条件を担当する。

| Model | File | Scope |
|---|---|---|
| Session | `spec/quint/session.qnt` | create / rotate / invalidate (expire, logout) / authenticate |
| Authz | `spec/quint/authz.qnt` | task create, owner/non-owner complete, unknown id, duplicate complete |
| CSRF | `spec/quint/csrf.qnt` | token issue / keep / form / header submit / verify-before-issue |

重要な状態・操作・前提条件・不変条件だけをモデルに含める。Quint の検証結果はモデルと明示した前提に対するものであり、
Go 実装の正しさを単独で証明するものではない。

## Model の前提

- Session: トークンは `t1`〜`t3`、主体は `alice` / `bob`。値 `""` は失効済みを表す。有効期限は時計を持たず、`expire` と `logout` は同じ失効イベントとしてモデルする。
- Session update は `rotate`（旧トークンを失効させ、同一主体に新しいトークンを発行）としてモデルする。
- Authz: タスク ID は `a`〜`c`、アクターは `alice` / `bob`。同時実行やロール変更はモデルしない。
- CSRF: トークン候補は `""`, `c1`, `c2`, `bad`。安全なリクエストでは Cookie を発行するか維持し、不安全なリクエストではフォーム項目またはヘッダで送る。
- 順序・重複・失敗例として、create 前の authenticate、expire 後の authenticate、unknown id の complete、二重 complete、verify-before-issue を含める。

## 検証コマンド

Quint CLI はリポジトリに同梱していない。Node/npm と JDK 25（Apalache 使用時）が必要である。
動作を固定するため、CLI と CI のランタイムバージョンを明示する。CI では Node 26.13.1 と Eclipse Temurin 25.0.5 を使用する。

```sh
# parse / typecheck / simulate (JDK 不要)
for m in session authz csrf; do
  npx --yes @informalsystems/quint@0.34.0 parse "spec/quint/$m.qnt"
  npx --yes @informalsystems/quint@0.34.0 typecheck "spec/quint/$m.qnt"
  npx --yes @informalsystems/quint@0.34.0 run "spec/quint/$m.qnt" --max-steps=10
done

# bounded model checking (JDK 25 が必要)
for m in session authz csrf; do
  npx --yes @informalsystems/quint@0.34.0 verify "spec/quint/$m.qnt" --max-steps=10
done
```

2026-11-27 時点で `parse` / `typecheck` / `run` は上記 CLI (v0.34.0) で実行し成功した。
`verify` は開発環境に Java Runtime がないため未実行である。`.github/workflows/ci.yml` は JDK 25 を用意した上で
`parse` / `typecheck` / `run` / `verify` と `go test ./tests/quint/...` を実行する。CI 上での `verify` の結果は
実際の CI 実行で確認する。

## トレースの生成

各 trace は Quint の simulator が `--mbt` 付きで出力した ITF JSON から選んだものである。
`mbt::actionTaken` と `mbt::nondetPicks` に加え、各状態のモデル変数（`sessions` / `tasks` / `done` / `cookie` / `submitted` / `last`）を
Go ドライバーの期待値として使用する。

```sh
mkdir -p /tmp/quint-traces
for m in session authz csrf; do
  npx --yes @informalsystems/quint@0.34.0 run "spec/quint/$m.qnt" \
    --mbt --n-traces=30 --max-steps=10 --max-samples=3000 \
    --out-itf="/tmp/quint-traces/$m.itf.json"
done
```

生成された `session<N>.itf.json` などから、必要な action をすべて含むものを選び、
`tests/quint/traces/<model>.itf.json` にコピーしてレビューする。現在のファイルは
`session215.itf.json` / `authz1.itf.json` / `csrf1.itf.json` に相当する。
モデルを変更した場合はトレースを再生成し、Go ドライバーの期待値も同じ変更単位でレビューする。

## Go トレースドライバー

`go test ./tests/quint/...` は `tests/quint/traces/` の各 trace を再生し、各 step の結果コード（`last.code`）と
全状態を比較する。標準ライブラリのみを使用する。

| Model action | Go の呼び出し | 期待値 |
|---|---|---|
| `issue` | `usecase.Login` | 発行したトークンが認証可能 |
| `rotate` | `usecase.LoginWithSession` | 旧トークン失効・新トークン有効 |
| `authenticate` / `authenticateUnknown` | `usecase.Authenticate` | 主体一致 / エラー |
| `expire` / `logout` | `usecase.Logout` | トークン失効（expiry の proxy） |
| `create` / `complete*` | tasks usecase + `WithIDFunc` | owner / forbidden / not_found / 冪等 |
| `issue` / `issueAgain` / `verify*` | `middleware.CSRF` 経由の HTTP | 200/204 or 403、Cookie 不変 |

補足:

- identity usecase は `session.Store` を受け取るため、テストでは `session.WithTokenFunc` でモデル上のトークン名をそのまま注入する。
- CSRF middleware のトークンはランダムなため、テスト内でモデル名から実トークンへの alias を張る。
- 実際の TTL 満了は `internal/platform/session` のテストが担当する。Quint の `expire` は失効イベントとして扱い、Go では `Logout` で再生する。
