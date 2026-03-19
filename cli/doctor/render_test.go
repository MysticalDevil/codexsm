package doctor

import (
	"strings"
	"testing"

	"github.com/MysticalDevil/codexsm/usecase"
)

func TestRenderChecksWrapsDetailToColumns(t *testing.T) {
	t.Setenv("COLUMNS", "72")

	checks := []usecase.DoctorCheck{
		{
			Name:  "session_host_paths",
			Level: usecase.DoctorWarn,
			Detail: "recommended_actions: 1. review: codexsm list --host-contains " +
				"/tmp/codexsm-fixture/worktrees/c",
		},
	}

	out := renderChecks(checks, false)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected wrapped output, got lines=%d output=%q", len(lines), out)
	}

	basePrefix := strings.Repeat(" ", len("session_host_paths")+2+len("STATUS")+2)
	if !strings.HasPrefix(lines[2], basePrefix+detailContinuationIndent) {
		t.Fatalf("expected wrapped line to keep detail indentation, line=%q", lines[2])
	}
}
