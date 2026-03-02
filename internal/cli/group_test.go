package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGroupByHealthJSON(t *testing.T) {
	root := t.TempDir()
	mustWriteSession(t, root, "a", "{\"type\":\"session_meta\",\"payload\":{\"id\":\"a\",\"timestamp\":\"2026-03-02T09:44:00.024Z\"}}\n", time.Now().Add(-1*time.Hour))
	mustWriteSession(t, root, "b", "{\"type\":\"session_meta\",\"payload\":{\"id\":\"b\",\"timestamp\":\"2026-03-02T09:44:00.024Z\"}}\n", time.Now().Add(-2*time.Hour))
	bad := filepath.Join(root, "2026", "03", "02", "bad.jsonl")
	if err := os.MkdirAll(filepath.Dir(bad), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("{bad-json\n"), 0o644); err != nil {
		t.Fatal(err)
	}

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
	if len(got) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(got))
	}
	if got[0].Group != "ok" || got[0].Count != 2 {
		t.Fatalf("unexpected first group: %+v", got[0])
	}
}

func TestGroupFormatCSVAndTSV(t *testing.T) {
	root := t.TempDir()
	mustWriteSession(t, root, "c", "{\"type\":\"session_meta\",\"payload\":{\"id\":\"c\",\"timestamp\":\"2026-03-02T09:44:00.024Z\"}}\n", time.Now())

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
	root := t.TempDir()
	base := time.Date(2026, 3, 2, 18, 0, 0, 0, time.Local)
	mustWriteSession(t, root, "d1", "{\"type\":\"session_meta\",\"payload\":{\"id\":\"d1\",\"timestamp\":\"2026-03-02T09:44:00.024Z\"}}\n", base)
	mustWriteSession(t, root, "d2", "{\"type\":\"session_meta\",\"payload\":{\"id\":\"d2\",\"timestamp\":\"2026-03-02T09:44:00.024Z\"}}\n", base.Add(-1*time.Hour))
	mustWriteSession(t, root, "d3", "{\"type\":\"session_meta\",\"payload\":{\"id\":\"d3\",\"timestamp\":\"2026-03-01T09:44:00.024Z\"}}\n", base.Add(-30*time.Hour))

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
	if got[0].Count != 2 {
		t.Fatalf("expected top group count=2, got %+v", got[0])
	}
}

func TestGroupInvalidSortReturnsError(t *testing.T) {
	root := t.TempDir()
	mustWriteSession(t, root, "x", "{\"type\":\"session_meta\",\"payload\":{\"id\":\"x\",\"timestamp\":\"2026-03-02T09:44:00.024Z\"}}\n", time.Now())

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"group", "--sessions-root", root, "--sort", "invalid"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected invalid sort error")
	}
}

func mustWriteSession(t *testing.T, root, id, firstLine string, mod time.Time) {
	t.Helper()
	path := filepath.Join(root, "2026", "03", "02", "rollout-"+id+".jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(firstLine), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatal(err)
	}
}
