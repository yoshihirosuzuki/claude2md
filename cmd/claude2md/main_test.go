package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yoshihirosuzuki/claude2md/internal/export"
	"github.com/yoshihirosuzuki/claude2md/internal/index"
	"github.com/yoshihirosuzuki/claude2md/internal/render"
)

func conv(uuid, name, createdAt string) *export.Conversation {
	return &export.Conversation{
		UUID:      uuid,
		Name:      name,
		CreatedAt: createdAt,
	}
}

func TestBuildRelPath_BasicShape(t *testing.T) {
	c := conv("aaaa1111-bbbb-2222-cccc-3333", "hello", "2026-04-07T05:53:28.348188Z")
	got, err := buildRelPath(c, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("2026-04", "2026-04-07_hello.md")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildRelPath_CollisionAddsUUIDSuffix(t *testing.T) {
	c1 := conv("aaaaaaaa-1111", "same", "2026-04-07T00:00:00Z")
	c2 := conv("bbbbbbbb-2222", "same", "2026-04-07T00:00:00Z")
	used := map[string]string{}
	r1, err := buildRelPath(c1, used)
	if err != nil {
		t.Fatal(err)
	}
	used[r1] = c1.UUID
	r2, err := buildRelPath(c2, used)
	if err != nil {
		t.Fatal(err)
	}
	if r1 == r2 {
		t.Errorf("collision not resolved")
	}
	if !strings.Contains(r2, "bbbbbbbb") {
		t.Errorf("expected uuid suffix; got %q", r2)
	}
}

func TestBuildRelPath_ShortUUID_DoesNotInfiniteLoop(t *testing.T) {
	c1 := conv("abc", "same", "2026-04-07T00:00:00Z")
	c2 := conv("xyz", "same", "2026-04-07T00:00:00Z")
	used := map[string]string{}
	r1, err := buildRelPath(c1, used)
	if err != nil {
		t.Fatal(err)
	}
	used[r1] = c1.UUID
	r2, err := buildRelPath(c2, used)
	if err != nil {
		t.Fatalf("expected fallback to full uuid for short uuid; got err=%v", err)
	}
	if !strings.Contains(r2, "xyz") {
		t.Errorf("expected short uuid in path; got %q", r2)
	}
}

func TestBuildRelPath_RejectsTraversingUUID(t *testing.T) {
	// 細工された uuid に "../" が含まれている場合、suffixSanitize が無害化することを確認
	c1 := conv("safe-1111", "same", "2026-04-07T00:00:00Z")
	c2 := conv("../../../etc/passwd", "same", "2026-04-07T00:00:00Z")
	used := map[string]string{}
	r1, _ := buildRelPath(c1, used)
	used[r1] = c1.UUID
	r2, err := buildRelPath(c2, used)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(r2, "..") || strings.Contains(r2, "/etc/") {
		t.Errorf("path traversal not sanitized: %q", r2)
	}
}

func TestBuildRelPath_NameWithTraversalStaysWithinOutDir(t *testing.T) {
	// name に "/" や ".." が含まれていても、最終的な absPath が outDir の外を指さないこと
	c := conv("aaaaaaaa", "../../etc/passwd", "2026-04-07T00:00:00Z")
	got, err := buildRelPath(c, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(got) != "2026-04" {
		t.Errorf("unexpected directory in path: %q", got)
	}
	outDir := "/tmp/out"
	abs := filepath.Join(outDir, got)
	if !isWithin(outDir, abs) {
		t.Errorf("absPath %q escaped outDir %q", abs, outDir)
	}
}

func TestIsWithin(t *testing.T) {
	cases := []struct {
		out, target string
		want        bool
	}{
		{"/tmp/out", "/tmp/out/2026-04/x.md", true},
		{"/tmp/out", "/tmp/outsider/x.md", false},
		{"/tmp/out", "/tmp/out", false}, // 自分自身に書こうとするケースは却下
		{"/tmp/out", "/tmp/out/../etc/passwd", false},
		{"/tmp/out", "/etc/passwd", false},
	}
	for _, tt := range cases {
		got := isWithin(tt.out, tt.target)
		if got != tt.want {
			t.Errorf("isWithin(%q, %q) = %v, want %v", tt.out, tt.target, got, tt.want)
		}
	}
}

func TestProcessConversation_WriteFailure_IncrementsErrs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file mode based test")
	}
	// 書き込み不可ディレクトリ (chmod 0o500) の中に出力させ、writeConversation が失敗することを確認。
	out := t.TempDir()
	if err := os.Chmod(out, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(out, 0o700) })

	c := &export.Conversation{
		UUID:      "u1",
		Name:      "x",
		CreatedAt: "2026-04-07T00:00:00Z",
		UpdatedAt: "2026-04-07T00:00:00Z",
	}
	idx := &index.File{Entries: map[string]index.Entry{}}
	used := map[string]string{}
	var st stats
	processConversation(c, out, idx, render.Options{}, false, used, &st)

	if st.errs != 1 {
		t.Errorf("expected errs=1, got %d", st.errs)
	}
	if st.created != 0 {
		t.Errorf("expected created=0, got %d", st.created)
	}
	// Walk 継続性: 別の正常な会話を書き込めない環境でも panic せず errs に計上されること
	if _, ok := idx.Entries["u1"]; ok {
		t.Errorf("failed write should not be applied to index")
	}
}

func TestProcessConversation_RefusesPathOutsideOutDir(t *testing.T) {
	out := t.TempDir()
	c := &export.Conversation{
		UUID:      "../../etc/evil",
		Name:      "x",
		CreatedAt: "2026-04-07T00:00:00Z",
		UpdatedAt: "2026-04-07T00:00:00Z",
	}
	// 同名衝突を強制して uuid サフィックスを取らせる
	used := map[string]string{
		filepath.Join("2026-04", "2026-04-07_x.md"): "different-uuid",
	}
	idx := &index.File{Entries: map[string]index.Entry{}}
	var st stats
	processConversation(c, out, idx, render.Options{}, false, used, &st)

	// suffixSanitize が `..` などを除去するため、最終 path は out 内に収まる。
	// 書き込みは成功するはず（traversal は防がれるが書き込み自体は許可）。
	// errs が増えないことを確認。
	if st.errs != 0 {
		t.Errorf("expected errs=0, got %d (sanitization should keep path inside out)", st.errs)
	}
}

func TestBuildBarTheme_Color(t *testing.T) {
	th := buildBarTheme(true)
	if !strings.Contains(th.Saucer, "[cyan]") || !strings.Contains(th.Saucer, "[reset]") {
		t.Errorf("expected color markers in Saucer, got %q", th.Saucer)
	}
	if th.SaucerPadding != "░" {
		t.Errorf("expected dense padding ░, got %q", th.SaucerPadding)
	}
}

func TestBuildBarTheme_Plain(t *testing.T) {
	th := buildBarTheme(false)
	if strings.Contains(th.Saucer, "cyan") {
		t.Errorf("expected no color marker, got %q", th.Saucer)
	}
	if th.Saucer != "█" {
		t.Errorf("expected block char █, got %q", th.Saucer)
	}
	if th.SaucerPadding != "░" {
		t.Errorf("expected dense padding ░, got %q", th.SaucerPadding)
	}
}

// 非カラー bar の出力に ANSI エスケープが混じっていないことを確認する。
// リダイレクト・パイプ時の事故防止のための回帰テスト。
// throttle(50ms) で Set(50) の描画がスキップされても、Finish() は強制描画するため
// バーの最終形は必ず buf に書かれる。
func TestNewProgressBar_PlainHasNoANSI(t *testing.T) {
	var buf bytes.Buffer
	bar := newProgressBar(&buf, false)
	bar.ChangeMax64(100)
	_ = bar.Set(50)
	_ = bar.Finish()
	if strings.ContainsRune(buf.String(), '\x1b') {
		t.Errorf("plain bar must not emit ANSI escape, got bytes: %q", buf.String())
	}
}

// 色付き bar の出力に ANSI エスケープが含まれることを確認する。
// OptionEnableColorCodes と Theme の `[cyan]...[reset]` が実際に ANSI 化されることの回帰防止。
func TestNewProgressBar_ColorEmitsANSI(t *testing.T) {
	var buf bytes.Buffer
	bar := newProgressBar(&buf, true)
	bar.ChangeMax64(100)
	_ = bar.Set(50)
	_ = bar.Finish()
	if !strings.ContainsRune(buf.String(), '\x1b') {
		t.Errorf("color bar must emit ANSI escape, got bytes: %q", buf.String())
	}
}

func TestSuffixSanitize(t *testing.T) {
	cases := map[string]string{
		"abc-123":    "abc-123",
		"../../etc":  "etc",
		"a/b\\c":     "abc",
		"":           "",
		"safe_uuid":  "safeuuid", // underscore は除去
		"!@#$%^&*()": "",
	}
	for in, want := range cases {
		got := suffixSanitize(in)
		if got != want {
			t.Errorf("suffixSanitize(%q) = %q, want %q", in, got, want)
		}
	}
}
