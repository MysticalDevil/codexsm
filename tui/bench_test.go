package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/internal/core"
	"github.com/MysticalDevil/codexsm/session"
)

func makeBenchSessions(n int) []session.Session {
	base := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	out := make([]session.Session, 0, n)
	for i := 0; i < n; i++ {
		health := session.HealthOK
		if i%97 == 0 {
			health = session.HealthCorrupted
		} else if i%41 == 0 {
			health = session.HealthMissingMeta
		}
		out = append(out, session.Session{
			SessionID: fmt.Sprintf("%08d-1111-2222-3333-444444444444", i),
			UpdatedAt: base.Add(time.Duration(i) * time.Second),
			SizeBytes: int64(256 + i%2048),
			HostDir:   fmt.Sprintf("/workspace/host-%d", i%32),
			Health:    health,
			Path:      fmt.Sprintf("/tmp/sessions/%08d.jsonl", i),
		})
	}
	return out
}

func BenchmarkSortTUISessions_3k(b *testing.B) {
	source := makeBenchSessions(3000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		items := append([]session.Session(nil), source...)
		core.SortSessionsByRisk(items, nil, nil)
	}
}

func BenchmarkSortTUISessions_10k(b *testing.B) {
	source := makeBenchSessions(10000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		items := append([]session.Session(nil), source...)
		core.SortSessionsByRisk(items, nil, nil)
	}
}

func BenchmarkBuildPreviewLines_LargeSession(b *testing.B) {
	dir := b.TempDir()
	p := filepath.Join(dir, "large.jsonl")
	f, err := os.Create(p)
	if err != nil {
		b.Fatalf("create fixture: %v", err)
	}
	_, _ = fmt.Fprintln(f, `{"type":"session_meta","payload":{"id":"bench","timestamp":"2026-03-01T00:00:00Z","cwd":"/workspace/bench"}}`)
	for i := 0; i < 2500; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		_, _ = fmt.Fprintf(
			f,
			`{"type":"response_item","payload":{"type":"message","role":"%s","content":[{"type":"input_text","text":"benchmark line %d with multilingual 文本 emoji 😄 and extra tokens for wrapping behavior checks"}]}}`+"\n",
			role, i,
		)
	}
	_ = f.Close()

	theme := tuiTheme{Name: "tokyonight", Colors: cloneColorMap(builtinThemes["tokyonight"])}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lines := buildPreviewLines(p, 60, 24, theme)
		if len(lines) == 0 {
			b.Fatal("expected non-empty lines")
		}
	}
}

func BenchmarkBuildPreviewLines_OversizeUser(b *testing.B) {
	path := writePreviewBenchSession(b, "oversize-user", "user", strings.Repeat("U-LONG benchmark payload ", 6000), false)
	theme := tuiTheme{Name: "tokyonight", Colors: cloneColorMap(builtinThemes["tokyonight"])}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lines := buildPreviewLines(path, 72, 18, theme)
		if len(lines) == 0 {
			b.Fatal("expected preview lines")
		}
	}
}

func BenchmarkBuildPreviewLines_OversizeAssistant(b *testing.B) {
	path := writePreviewBenchSession(b, "oversize-assistant", "assistant", strings.Repeat("A-LONG stacktrace retry bounded preview ", 6000), true)
	theme := tuiTheme{Name: "tokyonight", Colors: cloneColorMap(builtinThemes["tokyonight"])}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lines := buildPreviewLines(path, 72, 18, theme)
		if len(lines) == 0 {
			b.Fatal("expected preview lines")
		}
	}
}

func BenchmarkBuildPreviewLines_UnicodeWide(b *testing.B) {
	path := writePreviewBenchSession(b, "unicode-wide", "user", strings.Repeat("请处理宽字符 👨‍👩‍👧‍👦 مرحبا שלום セッション復元 ", 3000), false)
	theme := tuiTheme{Name: "tokyonight", Colors: cloneColorMap(builtinThemes["tokyonight"])}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lines := buildPreviewLines(path, 72, 18, theme)
		if len(lines) == 0 {
			b.Fatal("expected preview lines")
		}
	}
}

func BenchmarkPreviewIndexLoad_1k(b *testing.B) {
	indexPath := writePreviewIndexBenchFile(b, 1000, false)
	key := "bench-0999"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lines, ok, err := loadPreviewIndexEntry(indexPath, key)
		if err != nil {
			b.Fatalf("loadPreviewIndexEntry: %v", err)
		}
		if !ok || len(lines) == 0 {
			b.Fatalf("expected index hit for %s", key)
		}
	}
}

func BenchmarkPreviewIndexUpsert_1k(b *testing.B) {
	indexPath := writePreviewIndexBenchFile(b, 1000, false)
	seed, err := os.ReadFile(indexPath)
	if err != nil {
		b.Fatalf("read seed index: %v", err)
	}
	baseTouched := time.Now().UnixNano()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		if err := os.WriteFile(indexPath, seed, 0o644); err != nil {
			b.Fatalf("reset index file: %v", err)
		}
		rec := previewIndexRecord{
			Key:           fmt.Sprintf("bench-upsert-%04d", i),
			Path:          fmt.Sprintf("/tmp/upsert/%04d.jsonl", i),
			Width:         72,
			SizeBytes:     2048,
			UpdatedAtUnix: baseTouched + int64(i),
			TouchedAtUnix: baseTouched + int64(i),
			Lines:         []string{"updated preview line", "second line"},
		}
		b.StartTimer()
		if err := upsertPreviewIndex(indexPath, 1200, rec); err != nil {
			b.Fatalf("upsertPreviewIndex: %v", err)
		}
	}
}

func BenchmarkPreviewIndexUpsert_Trimmed(b *testing.B) {
	indexPath := writePreviewIndexBenchFile(b, 1000, false)
	seed, err := os.ReadFile(indexPath)
	if err != nil {
		b.Fatalf("read seed index: %v", err)
	}
	baseTouched := time.Now().UnixNano()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		if err := os.WriteFile(indexPath, seed, 0o644); err != nil {
			b.Fatalf("reset index file: %v", err)
		}
		rec := previewIndexRecord{
			Key:           fmt.Sprintf("bench-trim-%04d", i),
			Path:          fmt.Sprintf("/tmp/trim/%04d.jsonl", i),
			Width:         72,
			SizeBytes:     4096,
			UpdatedAtUnix: baseTouched + int64(i),
			TouchedAtUnix: baseTouched + int64(i),
			Lines:         []string{strings.Repeat("trim-me ", maxPreviewIndexBytes/4)},
		}
		b.StartTimer()
		if err := upsertPreviewIndex(indexPath, 1200, rec); err != nil {
			b.Fatalf("upsertPreviewIndex(trimmed): %v", err)
		}
	}
}

func writePreviewBenchSession(b *testing.B, name, role, text string, assistantPriming bool) string {
	b.Helper()
	dir := b.TempDir()
	path := filepath.Join(dir, name+".jsonl")
	f, err := os.Create(path)
	if err != nil {
		b.Fatalf("create preview fixture: %v", err)
	}
	_, _ = fmt.Fprintln(f, `{"type":"session_meta","payload":{"id":"bench","timestamp":"2026-03-01T00:00:00Z","cwd":"/workspace/bench"}}`)
	if assistantPriming {
		_, _ = fmt.Fprintln(f, `{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"priming prompt"}]}}`)
	}
	switch role {
	case "assistant":
		_, _ = fmt.Fprintf(f, `{"type":"response_item","payload":{"type":"message","role":"assistant","text":%q,"content":[]}}`+"\n", text)
	default:
		_, _ = fmt.Fprintf(f, `{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":%q}]}}`+"\n", text)
	}
	_ = f.Close()
	return path
}

func writePreviewIndexBenchFile(b *testing.B, count int, includeLargeLines bool) string {
	b.Helper()
	dir := b.TempDir()
	indexPath := filepath.Join(dir, "preview.v1.jsonl")
	entries := make(map[string]previewIndexRecord, count)
	baseTouched := time.Now().UnixNano()
	for i := 0; i < count; i++ {
		lines := []string{fmt.Sprintf("preview line %04d", i), "secondary line"}
		if includeLargeLines {
			lines = []string{strings.Repeat("large-line ", 2048)}
		}
		key := fmt.Sprintf("bench-%04d", i)
		entries[key] = previewIndexRecord{
			Key:           key,
			Path:          fmt.Sprintf("/tmp/preview/%04d.jsonl", i),
			Width:         72,
			SizeBytes:     int64(1024 + i),
			UpdatedAtUnix: baseTouched + int64(i),
			TouchedAtUnix: baseTouched + int64(i),
			Lines:         lines,
		}
	}
	if err := rewritePreviewIndex(indexPath, entries, count, maxPreviewIndexBytes); err != nil {
		b.Fatalf("rewritePreviewIndex setup: %v", err)
	}
	return indexPath
}
