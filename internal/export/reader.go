package export

import (
	"archive/zip"
	"encoding/json"
	"fmt"
)

const conversationsFileName = "conversations.json"

// Visitor は 1 会話ごとに呼ばれる。エラーを返すと Walk はそれを伝播する。
type Visitor func(c *Conversation) error

type Warnf func(format string, args ...any)

// ProgressFunc は conversations.json の読み込みバイト数 (uncompressed) と
// 全体バイト数を受け取る。プログレスバー反映に使う。
type ProgressFunc func(read, total int64)

// openConversationsFile は ZIP を開いて conversations.json の zip.File ハンドラを返す。
func openConversationsFile(zipPath string) (*zip.ReadCloser, *zip.File, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open zip: %w", err)
	}
	for _, zf := range r.File {
		if zf.Name == conversationsFileName {
			return r, zf, nil
		}
	}
	r.Close()
	return nil, nil, fmt.Errorf("%s not found in zip", conversationsFileName)
}

// Walk は ZIP 内の conversations.json を配列ストリームとして読み、各会話に対して visitor を呼ぶ。
// 不正な要素はスキップして warnf に流す。progress は読み込み済みバイト数を visitor 呼び出し前に通知する。
// total = uncompressed conversations.json サイズ。progress は nil 可。
//
// json.Decoder.InputOffset() で読み込み位置を取得し、ZIP を 1 回だけ走査する設計。
func Walk(zipPath string, visitor Visitor, warnf Warnf, progress ProgressFunc) (total int64, err error) {
	r, f, err := openConversationsFile(zipPath)
	if err != nil {
		return 0, err
	}
	defer r.Close()
	total = int64(f.UncompressedSize64)

	rc, err := f.Open()
	if err != nil {
		return total, fmt.Errorf("open %s: %w", conversationsFileName, err)
	}
	defer rc.Close()

	dec := json.NewDecoder(rc)
	tok, err := dec.Token()
	if err != nil {
		return total, fmt.Errorf("read token: %w", err)
	}
	if d, ok := tok.(json.Delim); !ok || d != '[' {
		return total, fmt.Errorf("expected array, got %v", tok)
	}

	for dec.More() {
		var c Conversation
		if err := dec.Decode(&c); err != nil {
			if warnf != nil {
				warnf("skipped malformed conversation: %v", err)
			}
			continue
		}
		if progress != nil {
			progress(dec.InputOffset(), total)
		}
		if err := visitor(&c); err != nil {
			return total, err
		}
	}
	if progress != nil {
		progress(total, total)
	}
	return total, nil
}

