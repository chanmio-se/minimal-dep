# ADR-0005: GitHub Actions の依存を lockfile で固定する

- 状態: Accepted
- 日付: 2026-11-27

## Context

`.github/workflows/ci.yml` は `actions/checkout@v4` など、タグで参照するアクションを使っている。タグとブランチは後から差し替えられるため、レビューしたコードと実行されるコードが一致しない可能性がある。2026-10-28 に GitHub Actions の workflow dependency lockfile が一般提供となり、推移的依存を含むアクションをコミット単位で固定できるようになった。

Go の依存と同様に、CI の依存も変更をレビュー可能である必要がある。外部 Go ライブラリを増やさず、既存の CI と同じ PR フローで検証できる仕組みが望ましい。

## Decision

- ロックファイルは `.github/workflows/actions.lock` に置き、`gh actions-lock` で生成・更新する。手編集はしない。
- すべての workflow を対象にし、`uses:` を追加・変更・削除したときは同じ変更で `gh actions-lock` を再実行し、workflow と lockfile を一緒にコミットする。
- workflow 側の参照は可読性を保つ。lockfile が固定したコミットを実行し、workflow 内をフル SHA で固定する方式は採らない。推移的依存も lockfile の対象に含める。
- `gh actions-lock --relock` は参照が移動した場合にのみ、上流の履歴を確認してから実行する。`--accept-moved` は自動化せず、調査後の手動操作に限定する。
- `.github/workflows/verify-lockfile.yml` で `github/gh-actions-lock@verify/v0.5.1` を実行し、push と pull request ごとに `gh actions-lock --verify` 相当の検査を行う。`.github/dependabot.yml` で GitHub Actions の更新を毎週確認し、version を更新するときは lockfile の対応エントリも同じ PR で更新する。
- lockfile の差分は依存の更新としてレビューする。`commit` の変更は実行コードの変更、`owner_id` / `repo_id` の変更は取得元リポジトリの変更として扱う。
- リポジトリ / Organization の **Require lockfile** ポリシーは管理者設定で必須化する。設定画面の操作はリポジトリ内のファイルでは表現できないため、ここでは CI の検証を必須化の代替手段として明示する。

## Consequences

- `gh actions-lock` は部分バージョンを完全なセマンティックバージョンへ置き換えるため、workflow の `uses:` と lockfile の両方が差分に現れる。
- CI のアクション参照と推移的依存は固定されるが、reusable workflow の呼び出しとコンテナイメージは対象外である。追加した場合はフル SHA または `@sha256:` ダイジェストで固定する。
- 検証は上流リポジトリへの参照が必要である。プライベート / 内部アクションを追加した場合は、読み取り権限を持つトークンを verify ステップへ渡すか、`--verify-local` によるカバレッジ検査に切り替える。
- lockfile は依存が安全であることを保証しない。固定する前に依存をレビューする運用を維持する。
