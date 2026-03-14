package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
	"github.com/MysticalDevil/codexsm/session"
	"github.com/MysticalDevil/codexsm/tui/preview"
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
		previewCache:       make(map[string][]string),
		previewBytesBudget: 2,
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

func TestPreviewCacheByteBudget(t *testing.T) {
	m := tuiModel{
		previewCache:       make(map[string][]string),
		previewBytesBudget: 5,
	}
	m.previewCachePut("a", []string{"1234"})
	m.previewCachePut("b", []string{"12"})
	if _, ok := m.previewCachePeek("a"); ok {
		t.Fatal("expected a evicted by byte budget")
	}
	if _, ok := m.previewCachePeek("b"); !ok {
		t.Fatal("expected b present")
	}
}

func TestPreviewIndexRoundTrip(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	indexPath := filepath.Join(workspace, "tmp", "preview.v1.jsonl")
	rec := preview.IndexRecord{
		Key:           "k-a",
		Path:          "/tmp/a.jsonl",
		Width:         32,
		SizeBytes:     11,
		UpdatedAtUnix: 22,
		TouchedAtUnix: 33,
		Lines:         []string{"l1", "l2"},
	}
	if err := preview.UpsertIndex(indexPath, 100, rec); err != nil {
		t.Fatalf("upsertPreviewIndex: %v", err)
	}
	lines, ok, err := preview.LoadIndexEntry(indexPath, rec.Key)
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
	loaded, ok := msg.(preview.LoadedMsg)
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

func TestBuildPreviewLinesExtremeStaticFixtures(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "extreme-static")
	sessionsRoot := filepath.Join(workspace, "sessions", "2026", "03", "09")
	theme := tuiTheme{Name: "tokyonight", Colors: cloneColorMap(builtinThemes["tokyonight"])}

	cases := []struct {
		name string
		file string
		want string
	}{
		{name: "oversize-user", file: "oversize-user-message-001.jsonl", want: "U-LONG-START"},
		{name: "assistant-text-fallback", file: "oversize-assistant-message-001.jsonl", want: "A-LONG-START"},
		{name: "unicode-wide", file: "unicode-wide-long-001.jsonl", want: "超长宽字符会话"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			lines := buildPreviewLines(filepath.Join(sessionsRoot, tc.file), 72, 10, theme)
			if len(lines) == 0 {
				t.Fatal("expected preview lines")
			}
			if !strings.Contains(strings.Join(lines, "\n"), tc.want) {
				t.Fatalf("preview missing %q in %#v", tc.want, lines)
			}
		})
	}
}

func TestBuildPreviewLinesSingleLineWithoutTrailingNewline(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "extreme-static")
	path := filepath.Join(workspace, "sessions", "2026", "03", "09", "single-line-no-newline-001.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if len(data) > 0 && data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("rewrite file without trailing newline: %v", err)
	}

	theme := tuiTheme{Name: "tokyonight", Colors: cloneColorMap(builtinThemes["tokyonight"])}
	lines := buildPreviewLines(path, 72, 10, theme)
	if len(lines) == 0 {
		t.Fatal("expected preview lines")
	}
	if !strings.Contains(strings.Join(lines, "\n"), "single line no newline fixture") {
		t.Fatalf("unexpected preview lines: %#v", lines)
	}
}
