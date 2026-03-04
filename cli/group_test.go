package cli

import (
	"bytes"
	"encoding/json/v2"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MysticalDevil/codex-sm/internal/testsupport"
)

const groupFixtureName = "rich"

func fixtureSessionsRoot(t *testing.T) string {
	t.Helper()
	workspace := testsupport.PrepareFixtureSandbox(t, groupFixtureName)
	return filepath.Join(workspace, "sessions")
}

func TestGroupByHealthJSON(t *testing.T) {
	root := fixtureSessionsRoot(t)

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"group", "--sessions-root", root, "--by", "health", "--format", "json", "--color", "never"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("group execute: %v", err)
	}

	var got []groupStat
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected non-empty groups")
	}
	foundOK := false
	for _, st := range got {
		if st.Group == "ok" && st.Count > 0 {
			foundOK = true
			break
		}
	}
	if !foundOK {
		t.Fatalf("expected 'ok' health group, got: %+v", got)
	}
}

func TestGroupFormatCSVAndTSV(t *testing.T) {
	root := fixtureSessionsRoot(t)

	for _, tc := range []struct {
		name   string
		format string
		header string
	}{
		{name: "csv", format: "csv", header: "group,count,size_bytes,latest"},
		{name: "tsv", format: "tsv", header: "group\tcount\tsize_bytes\tlatest"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewRootCmd()
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			cmd.SetOut(stdout)
			cmd.SetErr(stderr)
			cmd.SetArgs([]string{"group", "--sessions-root", root, "--by", "day", "--format", tc.format, "--color", "never"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("group execute: %v", err)
			}
			out := stdout.String()
			if !strings.Contains(out, tc.header) {
				t.Fatalf("missing header in output: %q", out)
			}
		})
	}
}

func TestGroupSortAndLimit(t *testing.T) {
	root := fixtureSessionsRoot(t)

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"group", "--sessions-root", root, "--by", "day", "--sort", "count", "--order", "desc", "--limit", "1", "--format", "json", "--color", "never"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("group execute: %v", err)
	}

	var got []groupStat
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 group, got %d", len(got))
	}
	if got[0].Count < 1 {
		t.Fatalf("expected positive count, got %+v", got[0])
	}
}

func TestGroupInvalidSortReturnsError(t *testing.T) {
	root := fixtureSessionsRoot(t)

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"group", "--sessions-root", root, "--sort", "invalid"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected invalid sort error")
	}
}
