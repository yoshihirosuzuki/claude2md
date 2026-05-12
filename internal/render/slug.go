package render

import (
	"strings"
	"unicode"
)

const (
	maxSlugRunes = 80
	emptySlug    = "untitled"
)

// Slug は会話名 → ファイル名用 slug を生成する。
// - 空白・改行・制御文字、およびファイル名禁止文字 `/\:*?"<>|` を `-` に置換
// - 連続する `-` を 1 個に圧縮
// - 先頭末尾の `-` を除去
// - 80 rune に切り詰め（マルチバイト rune 単位）
// - 結果が空なら "untitled"
func Slug(name string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range name {
		if shouldReplace(r) {
			if !prevDash {
				b.WriteRune('-')
				prevDash = true
			}
			continue
		}
		b.WriteRune(r)
		prevDash = false
	}
	s := strings.Trim(b.String(), "-")
	s = truncateRunes(s, maxSlugRunes)
	s = strings.Trim(s, "-")
	if s == "" {
		return emptySlug
	}
	return s
}

func shouldReplace(r rune) bool {
	switch r {
	case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
		return true
	}
	if unicode.IsSpace(r) || unicode.IsControl(r) {
		return true
	}
	return false
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	n := 0
	for i := range s {
		if n == max {
			return s[:i]
		}
		n++
	}
	return s
}
