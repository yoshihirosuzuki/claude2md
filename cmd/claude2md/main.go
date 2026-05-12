package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/yoshihirosuzuki/claude2md/internal/export"
	"github.com/yoshihirosuzuki/claude2md/internal/index"
	"github.com/yoshihirosuzuki/claude2md/internal/render"
	"github.com/yoshihirosuzuki/claude2md/internal/timestamp"
	"golang.org/x/term"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "claude2md: %v\n", err)
		os.Exit(1)
	}
}

type stats struct {
	created, updated, skipped, warnOlder, errs int
}

func run() error {
	out := flag.String("o", "out", "出力ディレクトリ")
	includeThinking := flag.Bool("include-thinking", false, "thinking ブロックを <details> 折りたたみで含める")
	includeTools := flag.Bool("include-tools", false, "tool_use / tool_result ブロックを <details> 折りたたみで含める")
	force := flag.Bool("force", false, "差分判定を無視して全件再生成")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: claude2md <export.zip> [options]\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
		return fmt.Errorf("引数 <export.zip> が必要です")
	}
	zipPath := flag.Arg(0)
	if strings.HasPrefix(zipPath, "-") {
		return fmt.Errorf("ZIP パスがフラグに見えます: %q（オプションを ZIP の前に置いてください）", zipPath)
	}
	// `flag` は最初の非フラグで解釈を止めるため、ZIP の後ろに置かれた `--force` 等も拾えるよう再 parse する
	if rest := flag.Args()[1:]; len(rest) > 0 {
		if err := flag.CommandLine.Parse(rest); err != nil {
			return err
		}
		if flag.NArg() > 0 {
			return fmt.Errorf("予期しない位置引数: %v", flag.Args())
		}
	}

	if strings.TrimSpace(*out) == "" {
		return fmt.Errorf("-o の値が空です")
	}
	outDir, err := resolveOutDir(*out)
	if err != nil {
		return err
	}

	idx, err := index.Load(outDir)
	if err != nil {
		return fmt.Errorf("load index: %w", err)
	}

	opt := render.Options{
		IncludeThinking: *includeThinking,
		IncludeTools:    *includeTools,
	}

	// progressbar を conversations.json の読み込みバイト数（uncompressed）で進める。
	// 会話単位ではなくバイト単位にすることで、ZIP を 2 回スキャンせずに済み、
	// 会話サイズが不揃いでも進捗が直線的に進む。
	// 色は stderr が TTY のときに限る。NO_COLOR 仕様（https://no-color.org/）で
	// 環境変数が非空のときは色を抑制する。
	useColor := term.IsTerminal(int(os.Stderr.Fd())) && os.Getenv("NO_COLOR") == ""
	bar := newProgressBar(os.Stderr, useColor)
	maxSet := false

	start := time.Now()
	var st stats
	// 同一ラン内で同じ rel path を使う 2 件目を検出するための一意性チェック用 map。
	// 検出は O(1)、保持コストはエントリ数に比例（167 件で約 7KB）。
	// 永続化が要らない一時的な状態であり、index.json への永続化に代えてラン内のみ生存させる
	// 設計が局所性とメモリ効率の両立として最適。
	usedPaths := map[string]string{}

	convN := 0
	total, walkErr := export.Walk(zipPath,
		func(c *export.Conversation) error {
			processConversation(c, outDir, idx, opt, *force, usedPaths, &st)
			convN++
			bar.Describe(fmt.Sprintf("converting [%d conv]", convN))
			return nil
		},
		func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, "WARN: "+format+"\n", args...)
		},
		func(read, totalBytes int64) {
			if !maxSet {
				bar.ChangeMax64(totalBytes)
				maxSet = true
			}
			_ = bar.Set64(read)
		},
	)
	_ = bar.Finish()
	// progressbar v3.19.0 の Finish() は末尾改行を出さないため、後続の Done. メッセージが
	// バー行に続いて表示されるのを防ぐ。OptionShowElapsedTimeOnFinish と組み合わせると
	// バー行は残るので、ここで明示的に改行して視覚的に分離する。
	fmt.Fprintln(os.Stderr)

	// Walk が中断しても、それまでに反映済みのインデックスは保存しておく
	// （次回実行で書き出し済みのファイルが再 Created にならないようにするため）
	if saveErr := idx.Save(outDir); saveErr != nil {
		fmt.Fprintf(os.Stderr, "WARN: save index: %v\n", saveErr)
	}
	if walkErr != nil {
		return walkErr
	}

	fmt.Printf("Done. created: %d, updated: %d, skipped: %d", st.created, st.updated, st.skipped)
	if st.warnOlder > 0 {
		fmt.Printf(", warn_older: %d", st.warnOlder)
	}
	if st.errs > 0 {
		fmt.Printf(", errors: %d", st.errs)
	}
	fmt.Printf(" (%d bytes scanned) in %s\n", total, time.Since(start).Round(time.Millisecond))
	return nil
}

// newProgressBar は useColor 指定に応じて色付き / プレーンの progressbar を返す。
// 進捗単位はバイト（json.Decoder.InputOffset()）。
// OptionShowCount / OptionShowIts はバイト単位の Set64 と組み合わせると分子にバイト数を出して
// ShowBytes と表示が冗長になるため使わない。会話件数は呼び出し側が Describe で動的に更新する。
func newProgressBar(w io.Writer, useColor bool) *progressbar.ProgressBar {
	opts := []progressbar.Option{
		progressbar.OptionSetDescription("converting"),
		progressbar.OptionSetWriter(w),
		progressbar.OptionShowBytes(true),
		progressbar.OptionThrottle(50 * time.Millisecond),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionFullWidth(),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionSetTheme(buildBarTheme(useColor)),
	}
	if useColor {
		opts = append(opts, progressbar.OptionEnableColorCodes(true))
	}
	return progressbar.NewOptions64(-1, opts...)
}

// buildBarTheme は progressbar の装飾文字を返す。useColor=true のとき Saucer に
// `[cyan]...[reset]` の色マーカーを埋め込み、OptionEnableColorCodes で ANSI 化させる。
func buildBarTheme(useColor bool) progressbar.Theme {
	th := progressbar.Theme{
		Saucer:        "█",
		SaucerHead:    "█",
		SaucerPadding: "░",
		BarStart:      "|",
		BarEnd:        "|",
	}
	if useColor {
		th.Saucer = "[cyan]█[reset]"
		th.SaucerHead = "[cyan]█[reset]"
	}
	return th
}

// resolveOutDir は出力ディレクトリを作成し、絶対 path を返す。
// 権限は標準的な mkdir(2) と同じ 0o755 を mode に渡し、umask を尊重する。
func resolveOutDir(out string) (string, error) {
	if info, err := os.Stat(out); err == nil && !info.IsDir() {
		return "", fmt.Errorf("-o の指す path がディレクトリではありません: %s", out)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return "", fmt.Errorf("create out dir: %w", err)
	}
	abs, err := filepath.Abs(out)
	if err != nil {
		return "", fmt.Errorf("resolve out dir: %w", err)
	}
	return abs, nil
}

func processConversation(c *export.Conversation, outDir string, idx *index.File, opt render.Options, force bool, usedPaths map[string]string, st *stats) {
	relPath, err := buildRelPath(c, usedPaths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARN: skipped %s: %v\n", c.UUID, err)
		st.errs++
		return
	}
	absPath := filepath.Join(outDir, relPath)
	if !isWithin(outDir, absPath) {
		fmt.Fprintf(os.Stderr, "WARN: refusing to write outside out dir for %s -> %s\n", c.UUID, absPath)
		st.errs++
		return
	}

	decision, err := idx.Decide(c.UUID, c.UpdatedAt, relPath, force)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARN: %v\n", err)
		st.errs++
		return
	}

	switch decision.Decision {
	case index.DecisionSkipped:
		st.skipped++
		usedPaths[relPath] = c.UUID
	case index.DecisionWarnOlder:
		fmt.Fprintf(os.Stderr, "WARN: %s: input updated_at older than indexed; keeping existing\n", c.UUID)
		st.warnOlder++
		usedPaths[relPath] = c.UUID
	case index.DecisionCreated, index.DecisionUpdated:
		if err := writeConversation(absPath, c, opt); err != nil {
			fmt.Fprintf(os.Stderr, "WARN: write %s: %v\n", c.UUID, err)
			st.errs++
			return
		}
		if decision.OldRelPath != "" && decision.OldRelPath != relPath {
			oldAbs := filepath.Join(outDir, decision.OldRelPath)
			if isWithin(outDir, oldAbs) {
				_ = os.Remove(oldAbs)
			}
		}
		if decision.Decision == index.DecisionCreated {
			st.created++
		} else {
			st.updated++
		}
		idx.Apply(c.UUID, decision)
		usedPaths[relPath] = c.UUID
	}
}

// isWithin は target が outDir の内側に収まっているかを返す。
// 攻撃者が制御するフィールド（uuid, name 等）から組み立てた相対パスが out dir の外を指していないかの最終防衛線。
func isWithin(outDir, target string) bool {
	out := filepath.Clean(outDir)
	tgt := filepath.Clean(target)
	if tgt == out {
		return false
	}
	prefix := out + string(os.PathSeparator)
	return strings.HasPrefix(tgt, prefix)
}

// buildRelPath は YYYY-MM/YYYY-MM-DD_<slug>.md を返す。
// 同一ラン内の usedPaths と衝突した場合は uuid サフィックスを段階的に伸ばして一意な path を作る。
func buildRelPath(c *export.Conversation, used map[string]string) (string, error) {
	t, err := timestamp.Parse(c.CreatedAt)
	if err != nil {
		return "", fmt.Errorf("created_at: %w", err)
	}
	t = t.UTC()
	yearMonth := t.Format("2006-01")
	day := t.Format("2006-01-02")
	slug := render.Slug(c.Name)
	rel := filepath.Join(yearMonth, fmt.Sprintf("%s_%s.md", day, slug))
	if existing, ok := used[rel]; !ok || existing == c.UUID {
		return rel, nil
	}
	// uuid サフィックス長を 8, 12, 16, ... と段階的に伸ばす。
	// uuid が短い場合はそのまま全文を使う（バッファ越えを防ぐ）。
	suffix := suffixSanitize(c.UUID)
	if suffix == "" {
		return "", fmt.Errorf("uuid is empty; cannot resolve path collision for name=%q", c.Name)
	}
	for n := 8; ; n += 4 {
		take := n
		if take > len(suffix) {
			take = len(suffix)
		}
		rel = filepath.Join(yearMonth, fmt.Sprintf("%s_%s-%s.md", day, slug, suffix[:take]))
		if existing, ok := used[rel]; !ok || existing == c.UUID {
			return rel, nil
		}
		if take == len(suffix) {
			return "", fmt.Errorf("could not resolve path collision for uuid=%s", c.UUID)
		}
	}
}

// suffixSanitize は uuid 文字列をパス安全な文字（英数字とハイフン）のみに制限する。
// 通常の uuid は `xxxxxxxx-xxxx-...` だが、入力が攻撃由来で `..`, `/`, `\` を含む可能性に備える。
func suffixSanitize(uuid string) string {
	var b strings.Builder
	for _, r := range uuid {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// writeConversation は会話を Markdown としてファイルに書き出す（atomic write）。
// render.Render は io.Writer 経由で直接 *os.File に書き込むため、
// 1 会話分の中間文字列バッファを発生させない。bufio.Writer で syscall を圧縮する。
func writeConversation(absPath string, c *export.Conversation, opt render.Options) error {
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(absPath)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			os.Remove(tmpPath)
		}
	}()
	bw := bufio.NewWriter(tmp)
	if err := render.Render(bw, c, opt); err != nil {
		tmp.Close()
		return err
	}
	if err := bw.Flush(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// os.CreateTemp は 0o600 で作るので、Rename 前に 0o644 へ変更し、
	// `os.WriteFile(..., 0o644)` 相当の標準的なファイル権限に揃える。
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, absPath); err != nil {
		return err
	}
	cleanup = false
	return nil
}
