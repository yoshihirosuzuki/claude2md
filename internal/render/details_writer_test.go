package render

import (
	"bytes"
	"strings"
	"testing"
)

func TestDetailsBodyWriter_EscapesCloseTag(t *testing.T) {
	var buf bytes.Buffer
	bw := newDetailsBodyWriter(&buf)
	bw.Write([]byte("foo </details> bar"))
	bw.Flush()
	got := buf.String()
	if strings.Contains(got, "</details>") {
		t.Errorf("close tag not escaped: %q", got)
	}
	if !strings.Contains(got, "&lt;/details&gt;") {
		t.Errorf("entity missing: %q", got)
	}
}

func TestDetailsBodyWriter_EscapesUppercase(t *testing.T) {
	var buf bytes.Buffer
	bw := newDetailsBodyWriter(&buf)
	bw.Write([]byte("foo </Details> bar </DETAILS> baz"))
	bw.Flush()
	got := buf.String()
	if strings.Contains(got, "</Details>") || strings.Contains(got, "</DETAILS>") {
		t.Errorf("case-insensitive escape failed: %q", got)
	}
}

func TestDetailsBodyWriter_HandlesChunkBoundary(t *testing.T) {
	// 閉じタグがチャンク境界をまたぐ場合
	var buf bytes.Buffer
	bw := newDetailsBodyWriter(&buf)
	bw.Write([]byte("foo </de"))
	bw.Write([]byte("tails> bar"))
	bw.Flush()
	got := buf.String()
	if strings.Contains(got, "</details>") {
		t.Errorf("close tag across chunks not escaped: %q", got)
	}
	if !strings.Contains(got, "&lt;/details&gt;") {
		t.Errorf("entity missing: %q", got)
	}
	if !strings.Contains(got, "foo ") || !strings.Contains(got, " bar") {
		t.Errorf("surrounding content lost: %q", got)
	}
}

func TestDetailsBodyWriter_PartialPrefixThenUnrelated(t *testing.T) {
	// </de で止まって全く別の文字が来た場合は元のまま出す
	var buf bytes.Buffer
	bw := newDetailsBodyWriter(&buf)
	bw.Write([]byte("foo </de"))
	bw.Write([]byte("scribe me"))
	bw.Flush()
	got := buf.String()
	if got != "foo </describe me" {
		t.Errorf("unexpected output: %q", got)
	}
}

func TestDetailsBodyWriter_PassesThroughOtherTags(t *testing.T) {
	var buf bytes.Buffer
	bw := newDetailsBodyWriter(&buf)
	bw.Write([]byte("<div>x</div> <details>y</details>"))
	bw.Flush()
	got := buf.String()
	if strings.Contains(got, "</details>") {
		t.Errorf("close tag should be escaped: %q", got)
	}
	if !strings.Contains(got, "<div>") {
		t.Errorf("unrelated tag was modified: %q", got)
	}
}

func TestRender_EscapesCloseDetailsInThinking(t *testing.T) {
	c := loadFixture(t)
	// thinking ブロックの内容に </details> を仕込む
	c.ChatMessages[1].Content[0].Thinking = "explaining </details> tag here"
	got := renderToString(t, c, Options{IncludeThinking: true})
	// thinking が details 内にあるので、内部の </details> がエンティティ化されている
	if strings.Count(got, "</details>") > strings.Count(got, "<details>") {
		t.Errorf("close tag in body broke nesting: counts mismatch in:\n%s", got)
	}
	if !strings.Contains(got, "&lt;/details&gt;") {
		t.Errorf("close tag in thinking was not escaped:\n%s", got)
	}
}
