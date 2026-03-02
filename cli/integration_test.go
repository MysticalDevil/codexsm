package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestList_DefaultLimitShowsLatest10(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 12; i++ {
		id := fmt.Sprintf("00000000-0000-0000-0000-%012d", i)
		p := filepath.Join(root, "2026", "03", "02", fmt.Sprintf("rollout-2026-03-02T17-39-%02d-%s.jsonl", i, id))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		meta := fmt.Sprintf(`{"type":"session_meta","payload":{"id":"%s","timestamp":"2026-03-02T09:44:00.024Z"}}\n`, id)
		if err := os.WriteFile(p, []byte(meta), 0o644); err != nil {
			t.Fatal(err)
		}
		mod := time.Now().Add(-time.Duration(12-i) * time.Minute)
		if err := os.Chtimes(p, mod, mod); err != nil {
			t.Fatal(err)
		}
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"list", "--sessions-root", root, "--color", "never"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("list execute: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "showing 10 of 12") {
		t.Fatalf("unexpected list footer: %q", out)
	}
	if count := strings.Count(out, "rollout-"); count != 10 {
		t.Fatalf("expected 10 rows, got %d", count)
	}
}

func TestList_FormatCSVAndTSV(t *testing.T) {
	root := t.TempDir()
	id := "99999999-9999-9999-9999-999999999999"
	path := filepath.Join(root, "2026", "03", "02", "rollout-csv.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	meta := `{"type":"session_meta","payload":{"id":"` + id + `","timestamp":"2026-03-02T09:44:00.024Z"}}` + "\n"
	if err := os.WriteFile(path, []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name    string
		format  string
		header  string
		contain string
	}{
		{name: "csv", format: "csv", header: "session_id,created_at,updated_at,size_bytes,health,path", contain: id + ","},
		{name: "tsv", format: "tsv", header: "session_id\tcreated_at\tupdated_at\tsize_bytes\thealth\tpath", contain: id + "\t"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewRootCmd()
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			cmd.SetOut(stdout)
			cmd.SetErr(stderr)
			cmd.SetArgs([]string{"list", "--sessions-root", root, "--format", tc.format, "--limit", "1"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("list execute: %v", err)
			}
			out := stdout.String()
			if !strings.Contains(out, tc.header) {
				t.Fatalf("missing header in output: %q", out)
			}
			if !strings.Contains(out, tc.contain) {
				t.Fatalf("missing session row in output: %q", out)
			}
		})
	}
}

func TestList_NoHeaderAndColumn(t *testing.T) {
	root := t.TempDir()
	id := "88888888-8888-8888-8888-888888888888"
	path := filepath.Join(root, "2026", "03", "02", "rollout-columns.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	meta := `{"type":"session_meta","payload":{"id":"` + id + `","timestamp":"2026-03-02T09:44:00.024Z"}}` + "\n"
	if err := os.WriteFile(path, []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"list",
		"--sessions-root", root,
		"--format", "csv",
		"--no-header",
		"--column", "session_id,health",
		"--limit", "1",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list execute: %v", err)
	}
	out := strings.TrimSpace(stdout.String())
	if strings.Contains(out, "session_id,health") {
		t.Fatalf("unexpected header in output: %q", out)
	}
	if !strings.Contains(out, id+",ok") {
		t.Fatalf("unexpected row output: %q", out)
	}
}

func TestDelete_DryRunWritesAuditAndKeepsFile(t *testing.T) {
	root := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "actions.log")

	id := "11111111-1111-1111-1111-111111111111"
	sessionPath := filepath.Join(root, "2026", "03", "02", "rollout-2026-03-02T17-44-00-11111111-1111-1111-1111-111111111111.jsonl")
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o755); err != nil {
		t.Fatal(err)
	}
	meta := `{"type":"session_meta","payload":{"id":"11111111-1111-1111-1111-111111111111","timestamp":"2026-03-02T09:44:00.024Z"}}` + "\n"
	if err := os.WriteFile(sessionPath, []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"delete", "--sessions-root", root, "--id", id, "--dry-run", "--log-file", logFile})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("delete execute: %v", err)
	}
	if _, err := os.Stat(sessionPath); err != nil {
		t.Fatalf("session file should remain on dry-run: %v", err)
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	line := strings.TrimSpace(string(data))
	if line == "" {
		t.Fatal("expected one audit log line")
	}

	var rec map[string]any
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		t.Fatalf("invalid audit json: %v", err)
	}
	sim, ok := rec["simulation"].(bool)
	if !ok || !sim {
		t.Fatalf("expected simulation=true, got: %v", rec["simulation"])
	}
}

func TestDelete_RealDeleteRequiresConfirm(t *testing.T) {
	root := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "actions.log")
	id := "22222222-2222-2222-2222-222222222222"
	p := filepath.Join(root, "2026", "03", "02", "rollout-2026-03-02T17-44-00-22222222-2222-2222-2222-222222222222.jsonl")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	meta := `{"type":"session_meta","payload":{"id":"22222222-2222-2222-2222-222222222222","timestamp":"2026-03-02T09:44:00.024Z"}}` + "\n"
	if err := os.WriteFile(p, []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"delete", "--sessions-root", root, "--id", id, "--dry-run=false", "--interactive-confirm=false", "--log-file", logFile})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when real delete misses --confirm")
	}
	if !strings.Contains(err.Error(), "--confirm") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(p); statErr != nil {
		t.Fatalf("session file should remain when validation fails: %v", statErr)
	}
}

func TestDelete_RealDeleteInteractiveConfirmDisabledNeedsYes(t *testing.T) {
	root := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "actions.log")
	p1 := filepath.Join(root, "2026", "03", "02", "rollout-2026-03-02T17-44-00-33333333-3333-3333-3333-333333333333.jsonl")
	if err := os.MkdirAll(filepath.Dir(p1), 0o755); err != nil {
		t.Fatal(err)
	}
	meta := `{"type":"session_meta","payload":{"id":"33333333-3333-3333-3333-333333333333","timestamp":"2026-03-02T09:44:00.024Z"}}` + "\n"
	if err := os.WriteFile(p1, []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	p2 := filepath.Join(root, "2026", "03", "02", "rollout-2026-03-02T17-44-01-33333333-3333-3333-3333-aaaaaaaaaaaa.jsonl")
	meta2 := `{"type":"session_meta","payload":{"id":"33333333-3333-3333-3333-aaaaaaaaaaaa","timestamp":"2026-03-02T09:45:00.024Z"}}` + "\n"
	if err := os.WriteFile(p2, []byte(meta2), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(bytes.NewBufferString(""))
	cmd.SetArgs([]string{"delete", "--sessions-root", root, "--id-prefix", "33333333-3333-3333-3333", "--dry-run=false", "--confirm", "--interactive-confirm=false", "--log-file", logFile})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when neither --yes nor interactive confirm is used")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(p1); statErr != nil {
		t.Fatalf("session file #1 should remain when validation fails: %v", statErr)
	}
	if _, statErr := os.Stat(p2); statErr != nil {
		t.Fatalf("session file #2 should remain when validation fails: %v", statErr)
	}
}

func TestDelete_RealSoftDeleteMovesToTrash(t *testing.T) {
	root := t.TempDir()
	trashRoot := filepath.Join(t.TempDir(), "trash")
	logFile := filepath.Join(t.TempDir(), "actions.log")
	id := "44444444-4444-4444-4444-444444444444"
	filename := "rollout-2026-03-02T17-44-00-44444444-4444-4444-4444-444444444444.jsonl"
	src := createSessionFile(t, root, "2026/03/02/"+filename, id)

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"delete",
		"--sessions-root", root,
		"--trash-root", trashRoot,
		"--log-file", logFile,
		"--id", id,
		"--dry-run=false",
		"--confirm",
		"--interactive-confirm=false",
		"--yes",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("delete execute: %v", err)
	}
	if _, err := os.Stat(src); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("source should be moved; stat err=%v", err)
	}
	dst := filepath.Join(trashRoot, "sessions", "2026", "03", "02", filename)
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("expected trashed file at %s: %v", dst, err)
	}
	out := stdout.String()
	if !strings.Contains(out, "action=soft-delete") || !strings.Contains(out, "succeeded=1") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestDelete_RealHardDeleteRemovesFile(t *testing.T) {
	root := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "actions.log")
	id := "55555555-5555-5555-5555-555555555555"
	src := createSessionFile(t, root, "2026/03/02/rollout-2026-03-02T17-44-00-55555555-5555-5555-5555-555555555555.jsonl", id)

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"delete",
		"--sessions-root", root,
		"--id", id,
		"--dry-run=false",
		"--confirm",
		"--hard",
		"--interactive-confirm=false",
		"--yes",
		"--log-file", logFile,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("delete execute: %v", err)
	}
	if _, err := os.Stat(src); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("source should be deleted; stat err=%v", err)
	}
}

func TestDelete_MaxBatchGuardBlocksRealDelete(t *testing.T) {
	root := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "actions.log")
	p1 := createSessionFile(t, root, "2026/03/02/rollout-1.jsonl", "66666666-6666-6666-6666-aaaaaaaaaaaa")
	p2 := createSessionFile(t, root, "2026/03/02/rollout-2.jsonl", "66666666-6666-6666-6666-bbbbbbbbbbbb")
	p3 := createSessionFile(t, root, "2026/03/02/rollout-3.jsonl", "66666666-6666-6666-6666-cccccccccccc")

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"delete",
		"--sessions-root", root,
		"--id-prefix", "66666666-6666-6666-6666",
		"--dry-run=false",
		"--confirm",
		"--interactive-confirm=false",
		"--yes",
		"--max-batch", "2",
		"--log-file", logFile,
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected max-batch guard error")
	}
	if !strings.Contains(err.Error(), "max-batch") {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, p := range []string{p1, p2, p3} {
		if _, statErr := os.Stat(p); statErr != nil {
			t.Fatalf("file should remain after guard rejection: %v", statErr)
		}
	}
}

func TestDelete_RequiresSelector(t *testing.T) {
	root := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "actions.log")
	createSessionFile(t, root, "2026/03/02/rollout-1.jsonl", "77777777-7777-7777-7777-777777777777")

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"delete", "--sessions-root", root, "--log-file", logFile})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected selector validation error")
	}
	if !strings.Contains(err.Error(), "at least one selector") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_RealDeleteShowsPreview(t *testing.T) {
	root := t.TempDir()
	trashRoot := filepath.Join(t.TempDir(), "trash")
	logFile := filepath.Join(t.TempDir(), "actions.log")
	id := "88888888-1111-2222-3333-444444444444"
	createSessionFile(t, root, "2026/03/02/rollout-preview.jsonl", id)

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"delete",
		"--sessions-root", root,
		"--trash-root", trashRoot,
		"--id", id,
		"--dry-run=false",
		"--confirm",
		"--interactive-confirm=false",
		"--yes",
		"--log-file", logFile,
		"--preview-limit", "1",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("delete execute: %v", err)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "preview action=soft-delete matched=1") {
		t.Fatalf("expected preview output, got: %q", errOut)
	}
}

func TestRestore_DryRunDoesNotMoveFile(t *testing.T) {
	root := t.TempDir()
	trashRoot := filepath.Join(t.TempDir(), "trash")
	logFile := filepath.Join(t.TempDir(), "actions.log")
	id := "99990000-1111-2222-3333-444444444444"
	trashed := createSessionFile(t, filepath.Join(trashRoot, "sessions"), "2026/03/02/rollout-r1.jsonl", id)

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"restore",
		"--sessions-root", root,
		"--trash-root", trashRoot,
		"--id", id,
		"--dry-run",
		"--log-file", logFile,
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("restore execute: %v", err)
	}
	if _, err := os.Stat(trashed); err != nil {
		t.Fatalf("trashed file should remain in dry-run: %v", err)
	}
	dst := filepath.Join(root, "2026", "03", "02", "rollout-r1.jsonl")
	if _, err := os.Stat(dst); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("destination should not exist in dry-run: %v", err)
	}
}

func TestRestore_RealMovesFileBack(t *testing.T) {
	root := t.TempDir()
	trashRoot := filepath.Join(t.TempDir(), "trash")
	logFile := filepath.Join(t.TempDir(), "actions.log")
	id := "99990000-1111-2222-3333-aaaaaaaaaaaa"
	trashed := createSessionFile(t, filepath.Join(trashRoot, "sessions"), "2026/03/02/rollout-r2.jsonl", id)

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"restore",
		"--sessions-root", root,
		"--trash-root", trashRoot,
		"--id", id,
		"--dry-run=false",
		"--confirm",
		"--interactive-confirm=false",
		"--yes",
		"--log-file", logFile,
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("restore execute: %v", err)
	}
	if _, err := os.Stat(trashed); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("trashed file should be moved out: %v", err)
	}
	dst := filepath.Join(root, "2026", "03", "02", "rollout-r2.jsonl")
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("destination should exist after restore: %v", err)
	}
}

func createSessionFile(t *testing.T, root, relPath, id string) string {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	meta := fmt.Sprintf(`{"type":"session_meta","payload":{"id":"%s","timestamp":"2026-03-02T09:44:00.024Z"}}`+"\n", id)
	if err := os.WriteFile(p, []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}
