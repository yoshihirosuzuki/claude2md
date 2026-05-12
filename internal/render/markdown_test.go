package render

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshihirosuzuki/claude2md/internal/export"
)

var update = flag.Bool("update", false, "update expected output files in testdata/expected/")

func loadFixture(t *testing.T) *export.Conversation {
	t.Helper()
	data, err := os.ReadFile("testdata/sample_conversation.json")
	if err != nil {
		t.Fatal(err)
	}
	var c export.Conversation
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	return &c
}

func renderToString(t *testing.T, c *export.Conversation, opt Options) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Render(&buf, c, opt); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestRender_Golden(t *testing.T) {
	c := loadFixture(t)
	cases := []struct {
		name string
		opt  Options
	}{
		{"default", Options{}},
		{"with_thinking", Options{IncludeThinking: true}},
		{"with_tools", Options{IncludeTools: true}},
		{"with_all", Options{IncludeThinking: true, IncludeTools: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := renderToString(t, c, tc.opt)
			expected := filepath.Join("testdata", "expected", tc.name+".md")
			if *update {
				if err := os.MkdirAll(filepath.Dir(expected), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(expected, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(expected)
			if err != nil {
				t.Fatalf("missing expected output file %s; run `make update-expected` first", expected)
			}
			if got != string(want) {
				t.Errorf("mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", tc.name, got, string(want))
			}
		})
	}
}

func TestRender_FrontmatterAggregatesAttachmentsAndFiles(t *testing.T) {
	c := loadFixture(t)
	got := renderToString(t, c, Options{})
	if !strings.Contains(got, "attachments: [notes.txt]") {
		t.Errorf("expected aggregated attachments in frontmatter; got\n%s", got)
	}
	if !strings.Contains(got, "files: [image.png]") {
		t.Errorf("expected aggregated files in frontmatter; got\n%s", got)
	}
}

func TestRender_EmptyConversation(t *testing.T) {
	c := &export.Conversation{
		UUID:      "u1",
		Name:      "empty",
		CreatedAt: "2026-01-01T00:00:00Z",
		UpdatedAt: "2026-01-01T00:00:00Z",
	}
	got := renderToString(t, c, Options{})
	if !strings.Contains(got, "message_count: 0") {
		t.Errorf("expected message_count: 0; got\n%s", got)
	}
	if !strings.Contains(got, "# empty") {
		t.Errorf("expected H1 title; got\n%s", got)
	}
}

// TestRender_PropagatesWriteError は io.Writer エラーが正しく伝搬することを確認する。
func TestRender_PropagatesWriteError(t *testing.T) {
	c := loadFixture(t)
	w := &shortWriter{n: 10} // 10 バイトでエラー
	err := Render(w, c, Options{})
	if err == nil {
		t.Errorf("expected error from short writer")
	}
}

type shortWriter struct {
	n       int
	written int
}

func (s *shortWriter) Write(p []byte) (int, error) {
	if s.written >= s.n {
		return 0, errShortWriter
	}
	allowed := s.n - s.written
	if allowed >= len(p) {
		s.written += len(p)
		return len(p), nil
	}
	s.written = s.n
	return allowed, errShortWriter
}

var errShortWriter = &writerError{"short writer triggered"}

type writerError struct{ msg string }

func (e *writerError) Error() string { return e.msg }

func toolResultString(blk export.ContentBlock) string {
	var buf bytes.Buffer
	writeToolResultText(&buf, blk)
	return buf.String()
}

func TestWriteToolResultText_StringContent(t *testing.T) {
	got := toolResultString(export.ContentBlock{
		Type:          "tool_result",
		Name:          "X",
		ResultContent: json.RawMessage(`"plain string output"`),
	})
	if got != "plain string output" {
		t.Errorf("string content not extracted; got %q", got)
	}
}

func TestWriteToolResultText_ArrayContent(t *testing.T) {
	got := toolResultString(export.ContentBlock{
		Type:          "tool_result",
		Name:          "X",
		ResultContent: json.RawMessage(`[{"type":"text","text":"a"},{"type":"text","text":"b"}]`),
	})
	if !strings.Contains(got, "a") || !strings.Contains(got, "b") {
		t.Errorf("array content not joined; got %q", got)
	}
}

func TestWriteToolResultText_FallbackToDisplayContent(t *testing.T) {
	got := toolResultString(export.ContentBlock{
		Type:           "tool_result",
		Name:           "X",
		DisplayContent: json.RawMessage(`{"key":"value"}`),
	})
	if !strings.Contains(got, "value") {
		t.Errorf("display_content fallback failed; got %q", got)
	}
}

func TestWriteToolResultText_Empty(t *testing.T) {
	got := toolResultString(export.ContentBlock{Type: "tool_result", Name: "X"})
	if got != "(empty result)" {
		t.Errorf("empty result fallback failed; got %q", got)
	}
}

func TestWriteToolResultText_NonStandardObjectContent(t *testing.T) {
	got := toolResultString(export.ContentBlock{
		Type:          "tool_result",
		Name:          "X",
		ResultContent: json.RawMessage(`{"questions":[{"q":"a"}]}`),
	})
	if !strings.Contains(got, "questions") {
		t.Errorf("object fallback should print raw json; got %q", got)
	}
}
