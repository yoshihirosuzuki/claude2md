package render

import (
	"bytes"
	"strings"
	"testing"
)

func renderFrontmatter(t *testing.T, f Frontmatter) string {
	t.Helper()
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestFrontmatter_Minimal(t *testing.T) {
	got := renderFrontmatter(t, Frontmatter{
		UUID:         "u1",
		Name:         "hello",
		CreatedAt:    "2026-01-01T00:00:00Z",
		UpdatedAt:    "2026-01-01T00:00:00Z",
		MessageCount: 0,
	})

	if strings.Contains(got, "attachments:") {
		t.Errorf("empty attachments should be omitted; got\n%s", got)
	}
	if strings.Contains(got, "files:") {
		t.Errorf("empty files should be omitted; got\n%s", got)
	}
	if !strings.Contains(got, "message_count: 0\n") {
		t.Errorf("message_count missing; got\n%s", got)
	}
}

func TestFrontmatter_AttachmentsAndFilesDedupedAndSorted(t *testing.T) {
	got := renderFrontmatter(t, Frontmatter{
		UUID:        "u",
		Attachments: []string{"b.txt", "a.txt", "b.txt"},
		Files:       []string{"z.png", "z.png"},
	})

	if !strings.Contains(got, "attachments: [a.txt, b.txt]") {
		t.Errorf("attachments not deduped/sorted; got\n%s", got)
	}
	if !strings.Contains(got, "files: [z.png]") {
		t.Errorf("files not deduped; got\n%s", got)
	}
}

func TestFrontmatter_QuoteSpecialNames(t *testing.T) {
	cases := map[string]bool{
		"normal":           false,
		"contains: colon":  true,
		"true":             true,
		"yes":              true,
		"  leading":        true,
		"trailing  ":       true,
		"line\nbreak":      true,
		"hash # in middle": true,
		"":                 true,
	}
	for in, wantQuoted := range cases {
		got := renderFrontmatter(t, Frontmatter{Name: in})
		hasQuote := strings.Contains(got, `name: "`)
		if hasQuote != wantQuoted {
			t.Errorf("name=%q quoted=%v want=%v\nout:\n%s", in, hasQuote, wantQuoted, got)
		}
	}
}

func TestFrontmatter_Wrappers(t *testing.T) {
	got := renderFrontmatter(t, Frontmatter{})
	if !strings.HasPrefix(got, "---\n") || !strings.HasSuffix(got, "---\n") {
		t.Errorf("expected --- delimiters; got\n%s", got)
	}
}
