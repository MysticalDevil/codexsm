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

	out := renderChecks(checks, false, false)

	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected wrapped output, got lines=%d output=%q", len(lines), out)
	}

	basePrefix := strings.Repeat(" ", len("session_host_paths")+2+len("STATUS")+2)
	if !strings.HasPrefix(lines[2], basePrefix+detailContinuationIndent) {
		t.Fatalf("expected wrapped line to keep detail indentation, line=%q", lines[2])
	}
}

func TestRenderChecksColorizesDetailTokens(t *testing.T) {
	checks := []usecase.DoctorCheck{
		{
			Name:   "session_host_paths",
			Level:  usecase.DoctorWarn,
			Detail: "1. review: codexsm delete --dry-run=false --confirm /tmp/codexsm-fixture/worktrees/c",
		},
	}

	out := renderChecks(checks, true, false)
	if !strings.Contains(out, ansiCyanBold+"codexsm"+ansiReset) {
		t.Fatalf("expected codexsm to be colorized, got: %q", out)
	}

	if !strings.Contains(out, ansiRed+"delete"+ansiReset) {
		t.Fatalf("expected subcommand to be colorized, got: %q", out)
	}

	if !strings.Contains(out, ansiYellow+"--dry-run=false"+ansiReset) {
		t.Fatalf("expected flag to be colorized, got: %q", out)
	}

	if !strings.Contains(out, ansiDim+"/tmp/codexsm-fixture/worktrees/c"+ansiReset) {
		t.Fatalf("expected path to be colorized, got: %q", out)
	}

	if strings.Contains(out, ansiMagenta+"migrate"+ansiReset) {
		t.Fatalf("did not expect migrate to be colorized, got: %q", out)
	}
}

func TestCompactHomePathToken(t *testing.T) {
	home := "/home/tester"

	if got := compactHomePathToken("/home/tester/worktrees/codexsm", home); got != "~/worktrees/codexsm" {
		t.Fatalf("unexpected compacted path: %q", got)
	}

	if got := compactHomePathToken("(/home/tester/worktrees/codexsm)", home); got != "(~/worktrees/codexsm)" {
		t.Fatalf("unexpected compacted wrapped path: %q", got)
	}

	if got := compactHomePathToken("/home/tester2/worktrees/codexsm", home); got != "/home/tester2/worktrees/codexsm" {
		t.Fatalf("unexpected compacted unrelated path: %q", got)
	}
}
