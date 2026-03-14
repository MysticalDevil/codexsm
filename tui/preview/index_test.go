package preview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpsertAndLoadIndexEntry(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	indexPath := filepath.Join(dir, "preview-index.jsonl")

	rec := IndexRecord{
		Key:           "k1",
		Path:          "/tmp/s1.jsonl",
		Width:         80,
		SizeBytes:     128,
		UpdatedAtUnix: 100,
		TouchedAtUnix: 101,
		Lines:         []string{"line1", "line2"},
	}
	if err := UpsertIndex(indexPath, 32, rec); err != nil {
		t.Fatalf("UpsertIndex: %v", err)
	}

	lines, ok, err := LoadIndexEntry(indexPath, rec.Key)
	if err != nil {
		t.Fatalf("LoadIndexEntry: %v", err)
	}
	if !ok {
		t.Fatalf("LoadIndexEntry(%q): expected hit", rec.Key)
	}
	if strings.Join(lines, "\n") != strings.Join(rec.Lines, "\n") {
		t.Fatalf("LoadIndexEntry lines mismatch: got=%q want=%q", lines, rec.Lines)
	}
}

func TestReadIndexSkipsCorruptedLinesAndRewrites(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	indexPath := filepath.Join(dir, "preview-index.jsonl")
	content := strings.Join([]string{
		`{"key":"ok1","path":"p1","touched_at_unix":10,"lines":["a"]}`,
		`{"key":"","path":"bad","touched_at_unix":11,"lines":["bad"]}`,
		`{not-json`,
		`{"key":"ok2","path":"p2","touched_at_unix":12,"lines":["b"]}`,
	}, "\n") + "\n"
	if err := os.WriteFile(indexPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	entries, corrupted, err := ReadIndex(indexPath)
	if err != nil {
		t.Fatalf("ReadIndex: %v", err)
	}
	if !corrupted {
		t.Fatalf("ReadIndex: expected corrupted=true")
	}
	if len(entries) != 2 {
		t.Fatalf("ReadIndex: got %d entries, want 2", len(entries))
	}

	lines, ok, err := LoadIndexEntry(indexPath, "ok1")
	if err != nil {
		t.Fatalf("LoadIndexEntry after rewrite: %v", err)
	}
	if !ok || len(lines) != 1 || lines[0] != "a" {
		t.Fatalf("LoadIndexEntry after rewrite mismatch: ok=%v lines=%q", ok, lines)
	}

	after, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("ReadFile after rewrite: %v", err)
	}
	if strings.Contains(string(after), "{not-json") {
		t.Fatalf("expected corrupted line removed after rewrite, got: %q", string(after))
	}
}

func TestRewriteIndexRespectsCapAndByteBudget(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	indexPath := filepath.Join(dir, "preview-index.jsonl")
	entries := map[string]IndexRecord{
		"k-newest": {
			Key:           "k-newest",
			Path:          "p1",
			TouchedAtUnix: 30,
			Lines:         []string{"newest", strings.Repeat("n", 128)},
		},
		"k-mid": {
			Key:           "k-mid",
			Path:          "p2",
			TouchedAtUnix: 20,
			Lines:         []string{"middle", strings.Repeat("m", 128)},
		},
		"k-oldest": {
			Key:           "k-oldest",
			Path:          "p3",
			TouchedAtUnix: 10,
			Lines:         []string{"oldest", strings.Repeat("o", 128)},
		},
	}

	if err := RewriteIndex(indexPath, entries, 2, 1<<20); err != nil {
		t.Fatalf("RewriteIndex cap phase: %v", err)
	}
	loaded, _, err := ReadIndex(indexPath)
	if err != nil {
		t.Fatalf("ReadIndex cap phase: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("cap phase expected 2 entries, got %d", len(loaded))
	}
	if _, ok := loaded["k-newest"]; !ok {
		t.Fatalf("cap phase expected newest entry preserved")
	}
	if _, ok := loaded["k-mid"]; !ok {
		t.Fatalf("cap phase expected middle entry preserved")
	}

	if err := RewriteIndex(indexPath, entries, 10, 1); err != nil {
		t.Fatalf("RewriteIndex byte-budget phase: %v", err)
	}
	loaded, _, err = ReadIndex(indexPath)
	if err != nil {
		t.Fatalf("ReadIndex byte-budget phase: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("byte-budget phase expected exactly 1 entry, got %d", len(loaded))
	}
	if _, ok := loaded["k-newest"]; !ok {
		t.Fatalf("byte-budget phase expected newest entry kept")
	}
}
