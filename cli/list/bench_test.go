package list

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/session"
)

func BenchmarkRenderTable(b *testing.B) {
	sessions := makeCLIBenchSessions(1200)

	cols, err := ParseColumns("id,updated_at,size,health,host,head", false, "table")
	if err != nil {
		b.Fatalf("parseListColumns setup: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		out := &bytes.Buffer{}

		table, err := RenderTable(sessions, len(sessions), RenderOptions{
			NoHeader:  false,
			ColorMode: "never",
			Out:       out,
			Columns:   cols,
			HeadWidth: 36,
		})
		if err != nil {
			b.Fatalf("renderTable: %v", err)
		}

		if len(table) == 0 {
			b.Fatal("expected non-empty output")
		}
	}
}

func BenchmarkRenderTable_LargeColumns(b *testing.B) {
	sessions := makeCLIBenchSessions(1200)

	cols, err := ParseColumns("session_id,created_at,updated_at,size_bytes,health,host_dir,head,path", true, "table")
	if err != nil {
		b.Fatalf("parseListColumns setup: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		out := &bytes.Buffer{}

		table, err := RenderTable(sessions, len(sessions), RenderOptions{
			NoHeader:  false,
			ColorMode: "never",
			Out:       out,
			Columns:   cols,
			HeadWidth: 64,
		})
		if err != nil {
			b.Fatalf("renderTable: %v", err)
		}

		if len(table) == 0 {
			b.Fatal("expected non-empty output")
		}
	}
}

func BenchmarkRenderJSON(b *testing.B) {
	sessions := makeCLIBenchSessions(1200)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := &bytes.Buffer{}

		data, err := json.Marshal(sessions)
		if err != nil {
			b.Fatalf("json.Marshal: %v", err)
		}

		if _, err := buf.Write(append(data, '\n')); err != nil {
			b.Fatalf("write json output: %v", err)
		}

		if buf.Len() == 0 {
			b.Fatal("expected non-empty json output")
		}
	}
}

func makeCLIBenchSessions(n int) []session.Session {
	base := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)

	out := make([]session.Session, 0, n)
	for i := 0; i < n; i++ {
		health := session.HealthOK
		if i%97 == 0 {
			health = session.HealthCorrupted
		} else if i%41 == 0 {
			health = session.HealthMissingMeta
		}

		out = append(out, session.Session{
			SessionID: fmt.Sprintf("%08x-1111-2222-3333-%012x", i, i),
			CreatedAt: base.Add(time.Duration(i) * time.Minute),
			UpdatedAt: base.Add(time.Duration(i) * time.Minute),
			SizeBytes: int64(1024 + i*3),
			HostDir:   fmt.Sprintf("/workspace/bench/host-%02d/project-%02d", i%24, i%9),
			Health:    health,
			Path:      fmt.Sprintf("/workspace/bench/host-%02d/project-%02d/session-%04d.jsonl", i%24, i%9, i),
			Head:      fmt.Sprintf("benchmark head %04d with multilingual 文本 and emoji 😄 for output width checks", i),
		})
	}

	return out
}
