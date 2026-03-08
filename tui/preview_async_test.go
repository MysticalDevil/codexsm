package tui

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
	"github.com/MysticalDevil/codexsm/session"
)

func TestPreviewCacheKeyForSessionIncludesWidth(t *testing.T) {
	s := session.Session{
		Path:      "/tmp/a.jsonl",
		SizeBytes: 123,
		UpdatedAt: time.Unix(1700000000, 12345),
	}
	k1 := previewCacheKeyForSession(s, 24)
	k2 := previewCacheKeyForSession(s, 48)
	if k1 == k2 {
		t.Fatalf("expected different keys by width, got same: %q", k1)
	}
}

func TestPreviewCacheLRUEviction(t *testing.T) {
	m := tuiModel{
		previewCache: make(map[string][]string),
		previewCap:   2,
	}

	m.previewCachePut("k1", []string{"1"})
	m.previewCachePut("k2", []string{"2"})
	if _, ok := m.previewCacheGet("k1"); !ok {
		t.Fatal("expected k1 in cache")
	}
	m.previewCachePut("k3", []string{"3"})

	if _, ok := m.previewCachePeek("k2"); ok {
		t.Fatal("expected k2 to be evicted as LRU")
	}
	if _, ok := m.previewCachePeek("k1"); !ok {
		t.Fatal("expected k1 to remain")
	}
	if _, ok := m.previewCachePeek("k3"); !ok {
		t.Fatal("expected k3 to remain")
	}
}

func TestPreviewIndexRoundTrip(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	indexPath := filepath.Join(workspace, "tmp", "preview.v1.jsonl")
	rec := previewIndexRecord{
		Key:           "k-a",
		Path:          "/tmp/a.jsonl",
		Width:         32,
		SizeBytes:     11,
		UpdatedAtUnix: 22,
		TouchedAtUnix: 33,
		Lines:         []string{"l1", "l2"},
	}
	if err := upsertPreviewIndex(indexPath, 100, rec); err != nil {
		t.Fatalf("upsertPreviewIndex: %v", err)
	}
	lines, ok, err := loadPreviewIndexEntry(indexPath, rec.Key)
	if err != nil {
		t.Fatalf("loadPreviewIndexEntry: %v", err)
	}
	if !ok {
		t.Fatal("expected index hit")
	}
	if len(lines) != 2 || lines[0] != "l1" || lines[1] != "l2" {
		t.Fatalf("unexpected lines: %#v", lines)
	}
}

func TestEnsurePreviewRequestAndUpdatePipeline(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	p := filepath.Join(workspace, "sessions", "2026", "03", "02", "rollout-preview.jsonl")
	m := tuiModel{
		sessions: []session.Session{
			{
				SessionID: "x1",
				Path:      p,
				SizeBytes: 1,
				UpdatedAt: time.Now(),
				Health:    session.HealthOK,
			},
		},
		previewCache: make(map[string][]string),
		width:        120,
		height:       36,
		theme:        tuiTheme{Name: "tokyonight", Colors: cloneColorMap(builtinThemes["tokyonight"])},
	}
	m.rebuildTree()

	cmd := m.ensurePreviewRequest()
	if cmd == nil {
		t.Fatal("expected preview load cmd")
	}
	msg := cmd()
	loaded, ok := msg.(previewLoadedMsg)
	if !ok {
		t.Fatalf("unexpected msg type: %T", msg)
	}
	model, persistCmd := m.Update(loaded)
	updated := model.(tuiModel)
	if updated.previewWait != "" {
		t.Fatalf("expected previewWait cleared, got %q", updated.previewWait)
	}
	if _, ok := updated.previewCachePeek(loaded.Key); !ok {
		t.Fatalf("expected cached preview for key %q", loaded.Key)
	}
	if persistCmd == nil {
		t.Fatal("expected persist cmd")
	}
}
