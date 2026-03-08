package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

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
		sortTUISessions(items)
	}
}

func BenchmarkSortTUISessions_10k(b *testing.B) {
	source := makeBenchSessions(10000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		items := append([]session.Session(nil), source...)
		sortTUISessions(items)
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
