package session

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
)

func BenchmarkScanSessions(b *testing.B) {
	root := filepath.Join(testsupport.TestdataRoot(), "fixtures", "rich", "sessions")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessions, err := ScanSessions(root)
		if err != nil {
			b.Fatalf("ScanSessions: %v", err)
		}
		if len(sessions) == 0 {
			b.Fatal("expected non-empty sessions")
		}
	}
}

func BenchmarkFilterSessions(b *testing.B) {
	root := filepath.Join(testsupport.TestdataRoot(), "fixtures", "rich", "sessions")
	sessions, err := ScanSessions(root)
	if err != nil {
		b.Fatalf("ScanSessions setup: %v", err)
	}
	if len(sessions) == 0 {
		b.Fatal("expected non-empty sessions")
	}

	now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		sel  Selector
	}{
		{
			name: "all",
			sel:  Selector{},
		},
		{
			name: "host_head_health",
			sel: Selector{
				HostContains: "workspace",
				HeadContains: "session",
				Health:       HealthOK,
				HasHealth:    true,
			},
		},
		{
			name: "older_than",
			sel: Selector{
				OlderThan:    24 * time.Hour,
				HasOlderThan: true,
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				out := FilterSessions(sessions, tc.sel, now)
				if len(out) == 0 && tc.name == "all" {
					b.Fatal("unexpected empty result for all selector")
				}
			}
		})
	}
}
