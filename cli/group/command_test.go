package group

import (
	"bytes"
	"strings"
	"testing"

	"github.com/MysticalDevil/codexsm/usecase"
)

func TestGroupRenderHelpers(t *testing.T) {
	stats := []usecase.GroupStat{{Group: "ok", Count: 2, SizeBytes: 1024, Latest: "2026-03-02 10:00:00"}}

	table, err := renderGroupTable(stats, "health", "never", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("renderGroupTable: %v", err)
	}

	if !strings.Contains(table, "GROUP") || !strings.Contains(table, "groups=1 by=health") {
		t.Fatalf("unexpected group table: %q", table)
	}

	colored := colorizeGroupedTable(table)
	if !strings.Contains(colored, "\x1b[") {
		t.Fatalf("expected ANSI colorized output: %q", colored)
	}
}
