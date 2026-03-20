package restore

import (
	"bytes"
	"strings"
	"testing"

	"github.com/MysticalDevil/codexsm/internal/ops"
	"github.com/MysticalDevil/codexsm/session"
	"github.com/spf13/cobra"
)

func TestPrintRestoreSummaryIncludesResultError(t *testing.T) {
	cmd := &cobra.Command{}
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)

	PrintRestoreSummary(cmd, session.RestoreSummary{
		Action:       "restore",
		Simulation:   true,
		MatchedCount: 1,
		Succeeded:    0,
		Failed:       1,
		Skipped:      0,
		Results: []session.DeleteResult{
			{
				Status:    "failed",
				SessionID: "abc123",
				Path:      "/tmp/a.jsonl",
				Error:     "boom",
			},
		},
	})

	out := stdout.String()
	if !strings.Contains(out, "action=restore simulation=true matched=1 succeeded=0 failed=1") {
		t.Fatalf("unexpected summary line: %q", out)
	}

	if !strings.Contains(out, "failed abc123 /tmp/a.jsonl err=boom") {
		t.Fatalf("expected error result line, got: %q", out)
	}
}

func TestPrintRestorePreviewSampleShowsRemainingCount(t *testing.T) {
	cmd := &cobra.Command{}
	stderr := &bytes.Buffer{}
	cmd.SetErr(stderr)

	candidates := []session.Session{
		{SessionID: "11111111-1111-1111-1111-111111111111", Path: "/tmp/1.jsonl", SizeBytes: 1024},
		{SessionID: "22222222-2222-2222-2222-222222222222", Path: "/tmp/2.jsonl", SizeBytes: 2048},
		{SessionID: "33333333-3333-3333-3333-333333333333", Path: "/tmp/3.jsonl", SizeBytes: 4096},
	}

	PrintRestorePreview(cmd, candidates, ops.PreviewSample, 1)

	out := stderr.String()
	if !strings.Contains(out, "preview action=restore matched=3") {
		t.Fatalf("unexpected preview header: %q", out)
	}

	if !strings.Contains(out, "... and 2 more") {
		t.Fatalf("expected remaining count line, got: %q", out)
	}
}

func TestInteractiveConfirmRestoreRequiresTerminalStdin(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBufferString("y\n"))
	cmd.SetErr(&bytes.Buffer{})

	ok, err := InteractiveConfirmRestore(cmd, 2)
	if err == nil {
		t.Fatal("expected non-terminal stdin error")
	}

	if ok {
		t.Fatal("expected confirmation=false for non-terminal stdin")
	}

	if !strings.Contains(err.Error(), "requires a terminal stdin") {
		t.Fatalf("unexpected error: %v", err)
	}
}
