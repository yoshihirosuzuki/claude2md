package index

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/yoshihirosuzuki/claude2md/internal/timestamp"
)

const (
	FileName       = ".index.json"
	currentVersion = 1
)

type Entry struct {
	Path      string `json:"path"`
	UpdatedAt string `json:"updated_at"`
}

type File struct {
	Version int              `json:"version"`
	Entries map[string]Entry `json:"entries"`
}

type Decision int

const (
	DecisionUnknown Decision = iota
	DecisionCreated
	DecisionUpdated
	DecisionSkipped
	DecisionWarnOlder
)

func (d Decision) String() string {
	switch d {
	case DecisionCreated:
		return "created"
	case DecisionUpdated:
		return "updated"
	case DecisionSkipped:
		return "skipped"
	case DecisionWarnOlder:
		return "warn_older"
	}
	return "unknown"
}

type Result struct {
	Decision     Decision
	OldRelPath   string
	NewRelPath   string
	NewUpdatedAt string
}

// Load は dir 直下の .index.json を読み込む。存在しなければ空のインデックスを返す。
func Load(dir string) (*File, error) {
	path := filepath.Join(dir, FileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &File{Version: currentVersion, Entries: map[string]Entry{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", FileName, err)
	}
	if f.Entries == nil {
		f.Entries = map[string]Entry{}
	}
	return &f, nil
}

// Save は dir 直下に .index.json を atomic write で書き出す。
// json.NewEncoder.Encode は内部 buffer に encode してから writer へ Write するが、
// json.MarshalIndent と異なり呼び出し元（本関数）に全エントリ分の []byte を返さないため、
// 本関数のスタックに重ねて保持しなくて済む。
func (f *File) Save(dir string) error {
	if f.Version == 0 {
		f.Version = currentVersion
	}
	if f.Entries == nil {
		f.Entries = map[string]Entry{}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".index.json.*.tmp")
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
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(f); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, filepath.Join(dir, FileName)); err != nil {
		return err
	}
	cleanup = false
	return nil
}

// Decide は会話の uuid・updated_at・出力相対 path を受け取り、必要なアクションを返す。
// force=true の場合は常に Updated/Created を返す（updated_at 比較なし）。
func (f *File) Decide(uuid, updatedAt, newRelPath string, force bool) (Result, error) {
	res := Result{NewRelPath: newRelPath, NewUpdatedAt: updatedAt}
	if force {
		if e, ok := f.Entries[uuid]; ok {
			if e.Path != newRelPath {
				res.OldRelPath = e.Path
			}
			res.Decision = DecisionUpdated
		} else {
			res.Decision = DecisionCreated
		}
		return res, nil
	}

	e, ok := f.Entries[uuid]
	if !ok {
		res.Decision = DecisionCreated
		return res, nil
	}

	cmp, err := compareUpdated(updatedAt, e.UpdatedAt)
	if err != nil {
		return res, fmt.Errorf("uuid=%s: %w", uuid, err)
	}
	switch {
	case cmp > 0:
		if e.Path != newRelPath {
			res.OldRelPath = e.Path
		}
		res.Decision = DecisionUpdated
	case cmp == 0:
		res.Decision = DecisionSkipped
	default:
		res.Decision = DecisionWarnOlder
	}
	return res, nil
}

// Apply は decision の結果をインデックスに反映する。
// Skipped/WarnOlder の場合はインデックスを変更しない。
func (f *File) Apply(uuid string, r Result) {
	if r.Decision == DecisionSkipped || r.Decision == DecisionWarnOlder {
		return
	}
	f.Entries[uuid] = Entry{Path: r.NewRelPath, UpdatedAt: r.NewUpdatedAt}
}

func compareUpdated(a, b string) (int, error) {
	return timestamp.Compare(a, b)
}
