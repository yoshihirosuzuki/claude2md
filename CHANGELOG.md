# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-05-12

### Added

- Initial public release.
- Convert Claude.ai data export ZIP into per-conversation Markdown files.
- YAML frontmatter with `uuid`, `name`, `created_at`, `updated_at`, `message_count`, `attachments`, `files`.
- Year-month directory layout: `out/YYYY-MM/YYYY-MM-DD_<slug>.md`.
- Incremental update via `.index.json` sidecar (`-force` to regenerate all).
- `-include-thinking` / `-include-tools` flags for optional folded `<details>` blocks.
- Streaming JSON / ZIP reading with byte-based progress bar (TTY: cyan ANSI; non-TTY: plain).
- Path traversal defense (`isWithin` + `suffixSanitize`) and atomic writes.

[Unreleased]: https://github.com/yoshihirosuzuki/claude2md/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/yoshihirosuzuki/claude2md/releases/tag/v0.1.0
