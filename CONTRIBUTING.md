# Contributing to claude2md

claude2md への貢献を歓迎します。本ドキュメントは開発環境構築、変更フロー、コーディング規約をまとめます。

## 開発環境

- Go 1.22 以上
- GNU Make (Windows では別途インストール)
- [golangci-lint](https://golangci-lint.run/) v2 系
  - macOS: `brew install golangci-lint`
  - Linux: `curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin`
  - 公式 install ページ: <https://golangci-lint.run/docs/welcome/install/local/>
  - 注: v2 では `go install` 経由のインストールは公式に「動作保証しない」と明記されているため、バイナリインストールを使ってください。

## 開発フロー

```bash
git clone https://github.com/yoshihirosuzuki/claude2md.git
cd claude2md
make build            # bin/claude2md をビルド
make test             # 全テストを実行
make vet              # go vet
make fmt              # gofmt -w .
make lint             # golangci-lint run
make update-expected  # internal/render/testdata/expected/*.md を再生成
```

## プルリクエストフロー

1. 挙動を変える変更は issue で議題を共有してから着手することを推奨します
2. `main` から feature branch を切って変更を加える
3. `make fmt` / `make vet` / `make lint` / `make test` がクリーンであることを確認
4. PR を作成し、`PULL_REQUEST_TEMPLATE` のチェックリストを埋める
5. CI のすべてのチェックがグリーンであること
6. レビュー後マージ

## コミットメッセージ

簡潔な英語または日本語で、変更の意図を 1 行サマリ + 必要に応じて本文で説明します。Conventional Commits の prefix（`feat:` / `fix:` / `docs:` / `refactor:` / `test:` / `chore:` / `ci:` / `deps:` 等）を推奨します。GoReleaser のリリースノート生成でこれらの prefix が分類に使われます。

## コーディング規約

- `gofmt` 整形済みであること（CI で検証）
- `go vet` クリーン、`golangci-lint run` クリーン
- サードパーティ assertion ライブラリは使わない（標準 `testing` のみ）
- `internal/render/testdata/` の fixture は完全架空データ（開発者ローカルの実 ZIP からデータを流用しない）
- パストラバーサル防御（`cmd/claude2md/main.go` の `isWithin`）を経由しない出力パスを増やさない
- `<details>` 折りたたみを新規出力する場合は `internal/render/markdown.go` の `writeFoldedDetails` 経由で書く

## テスト戦略

- `internal/render` は期待出力比較テスト。意図的に出力仕様を変える場合は `make update-expected` で `testdata/expected/*.md` を再生成し、差分を PR でレビュー
- `cmd/claude2md/main_test.go` にセキュリティ系（パストラバーサル、不正 UUID、書き込み失敗時の振る舞い）を集中

## リリース（メンテナ向け）

1. `CHANGELOG.md` の `[Unreleased]` セクションをリリースバージョンに昇格、ISO 8601 日付を追加
2. `vX.Y.Z` 形式の annotated tag を作成して push:
   ```bash
   git tag -a v0.1.0 -m "v0.1.0"
   git push origin v0.1.0
   ```
3. `.github/workflows/release.yml` が GoReleaser を実行し、各 OS/arch 向けバイナリと GitHub Release が自動生成される

## 行動規範

本プロジェクトは [Contributor Covenant v2.1](CODE_OF_CONDUCT.md) を採用しています。

## ライセンス

提出されたパッチは [MIT License](LICENSE) の下で配布されることに同意したものとみなします。
