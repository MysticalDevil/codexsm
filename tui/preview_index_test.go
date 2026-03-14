package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/tui/preview"
)

func TestLoadPreviewIndexEntry_RecoversCorruptedLines(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "preview.v1.jsonl")
	content := strings.Join([]string{
		`{"key":"k1","path":"/tmp/a","width":32,"size_bytes":1,"updated_at_unix":1,"touched_at_unix":2,"lines":["ok"]}`,
		`{broken-json`,
		"",
	}, "\n")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write index fixture: %v", err)
	}

	lines, ok, err := preview.LoadIndexEntry(p, "k1")
	if err != nil {
		t.Fatalf("loadPreviewIndexEntry: %v", err)
	}
	if !ok || len(lines) != 1 || lines[0] != "ok" {
		t.Fatalf("unexpected load result ok=%v lines=%#v", ok, lines)
	}

	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read compacted index: %v", err)
	}
	if strings.Contains(string(data), "{broken-json") {
		t.Fatalf("expected corrupted line removed after recovery, got: %q", string(data))
	}
}

func TestUpsertPreviewIndex_WaitsForLock(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "preview.v1.jsonl")
	lockPath := indexPath + ".lock"
	if err := os.WriteFile(lockPath, []byte("lock"), 0o644); err != nil {
		t.Fatalf("create lock file: %v", err)
	}

	go func() {
		time.Sleep(120 * time.Millisecond)
		_ = os.Remove(lockPath)
	}()

	rec := preview.IndexRecord{
		Key:           "k-lock",
		Path:          "/tmp/k-lock",
		Width:         16,
		SizeBytes:     2,
		UpdatedAtUnix: 3,
		TouchedAtUnix: time.Now().UnixNano(),
		Lines:         []string{"hello"},
	}
	if err := preview.UpsertIndex(indexPath, 10, rec); err != nil {
		t.Fatalf("upsertPreviewIndex with delayed lock release: %v", err)
	}
	lines, ok, err := preview.LoadIndexEntry(indexPath, rec.Key)
	if err != nil {
		t.Fatalf("load after upsert: %v", err)
	}
	if !ok || len(lines) != 1 || lines[0] != "hello" {
		t.Fatalf("unexpected entry after lock wait: ok=%v lines=%#v", ok, lines)
	}
}

func TestReadPreviewIndex_TrimsToByteBudget(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "preview.v1.jsonl")

	entries := map[string]preview.IndexRecord{
		"old": {
			Key:           "old",
			Path:          "/tmp/old",
			Width:         32,
			SizeBytes:     1,
			UpdatedAtUnix: 1,
			TouchedAtUnix: 1,
			Lines:         []string{strings.Repeat("a", preview.MaxIndexBytes)},
		},
		"new": {
			Key:           "new",
			Path:          "/tmp/new",
			Width:         32,
			SizeBytes:     1,
			UpdatedAtUnix: 2,
			TouchedAtUnix: 2,
			Lines:         []string{"keep-me"},
		},
	}
	if err := preview.RewriteIndex(indexPath, entries, 10, preview.MaxIndexBytes); err != nil {
		t.Fatalf("RewriteIndex: %v", err)
	}

	lines, ok, err := preview.LoadIndexEntry(indexPath, "new")
	if err != nil {
		t.Fatalf("loadPreviewIndexEntry(new): %v", err)
	}
	if !ok || len(lines) != 1 || lines[0] != "keep-me" {
		t.Fatalf("unexpected retained entry ok=%v lines=%#v", ok, lines)
	}
	if _, ok, err := preview.LoadIndexEntry(indexPath, "old"); err != nil {
		t.Fatalf("loadPreviewIndexEntry(old): %v", err)
	} else if ok {
		t.Fatal("expected oversized old entry to be trimmed from index")
	}
}
