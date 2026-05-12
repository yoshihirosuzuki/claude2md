package render

import (
	"strings"
	"testing"
)

func TestSlug(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"plain ascii", "Hello World", "Hello-World"},
		{"japanese kept", "とあるパッケージについて", "とあるパッケージについて"},
		{"slashes replaced", "foo/bar\\baz", "foo-bar-baz"},
		{"newlines and tabs", "a\nb\tc", "a-b-c"},
		{"consecutive spaces collapsed", "a    b", "a-b"},
		{"leading and trailing whitespace", "   hi   ", "hi"},
		{"empty string", "", "untitled"},
		{"only whitespace", "   ", "untitled"},
		{"only punctuation that becomes dashes", "///   \\\\", "untitled"},
		{"forbidden filename chars", `a:b*c?d"e<f>g|h`, "a-b-c-d-e-f-g-h"},
		{"control chars", "a\x00b\x01c", "a-b-c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Slug(tt.in); got != tt.want {
				t.Errorf("Slug(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSlug_TruncatesAt80Runes(t *testing.T) {
	in := strings.Repeat("あ", 100)
	got := Slug(in)
	if runeCount(got) != 80 {
		t.Errorf("rune count = %d, want 80", runeCount(got))
	}
}

func TestSlug_TruncationDoesNotLeaveTrailingDash(t *testing.T) {
	// 80 文字目がちょうど `-` に切り詰められた直前であっても、最終出力の末尾は `-` で終わらないこと
	in := strings.Repeat("a", 80) + " trailing"
	got := Slug(in)
	if strings.HasSuffix(got, "-") {
		t.Errorf("slug %q has trailing dash", got)
	}
}

func runeCount(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
