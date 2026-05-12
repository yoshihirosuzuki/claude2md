package render

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Frontmatter は会話 MD ファイル先頭の YAML frontmatter。
type Frontmatter struct {
	UUID         string
	Name         string
	CreatedAt    string
	UpdatedAt    string
	MessageCount int
	Attachments  []string
	Files        []string
}

// Write は YAML frontmatter を `---` 区切りで w に書き出す。末尾改行付き。
// w に書き込み中にエラーが起きた時点で以降の出力をスキップし、最終的なエラーを返す。
func (f Frontmatter) Write(w io.Writer) error {
	ew := newErrWriter(w)
	io.WriteString(ew, "---\n")
	writeScalar(ew, "uuid", f.UUID)
	writeScalar(ew, "name", f.Name)
	writeScalar(ew, "created_at", f.CreatedAt)
	writeScalar(ew, "updated_at", f.UpdatedAt)
	fmt.Fprintf(ew, "message_count: %d\n", f.MessageCount)
	writeList(ew, "attachments", f.Attachments)
	writeList(ew, "files", f.Files)
	io.WriteString(ew, "---\n")
	return ew.err
}

func writeScalar(w io.Writer, key, value string) {
	fmt.Fprintf(w, "%s: %s\n", key, yamlQuote(value))
}

func writeList(w io.Writer, key string, items []string) {
	uniq := dedupeSorted(items)
	if len(uniq) == 0 {
		return
	}
	parts := make([]string, len(uniq))
	for i, s := range uniq {
		parts[i] = yamlQuote(s)
	}
	fmt.Fprintf(w, "%s: [%s]\n", key, strings.Join(parts, ", "))
}

func dedupeSorted(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// yamlQuote は YAML スカラーとして安全にクォートが必要かを判定し、必要ならダブルクォートで囲む。
// 空文字列、フロー指示子、コロン+空白、先頭末尾の空白、改行、特殊リテラル（true/false/null/number 等）を対象。
func yamlQuote(s string) string {
	if s == "" {
		return `""`
	}
	if needsQuote(s) {
		var b strings.Builder
		b.WriteByte('"')
		for _, r := range s {
			switch r {
			case '\\':
				b.WriteString(`\\`)
			case '"':
				b.WriteString(`\"`)
			case '\n':
				b.WriteString(`\n`)
			case '\r':
				b.WriteString(`\r`)
			case '\t':
				b.WriteString(`\t`)
			case '\x00':
				b.WriteString(`\0`)
			default:
				b.WriteRune(r)
			}
		}
		b.WriteByte('"')
		return b.String()
	}
	return s
}

func needsQuote(s string) bool {
	if s != strings.TrimSpace(s) {
		return true
	}
	switch strings.ToLower(s) {
	case "true", "false", "yes", "no", "null", "~":
		return true
	}
	for i, r := range s {
		switch r {
		case '#', '&', '*', '!', '|', '>', '\'', '"', '%', '@', '`', '\n', '\r', '\t':
			return true
		case '{', '}', '[', ']', ',':
			return true
		case ':':
			next := byte(0)
			if i+1 < len(s) {
				next = s[i+1]
			}
			if next == ' ' || next == 0 {
				return true
			}
		case '-', '?':
			if i == 0 {
				next := byte(0)
				if i+1 < len(s) {
					next = s[i+1]
				}
				if next == ' ' || next == 0 {
					return true
				}
			}
		}
		if r < 0x20 {
			return true
		}
	}
	return false
}
