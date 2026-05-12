package export

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func writeFixtureZip(t *testing.T, conversationsJSON string, extraFiles map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.zip")

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	w, err := zw.Create("conversations.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(conversationsJSON)); err != nil {
		t.Fatal(err)
	}
	for name, content := range extraFiles {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestWalk_Happy(t *testing.T) {
	src := `[
		{"uuid":"u1","name":"first","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","chat_messages":[]},
		{"uuid":"u2","name":"second","created_at":"2026-02-01T00:00:00Z","updated_at":"2026-02-01T00:00:00Z","chat_messages":[]},
		{"uuid":"u3","name":"third","created_at":"2026-03-01T00:00:00Z","updated_at":"2026-03-01T00:00:00Z","chat_messages":[]}
	]`
	zipPath := writeFixtureZip(t, src, nil)

	var got []string
	total, err := Walk(zipPath, func(c *Conversation) error {
		got = append(got, c.UUID)
		return nil
	}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if total != int64(len(src)) {
		t.Errorf("total = %d, want %d", total, len(src))
	}
	if len(got) != 3 || got[0] != "u1" || got[2] != "u3" {
		t.Errorf("got %v", got)
	}
}

func TestWalk_ProgressCallback(t *testing.T) {
	src := `[{"uuid":"u1","chat_messages":[]},{"uuid":"u2","chat_messages":[]}]`
	zipPath := writeFixtureZip(t, src, nil)

	var reads []int64
	var lastTotal int64
	_, err := Walk(zipPath, func(c *Conversation) error { return nil }, nil, func(read, total int64) {
		reads = append(reads, read)
		lastTotal = total
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(reads) < 2 {
		t.Errorf("expected at least 2 progress callbacks; got %d", len(reads))
	}
	for i := 1; i < len(reads); i++ {
		if reads[i] < reads[i-1] {
			t.Errorf("progress is not monotonic: %v", reads)
		}
	}
	if reads[len(reads)-1] != lastTotal {
		t.Errorf("final progress %d != total %d", reads[len(reads)-1], lastTotal)
	}
}

func TestWalk_SkipsMalformed(t *testing.T) {
	src := `[
		{"uuid":"u1","chat_messages":"oops"},
		{"uuid":"u2","name":"ok","chat_messages":[]}
	]`
	zipPath := writeFixtureZip(t, src, nil)

	var warnings []string
	var got []string
	_, err := Walk(zipPath, func(c *Conversation) error {
		got = append(got, c.UUID)
		return nil
	}, func(f string, args ...any) {
		warnings = append(warnings, f)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "u2" {
		t.Errorf("visited %v, want [u2]", got)
	}
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warnings))
	}
}

func TestWalk_NoConversationsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.zip")
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	w, _ := zw.Create("users.json")
	w.Write([]byte("[]"))
	zw.Close()
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Walk(path, func(*Conversation) error { return nil }, nil, nil); err == nil {
		t.Errorf("expected error when conversations.json missing")
	}
}

func TestWalk_IgnoresUnrelatedFiles(t *testing.T) {
	src := `[{"uuid":"only","name":"x","chat_messages":[]}]`
	extras := map[string]string{
		"users.json":              `[]`,
		"memories.json":           `[]`,
		"projects/abc.json":       `{}`,
		"some/random/file.binary": "garbage",
	}
	zipPath := writeFixtureZip(t, src, extras)

	count := 0
	_, err := Walk(zipPath, func(c *Conversation) error {
		count++
		return nil
	}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("count=%d", count)
	}
}
