package index

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()
	f, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if f.Version != currentVersion {
		t.Errorf("version = %d", f.Version)
	}
	if len(f.Entries) != 0 {
		t.Errorf("expected empty entries; got %v", f.Entries)
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	f := &File{
		Entries: map[string]Entry{
			"u1": {Path: "2026-04/2026-04-07_x.md", UpdatedAt: "2026-04-07T05:53:28.000000Z"},
		},
	}
	if err := f.Save(dir); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Entries["u1"].Path != f.Entries["u1"].Path {
		t.Errorf("path mismatch")
	}
}

func TestDecide_NewEntry_IsCreated(t *testing.T) {
	f := &File{Entries: map[string]Entry{}}
	r, err := f.Decide("u1", "2026-01-01T00:00:00Z", "a/b.md", false)
	if err != nil {
		t.Fatal(err)
	}
	if r.Decision != DecisionCreated {
		t.Errorf("got %v", r.Decision)
	}
	if r.OldRelPath != "" {
		t.Errorf("OldRelPath should be empty; got %q", r.OldRelPath)
	}
}

func TestDecide_SameUpdatedAt_IsSkipped(t *testing.T) {
	f := &File{Entries: map[string]Entry{
		"u1": {Path: "a/b.md", UpdatedAt: "2026-01-01T00:00:00Z"},
	}}
	r, err := f.Decide("u1", "2026-01-01T00:00:00Z", "a/b.md", false)
	if err != nil {
		t.Fatal(err)
	}
	if r.Decision != DecisionSkipped {
		t.Errorf("got %v", r.Decision)
	}
}

func TestDecide_NewerInput_IsUpdated(t *testing.T) {
	f := &File{Entries: map[string]Entry{
		"u1": {Path: "a/b.md", UpdatedAt: "2026-01-01T00:00:00Z"},
	}}
	r, err := f.Decide("u1", "2026-02-01T00:00:00Z", "a/b.md", false)
	if err != nil {
		t.Fatal(err)
	}
	if r.Decision != DecisionUpdated {
		t.Errorf("got %v", r.Decision)
	}
	if r.OldRelPath != "" {
		t.Errorf("path same → OldRelPath should be empty")
	}
}

func TestDecide_PathChanged_ReturnsOldPath(t *testing.T) {
	f := &File{Entries: map[string]Entry{
		"u1": {Path: "old/old.md", UpdatedAt: "2026-01-01T00:00:00Z"},
	}}
	r, err := f.Decide("u1", "2026-02-01T00:00:00Z", "new/new.md", false)
	if err != nil {
		t.Fatal(err)
	}
	if r.Decision != DecisionUpdated {
		t.Errorf("decision %v", r.Decision)
	}
	if r.OldRelPath != "old/old.md" {
		t.Errorf("OldRelPath = %q", r.OldRelPath)
	}
}

func TestDecide_OlderInput_IsWarnOlder(t *testing.T) {
	f := &File{Entries: map[string]Entry{
		"u1": {Path: "a/b.md", UpdatedAt: "2026-02-01T00:00:00Z"},
	}}
	r, err := f.Decide("u1", "2026-01-01T00:00:00Z", "a/b.md", false)
	if err != nil {
		t.Fatal(err)
	}
	if r.Decision != DecisionWarnOlder {
		t.Errorf("got %v", r.Decision)
	}
}

func TestDecide_Force_UpdatesEvenWhenSame(t *testing.T) {
	f := &File{Entries: map[string]Entry{
		"u1": {Path: "a/b.md", UpdatedAt: "2026-01-01T00:00:00Z"},
	}}
	r, err := f.Decide("u1", "2026-01-01T00:00:00Z", "a/b.md", true)
	if err != nil {
		t.Fatal(err)
	}
	if r.Decision != DecisionUpdated {
		t.Errorf("force should make it Updated; got %v", r.Decision)
	}
}

func TestApply_SkippedDoesNotMutate(t *testing.T) {
	f := &File{Entries: map[string]Entry{
		"u1": {Path: "a/b.md", UpdatedAt: "2026-01-01T00:00:00Z"},
	}}
	f.Apply("u1", Result{Decision: DecisionSkipped, NewRelPath: "should/not/change.md", NewUpdatedAt: "2099-01-01T00:00:00Z"})
	if f.Entries["u1"].Path != "a/b.md" {
		t.Errorf("entry mutated")
	}
}

func TestLoad_BrokenJSON_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Errorf("expected error for malformed index.json")
	}
}

func TestSave_AtomicAndOverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	(&File{Entries: map[string]Entry{"u1": {Path: "1", UpdatedAt: "2026-01-01T00:00:00Z"}}}).Save(dir)
	(&File{Entries: map[string]Entry{"u2": {Path: "2", UpdatedAt: "2026-01-01T00:00:00Z"}}}).Save(dir)

	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Entries["u2"]; !ok {
		t.Errorf("u2 missing after second save")
	}
	if _, ok := got.Entries["u1"]; ok {
		t.Errorf("u1 should be overwritten")
	}

	// 一時ファイルが残っていないこと
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("temp file leaked: %s", e.Name())
		}
	}
}
