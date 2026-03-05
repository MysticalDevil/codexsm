package cli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
	"github.com/MysticalDevil/codexsm/session"
)

func BenchmarkRenderTable(b *testing.B) {
	root := filepath.Join(testsupport.TestdataRoot(), "fixtures", "rich", "sessions")
	sessions, err := session.ScanSessions(root)
	if err != nil {
		b.Fatalf("ScanSessions setup: %v", err)
	}
	if len(sessions) == 0 {
		b.Fatal("expected non-empty sessions")
	}

	cols, err := parseListColumns("id,updated_at,size,health,host,head", false, "table")
	if err != nil {
		b.Fatalf("parseListColumns setup: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out := &bytes.Buffer{}
		table, err := renderTable(sessions, len(sessions), listRenderOptions{
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
