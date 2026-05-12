package render

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/yoshihirosuzuki/claude2md/internal/export"
)

// Options はレンダラの挙動切り替え。
type Options struct {
	IncludeThinking bool
	IncludeTools    bool
}

// Render は 1 会話を Markdown として w に書き出す。
// 中間で 1 会話分の文字列を生成しないため、最大会話 1MB クラスのコピーが発生しない。
func Render(w io.Writer, c *export.Conversation, opt Options) error {
	ew := newErrWriter(w)
	fm := buildFrontmatter(c)
	if err := fm.Write(ew); err != nil {
		return err
	}
	io.WriteString(ew, "\n# ")
	io.WriteString(ew, displayName(c.Name))
	io.WriteString(ew, "\n")

	if s := strings.TrimSpace(c.Summary); s != "" {
		io.WriteString(ew, "\n## Summary\n\n")
		io.WriteString(ew, s)
		io.WriteString(ew, "\n")
	}

	toolNames := buildToolNameIndex(c)
	for _, m := range c.ChatMessages {
		writeMessage(ew, m, opt, toolNames)
	}
	return ew.err
}

func toolUseID(blk export.ContentBlock) string {
	if blk.Type == "tool_use" {
		return blk.ID
	}
	if blk.Type == "tool_result" {
		return blk.ToolUseID
	}
	return ""
}

// buildToolNameIndex は会話内の tool_use ブロックを走査し、id → name の対応を作る。
// tool_result.name が空のときに対応する tool_use の name を引くためのインデックス。
func buildToolNameIndex(c *export.Conversation) map[string]string {
	idx := make(map[string]string)
	for _, m := range c.ChatMessages {
		for _, blk := range m.Content {
			if blk.Type != "tool_use" {
				continue
			}
			id := toolUseID(blk)
			if id == "" || blk.Name == "" {
				continue
			}
			idx[id] = blk.Name
		}
	}
	return idx
}

func buildFrontmatter(c *export.Conversation) Frontmatter {
	var atts, files []string
	for _, m := range c.ChatMessages {
		for _, a := range m.Attachments {
			if a.FileName != "" {
				atts = append(atts, a.FileName)
			}
		}
		for _, f := range m.Files {
			if f.FileName != "" {
				files = append(files, f.FileName)
			}
		}
	}
	return Frontmatter{
		UUID:         c.UUID,
		Name:         c.Name,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
		MessageCount: len(c.ChatMessages),
		Attachments:  atts,
		Files:        files,
	}
}

func displayName(name string) string {
	if strings.TrimSpace(name) == "" {
		return "Untitled"
	}
	return name
}

func writeMessage(w io.Writer, m export.Message, opt Options, toolNames map[string]string) {
	fmt.Fprintf(w, "\n## %s\n", senderHeading(m.Sender))

	wrote := false
	for _, blk := range m.Content {
		if writeBlock(w, blk, opt, toolNames) {
			wrote = true
		}
	}
	if !wrote && strings.TrimSpace(m.Text) != "" {
		// content[] が空で text フィールドだけにあるレガシー形式の保険
		io.WriteString(w, "\n")
		io.WriteString(w, m.Text)
		io.WriteString(w, "\n")
	}

	for _, a := range m.Attachments {
		if strings.TrimSpace(a.ExtractedContent) == "" {
			continue
		}
		io.WriteString(w, "\n")
		writeFoldedDetails(w,
			fmt.Sprintf("Attachment: %s", escapeSummary(a.FileName)),
			func(body io.Writer) { io.WriteString(body, strings.TrimRight(a.ExtractedContent, "\n")) },
		)
		io.WriteString(w, "\n")
	}
}

func senderHeading(sender string) string {
	switch sender {
	case "human":
		return "You"
	case "assistant":
		return "Claude"
	case "":
		return "Message"
	default:
		return sender
	}
}

// writeBlock は 1 ブロックを w に書き出し、何かを書いた場合 true を返す。
// 全ブロックの中間文字列を 1 度に作らず、ブロック単位で w にストリームする。
// toolNames は tool_result.name が空のときに対応する tool_use.name を引くためのインデックス。
func writeBlock(w io.Writer, blk export.ContentBlock, opt Options, toolNames map[string]string) bool {
	switch blk.Type {
	case "text":
		body := strings.TrimRight(blk.Text, "\n")
		if body == "" {
			return false
		}
		io.WriteString(w, "\n")
		io.WriteString(w, body)
		io.WriteString(w, "\n")
		return true
	case "thinking":
		if !opt.IncludeThinking {
			return false
		}
		io.WriteString(w, "\n")
		writeFoldedDetails(w, "Thinking", func(body io.Writer) {
			io.WriteString(body, strings.TrimRight(blk.Thinking, "\n"))
		})
		io.WriteString(w, "\n")
		return true
	case "tool_use":
		if !opt.IncludeTools {
			return false
		}
		io.WriteString(w, "\n")
		writeFoldedDetails(w,
			fmt.Sprintf("Tool: %s", escapeSummary(blk.Name)),
			func(body io.Writer) { writeFencedJSON(body, blk.Input) },
		)
		io.WriteString(w, "\n")
		return true
	case "tool_result":
		if !opt.IncludeTools {
			return false
		}
		name := blk.Name
		if name == "" {
			if linked, ok := toolNames[blk.ToolUseID]; ok {
				name = linked
			}
		}
		io.WriteString(w, "\n")
		writeFoldedDetails(w,
			fmt.Sprintf("Tool result: %s", escapeSummary(name)),
			func(body io.Writer) { writeToolResultText(body, blk) },
		)
		io.WriteString(w, "\n")
		return true
	}
	return false
}

func writeFoldedDetails(w io.Writer, summary string, body func(io.Writer)) {
	fmt.Fprintf(w, "<details><summary>%s</summary>\n\n", summary)
	bw := newDetailsBodyWriter(w)
	body(bw)
	_ = bw.Flush()
	io.WriteString(w, "\n\n</details>")
}

// detailsBodyWriter は <details> ブロックの body に書き込まれるバイト列に対して、
// `</details>` （大小無視）を `&lt;/details&gt;` にエスケープして外側の閉じタグが破壊されないようにする。
// チャンク境界に閉じタグが跨る場合にも対応するため、未確定接尾辞を内部で保留する。
type detailsBodyWriter struct {
	w       io.Writer
	pending []byte // </details... まで書きかけの未確定バイト列
}

func newDetailsBodyWriter(w io.Writer) *detailsBodyWriter {
	return &detailsBodyWriter{w: w}
}

// closeTag は照合対象（小文字）。
var closeTag = []byte("</details>")

func (d *detailsBodyWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	// pending と p を独立した buf に連結し、別途 out バッファを用意して
	// underlying array の共有による上書きを防ぐ。
	buf := make([]byte, 0, len(d.pending)+len(p))
	buf = append(buf, d.pending...)
	buf = append(buf, p...)
	d.pending = nil

	out := make([]byte, 0, len(buf))
	i := 0
	for i < len(buf) {
		if buf[i] != '<' {
			out = append(out, buf[i])
			i++
			continue
		}
		rem := len(buf) - i
		matchLen := rem
		if matchLen > len(closeTag) {
			matchLen = len(closeTag)
		}
		if !hasCloseTagPrefix(buf[i : i+matchLen]) {
			out = append(out, buf[i])
			i++
			continue
		}
		if matchLen == len(closeTag) {
			out = append(out, []byte("&lt;/details&gt;")...)
			i += len(closeTag)
			continue
		}
		// prefix 一致だが続きがまだ来ていない → 以降は pending に保留
		d.pending = append(d.pending, buf[i:]...)
		break
	}

	if len(out) > 0 {
		if _, err := d.w.Write(out); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

// Flush は pending を書き出す。レンダリング末尾で必ず呼ぶ。
func (d *detailsBodyWriter) Flush() error {
	if len(d.pending) == 0 {
		return nil
	}
	_, err := d.w.Write(d.pending)
	d.pending = nil
	return err
}

// hasCloseTagPrefix は b が closeTag の prefix と (大小無視で) 一致するかを返す。
func hasCloseTagPrefix(b []byte) bool {
	if len(b) > len(closeTag) {
		return false
	}
	for i, c := range b {
		want := closeTag[i]
		got := c
		if got >= 'A' && got <= 'Z' {
			got += 32
		}
		if got != want {
			return false
		}
	}
	return true
}

// escapeSummary は <summary> 要素内に書き出すテキストをエスケープする。
// 置換順は `&` → `<` → `>` の順に登録するが、strings.NewReplacer は
// 1-pass で全パターンを同時に走査するため二重エスケープは発生しない
// （登録順は走査の優先度であり、結果の入れ子では無い）。
// summary は HTML 要素のテキストコンテンツであり属性値ではないため、
// 構造を破壊しうる `<` `>` `&` の 3 文字のみエスケープすれば十分。
func escapeSummary(s string) string {
	if s == "" {
		return "(unnamed)"
	}
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

func writeFencedJSON(w io.Writer, raw json.RawMessage) {
	if len(raw) == 0 {
		io.WriteString(w, "```\n```")
		return
	}
	var pretty any
	if err := json.Unmarshal(raw, &pretty); err == nil {
		if out, err := json.MarshalIndent(pretty, "", "  "); err == nil {
			io.WriteString(w, "```json\n")
			w.Write(out)
			io.WriteString(w, "\n```")
			return
		}
	}
	io.WriteString(w, "```\n")
	w.Write(raw)
	io.WriteString(w, "\n```")
}

// writeToolResultText は tool_result の content を Markdown として w に書き出す。
// content は配列 [{"type":"text","text":"..."}] か、文字列、または任意 JSON。
// 中間文字列を作らず、各分岐で直接 w に書き込むことでブロック単位のコピーを排除する。
func writeToolResultText(w io.Writer, blk export.ContentBlock) {
	if len(blk.ResultContent) > 0 {
		var arr []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(blk.ResultContent, &arr); err == nil && len(arr) > 0 {
			wrote := 0
			for _, e := range arr {
				if e.Text == "" {
					continue
				}
				if wrote > 0 {
					io.WriteString(w, "\n\n")
				}
				io.WriteString(w, e.Text)
				wrote++
			}
			if wrote > 0 {
				return
			}
		}
		var s string
		if err := json.Unmarshal(blk.ResultContent, &s); err == nil && s != "" {
			io.WriteString(w, s)
			return
		}
		io.WriteString(w, "```json\n")
		w.Write(blk.ResultContent)
		io.WriteString(w, "\n```")
		return
	}
	if len(blk.DisplayContent) > 0 {
		io.WriteString(w, "```json\n")
		w.Write(blk.DisplayContent)
		io.WriteString(w, "\n```")
		return
	}
	io.WriteString(w, "(empty result)")
}
