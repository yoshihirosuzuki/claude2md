# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- `internal/render`: add an overflow guard on the chunk buffer
  allocation in `detailsBodyWriter.Write` to satisfy CodeQL's
  `go/allocation-size-overflow` query. Defensive — the bound is
  unreachable in practice (`d.pending` is at most 9 bytes by
  construction, derived from the `</details>` partial-match buffer).

## [0.3.0] - 2026-05-13

### Added

- `-version` flag prints the version. GoReleaser binaries embed the release
  tag via `-ldflags -X main.version={{.Tag}}`; `go install` builds recover
  it from the module info via `runtime/debug`; local source builds report `dev`.

### Fixed

- LICENSE: remove a spurious "OF" from the warranty clause ("THE USE OF OR
  OTHER DEALINGS" → "THE USE OR OTHER DEALINGS") so that the file matches
  the SPDX MIT template. The typo prevented google/licensecheck (which
  pkg.go.dev uses) from identifying the license, leading to a
  `License: UNKNOWN` display on the module page. Verified locally with
  google/licensecheck v0.3.1: coverage rose from 0.00% to 98.82% with
  `ID="MIT"`.

## [0.2.0] - 2026-05-13

### Changed

- Bump `golang.org/x/term` from 0.28.0 to 0.43.0 (indirect `golang.org/x/sys` 0.29.0 → 0.44.0).

### Removed

- Drop support for Go versions below 1.25 — minimum required Go version is now 1.25 (was 1.22). The bump is forced by `golang.org/x/term` v0.43.0 declaring `go 1.25.0`; Go 1.22 is also out of upstream support since 2025-02-11. Source builds and `go install` now require Go 1.25 or newer; pre-built binaries from GitHub Releases are unaffected.

## [0.1.0] - 2026-05-13

### Added

- Initial public release.
- Convert Claude.ai data export ZIP into per-conversation Markdown files.
- YAML frontmatter with `uuid`, `name`, `created_at`, `updated_at`, `message_count`, `attachments`, `files`.
- Year-month directory layout: `out/YYYY-MM/YYYY-MM-DD_<slug>.md`.
- Incremental update via `.index.json` sidecar (`-force` to regenerate all).
- `-include-thinking` / `-include-tools` flags for optional folded `<details>` blocks.
- Streaming JSON / ZIP reading with byte-based progress bar (TTY: cyan ANSI; non-TTY: plain).
- Path traversal defense (`isWithin` + `suffixSanitize`) and atomic writes.

[Unreleased]: https://github.com/yoshihirosuzuki/claude2md/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/yoshihirosuzuki/claude2md/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/yoshihirosuzuki/claude2md/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/yoshihirosuzuki/claude2md/releases/tag/v0.1.0
