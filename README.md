# claude2md

[![CI](https://github.com/yoshihirosuzuki/claude2md/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/yoshihirosuzuki/claude2md/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/yoshihirosuzuki/claude2md?sort=semver)](https://github.com/yoshihirosuzuki/claude2md/releases)
[![License: MIT](https://img.shields.io/github/license/yoshihirosuzuki/claude2md)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/yoshihirosuzuki/claude2md)](https://goreportcard.com/report/github.com/yoshihirosuzuki/claude2md)

Claude.ai のデータエクスポート ZIP を、会話単位の Markdown ファイル群に変換する CLI です。

A CLI that converts Claude.ai data export ZIP archives into per-conversation Markdown files.

## Overview / 概要

Claude.ai's data export ZIP places all conversations into a single `conversations.json`, which is hard to search, browse, or feed into other tools. claude2md expands the ZIP into:

- one Markdown file per conversation
- year-month directory layout (`out/2026-04/2026-04-07_<slug>.md`)
- YAML frontmatter with metadata
- incremental updates via a `.index.json` sidecar

This lets you reuse the corpus with generic tools (grep / ripgrep), Markdown viewers, documentation pipelines, or as input to other LLMs.

The rest of this README is in Japanese only.

---

Claude.ai からダウンロードできるデータエクスポート ZIP は、すべての会話履歴が単一の `conversations.json` に格納されています。そのままでは検索・閲覧・他ツールへの投入が困難です。

claude2md は ZIP を以下の構造に展開します:

- 1 会話 = 1 Markdown ファイル
- 年月別ディレクトリ (`out/2026-04/2026-04-07_<slug>.md`)
- YAML frontmatter にメタデータ
- `.index.json` による差分更新

これにより、grep / ripgrep などの汎用ツール、Markdown ビューワ、ドキュメント生成パイプライン、別 LLM への入力など、用途を限定せず Markdown として扱えます。

## エクスポート ZIP の入手

1. <https://claude.ai/> にログイン
2. 設定 → Privacy → "Export data" を実行
3. メールで届く ZIP をダウンロード

公式: <https://support.claude.com/en/articles/9450526-how-can-i-export-my-claude-ai-data>

> **注意**: Team / Enterprise プランの組織データエクスポートは **Primary Owner のみ** 実行可能です（一般メンバー・管理者 (Owner 以外) は利用できません）。詳細は [Export your organization's data](https://support.claude.com/en/articles/13346720-export-your-organization-s-data) を参照。個人プラン（Free / Pro / Max）では本人が Settings → Privacy から実行できます。

## インストール

### バイナリ（推奨）

[Releases](https://github.com/yoshihirosuzuki/claude2md/releases) から OS / アーキテクチャ向けのアーカイブをダウンロードし、展開後 `claude2md`（Windows は `claude2md.exe`）を `PATH` 上のディレクトリに配置してください。

### `go install`

Go 1.25 以上が必要です。

```bash
go install github.com/yoshihirosuzuki/claude2md/cmd/claude2md@latest
```

### ソースから

```bash
git clone https://github.com/yoshihirosuzuki/claude2md.git
cd claude2md
make build
```

## 使い方

```bash
claude2md path/to/data-export.zip
```

オプション:

| フラグ | デフォルト | 説明 |
|---|---|---|
| `-o <dir>` | `out` | 出力ディレクトリ（カレント直下） |
| `-include-thinking` | off | `thinking` ブロックを `<details>` 折りたたみで含める |
| `-include-tools` | off | `tool_use` / `tool_result` ブロックを `<details>` 折りたたみで含める |
| `-force` | off | 差分判定を無視して全件再生成 |

オプションは ZIP パスの前後どちらに置いても動作します。ただし ZIP パスが `-` で始まる場合はフラグと誤認されて拒否されるため、`./` を先頭に付けて渡してください。

## 出力フォーマット

### ディレクトリ構造

```
out/
├── .index.json
├── YYYY-MM/
│   └── YYYY-MM-DD_<slug>.md
└── ...
```

ディレクトリ名の `YYYY-MM` とファイル先頭の `YYYY-MM-DD` は、会話の `created_at`（作成日時）を UTC に揃えて整形した値です。

`<slug>` は会話タイトルから生成（空白・改行・制御文字、およびファイル名禁止文字 `/\:*?"<>|` を `-` に置換、連続する `-` は 1 個に圧縮、最大 80 文字（rune 単位））。同一日付・同一 slug が衝突した場合は uuid を先頭 8 文字でサフィックス付与し、一意になるまで段階的に延長（8→12→16... 4 文字刻み）。

### Markdown ファイルの中身

```markdown
---
uuid: <conversation uuid>
name: <会話タイトル>
created_at: <RFC3339 タイムスタンプ>
updated_at: <RFC3339 タイムスタンプ>
message_count: <数>
attachments: [<file_name>, ...]   # 会話に含まれる添付ファイル名（あれば）
files: [<file_name>, ...]         # 会話に含まれる参照ファイル名（あれば）
---

# <会話タイトル>

## Summary

<Claude.ai が自動生成したサマリ。空なら省略>

## You

<ユーザー発言>

## Claude

<Claude 発言>

...
```

### ブロックの扱い

| ブロック種別 | デフォルト | 出力先 |
|---|---|---|
| `text` | 常に出力 | 段落として |
| `thinking` | 省略 | `--include-thinking` 時のみ `<details>` 内 |
| `tool_use` / `tool_result` | 省略 | `--include-tools` 時のみ `<details>` 内 |
| `attachments[].extracted_content` | 常に出力 | 該当メッセージ末尾の `<details>` 内 |

`<details>` の本文に `</details>` 文字列が含まれる場合は HTML エンティティに変換して、外側の折りたたみ構造が破壊されないようにします。

## 差分更新

出力ディレクトリ直下の `.index.json` で会話 uuid と updated_at を管理し、以下の判定で動作します。

| 入力側の状態 | 動作 |
|---|---|
| 未登録 | 新規作成 (`created`) |
| 登録あり、入力 `updated_at` が新しい | 上書き (`updated`)。slug 変更で path が変わった場合は旧 path を削除 |
| 登録あり、`updated_at` 同一 | スキップ (`skipped`) |
| 登録あり、入力 `updated_at` が古い | 警告して既存ファイルを保持 (`warn_older`) |

同じ ZIP を再投入してもほぼ即座に終わります。`--force` で全件再生成。

実行終了時に統計を出力:

```
Done. created: <N>, updated: <M>, skipped: <K> (<bytes> bytes scanned) in <duration>
```

## 制限事項

- `projects/*.json` と `memories.json` は出力対象外です
- エクスポート ZIP には会話 ↔ project の紐付け情報が含まれていないため、所属 project は復元できません
- `message.files[]`（画像など）は ZIP に実体が含まれないため、frontmatter にファイル名のみ記録します

## Contributing

主要な開発コマンドは `Makefile` にまとめてあります。

```bash
make build          # bin/claude2md をビルド
make test           # 全テスト
make update-expected  # 期待出力ファイル (testdata/expected/*.md) の再生成
make vet            # go vet
make fmt            # gofmt -w .
make clean          # bin を削除
```

詳細な開発手順、PR フロー、コミット規約は [CONTRIBUTING.md](CONTRIBUTING.md) を参照してください。

## Acknowledgments

本ツールは以下のオープンソースライブラリを利用しています。

- [schollz/progressbar](https://github.com/schollz/progressbar) — 進捗バー表示（MIT License）
- [golang.org/x/term](https://pkg.go.dev/golang.org/x/term) — 端末判定（BSD-3-Clause License）

## ライセンス

MIT License。詳細は [LICENSE](LICENSE) を参照。
