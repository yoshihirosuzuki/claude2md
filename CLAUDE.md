# claude2md プロジェクト指示

このリポジトリ固有の Claude Code 向け指示。グローバル指示（`~/.claude/CLAUDE.md`）を補完する。

## プロジェクト概要

Claude.ai のデータエクスポート ZIP を会話単位の Markdown に変換する Go 製 CLI。GitHub public 公開を前提に開発する。

## ディレクトリ構成

- `cmd/claude2md/` — CLI エントリポイント
- `internal/export/` — ZIP / JSON ストリーミング読み込み
- `internal/render/` — Markdown 整形（io.Writer ベース）
- `internal/index/` — 差分更新インデックス（`.index.json`）
- `internal/timestamp/` — RFC3339(Nano) パース集約
- `internal/render/testdata/` — 期待出力比較テスト用 fixture（`testdata/expected/` に期待出力 Markdown、架空データのみ）

## 開発コマンド

```bash
go test ./...                            # 全テスト
go test ./internal/render -update        # 期待出力ファイル再生成 (testdata/expected/*.md)
go build -o bin/claude2md ./cmd/claude2md
```

## 厳守事項

### 実データの混入禁止

README、testdata fixture、ソースコメント、エラーメッセージ、ログ、コミットメッセージ、PR 説明文に、開発者がローカルで観察した実 ZIP のデータ（会話タイトル、UUID、発言内容、ファイル名）を**一切引用しない**。

- fixture は完全架空: generic な英語、ダミー UUID（`00000000-0000-0000-0000-000000000001` のようなパターン）、切りの良いタイムスタンプ（`2024-01-01T00:00:00Z`）
- 例示・fixture の語はローカル実データから流用しない。実データに登場しない別の語を選ぶ（実プロダクト名・実在サービス名であっても、実データに出てこなければ OK）
- 「あとで差し替える」発想は事故の元。最初から架空で書く

### パストラバーサル防御は実装層で完結

`cmd/claude2md/main.go` の `isWithin(outDir, target)` が最終防衛線。攻撃者由来の `uuid` / `name` から組み立てた path が `outDir` 外を指す場合に書き込みを拒否する。`buildRelPath` の `suffixSanitize` で UUID 文字列を `[A-Za-z0-9-]` のみに制限する。これらはユーザー前提（個人/共有等）と独立した実装の正しさ。

### パーミッション設計

- ディレクトリ作成: `0o755`、ファイル作成: `0o644`（標準的な mkdir/write の挙動、umask 尊重）
- 一時ファイルは `os.CreateTemp` の OS 仕様（0o600）→ Rename 前に `os.Chmod(0o644)` で揃える
- 「個人データ前提」を理由に権限を絞らない。シンボリックリンク事前検出もしない（意図的な利用を阻害するため）

### メモリ効率: 最大会話分の中間文字列を作らない

- `internal/export/reader.go`: `json.Decoder.Decode(&c)` で直接デコード（`json.RawMessage` 経由の 2 段デコードは禁止）
- `internal/render/markdown.go`: `Render(w io.Writer, ...)` で Writer 直書き（`string` 経由は禁止）
- `internal/index/index.go`: `json.NewEncoder` でストリーム保存（`json.MarshalIndent` は禁止）
- progressbar はバイト単位（`json.Decoder.InputOffset()`）で進める。会話件数を取るための ZIP 2 回スキャンは禁止

### 設計判断の根拠

レビュー指摘に対して「対応しない」と判断する場合、以下の言い訳で済ませない:
- 「個人ツールだから」「動けば OK」
- 「実装コストが見合わない」
- 「現状の規模では問題ない」

技術的事実（公式ドキュメントの仕様、別の防御層が同等効果を持つ、言語標準ライブラリの範囲）で根拠付ける。対応しない判断は**ソースコメントまたは PR 説明**に理由を残す。

### Markdown 折りたたみ構造の保護

`<details>` ブロックの body に書き出すテキストは `internal/render/markdown.go` の `detailsBodyWriter` を経由する。body 内の `</details>` は `&lt;/details&gt;` に変換され、外側の閉じタグが破壊されないことを保証する。新たに `<details>` を出力する箇所を追加する場合も必ず `writeFoldedDetails` を経由させる。

## テスト戦略

- 標準 `testing` のみ。サードパーティ assertion 系（testify 等）は使わない
- 表駆動 + 期待出力比較（`internal/render` の `testdata/expected/*.md`）+ unit（その他）
- `go test ./internal/render -update` で `testdata/expected/*.md` を再生成。差分は PR でレビュー
- セキュリティ系（パストラバーサル、不正 uuid、書き込み失敗）は `cmd/claude2md/main_test.go` に網羅

