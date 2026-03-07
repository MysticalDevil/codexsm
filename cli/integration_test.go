//go:build integration
// +build integration

package cli

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
	"github.com/spf13/cobra"
)

const (
	fixtureName = "rich"

	idCSV        = "99999999-9999-9999-9999-999999999999"
	idHomeHost   = "34343434-3434-3434-3434-343434343434"
	idNonHome    = "45454545-4545-4545-4545-454545454545"
	idColumns    = "88888888-8888-8888-8888-888888888888"
	idHeadWidth  = "12121212-1212-1212-1212-121212121212"
	idDefault    = "23232323-2323-2323-2323-232323232323"
	idDeleteDry  = "11111111-1111-1111-1111-111111111111"
	idDeleteNeed = "22222222-2222-2222-2222-222222222222"
	idSoftDelete = "44444444-4444-4444-4444-444444444444"
	idHardDelete = "55555555-5555-5555-5555-555555555555"
	idPreview    = "88888888-1111-2222-3333-444444444444"
	idRestoreR1  = "99990000-1111-2222-3333-444444444444"
	idRestoreR2  = "99990000-1111-2222-3333-aaaaaaaaaaaa"
)

func fixtureRoots(t *testing.T) (workspace, sessionsRoot, trashRoot, logFile string) {
	t.Helper()
	workspace = testsupport.PrepareFixtureSandbox(t, fixtureName)
	sessionsRoot = filepath.Join(workspace, "sessions")
	trashRoot = filepath.Join(workspace, "trash")
	logFile = filepath.Join(workspace, "logs", "actions.log")
	return workspace, sessionsRoot, trashRoot, logFile
}

func newIsolatedRootCmd(t *testing.T, sessionsRoot string) *cobra.Command {
	t.Helper()
	t.Setenv("SESSIONS_ROOT", sessionsRoot)
	return NewRootCmd()
}

func TestList_DefaultLimitShowsLatest10(t *testing.T) {
	_, root, _, _ := fixtureRoots(t)
	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"list", "--sessions-root", root, "--color", "never"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("list execute: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "showing 10 of ") {
		t.Fatalf("unexpected list footer: %q", out)
	}
	if !strings.Contains(out, "HEAD") {
		t.Fatalf("expected HEAD column in output: %q", out)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	count := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "ID") || strings.HasPrefix(line, "showing ") {
			continue
		}
		count++
	}
	if count != 10 {
		t.Fatalf("expected 10 rows, got %d", count)
	}
}

func TestList_FormatCSVAndTSV(t *testing.T) {
	_, root, _, _ := fixtureRoots(t)

	for _, tc := range []struct {
		name    string
		format  string
		header  string
		contain string
	}{
		{name: "csv", format: "csv", header: "session_id,created_at,updated_at,size_bytes,health,host_dir,head,path", contain: idCSV + ","},
		{name: "tsv", format: "tsv", header: "session_id\tcreated_at\tupdated_at\tsize_bytes\thealth\thost_dir\thead\tpath", contain: idCSV + "\t"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newIsolatedRootCmd(t, root)
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			cmd.SetOut(stdout)
			cmd.SetErr(stderr)
			cmd.SetArgs([]string{"list", "--sessions-root", root, "--format", tc.format, "--id", idCSV, "--limit", "1"})
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

func TestList_HostDirDisplayHomeAndNonHome(t *testing.T) {
	_, root, _, _ := fixtureRoots(t)
	t.Setenv("HOME", "/tmp/home-sim")

	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"list", "--sessions-root", root, "--limit", "0", "--color", "never"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list execute: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "HOST") {
		t.Fatalf("expected HOST column in output: %q", out)
	}
	if !strings.Contains(out, idHomeHost[:12]) || !strings.Contains(out, "~/work/a") {
		t.Fatalf("expected home host path compacted to ~: %q", out)
	}
	if !strings.Contains(out, idNonHome[:12]) || !strings.Contains(out, "/var/tmp/proj") {
		t.Fatalf("expected non-home host path kept full: %q", out)
	}
}

func TestList_ComposableContainsFilters(t *testing.T) {
	_, root, _, _ := fixtureRoots(t)

	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"list",
		"--sessions-root", root,
		"--format", "csv",
		"--column", "session_id",
		"--no-header",
		"--host-contains", "/var/tmp",
		"--path-contains", "rollout-non-home-host",
		"--head-contains", "NON HOME",
		"--limit", "0",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list execute: %v", err)
	}

	out := strings.TrimSpace(stdout.String())
	if out != idNonHome {
		t.Fatalf("expected only %s, got %q", idNonHome, out)
	}
}

func TestList_ShortFlags(t *testing.T) {
	_, root, _, _ := fixtureRoots(t)

	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"list",
		"--sessions-root", root,
		"-f", "csv",
		"--column", "session_id",
		"--no-header",
		"-i", idCSV,
		"-l", "1",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list short flags execute: %v", err)
	}

	out := strings.TrimSpace(stdout.String())
	if out != idCSV {
		t.Fatalf("expected %s, got %q", idCSV, out)
	}
}

func TestList_LongFlags(t *testing.T) {
	_, root, _, _ := fixtureRoots(t)

	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"list",
		"--sessions-root", root,
		"--format", "csv",
		"--column", "session_id",
		"--no-header",
		"--id", idCSV,
		"--limit", "1",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list long flags execute: %v", err)
	}

	out := strings.TrimSpace(stdout.String())
	if out != idCSV {
		t.Fatalf("expected %s, got %q", idCSV, out)
	}
}

func TestList_MixedShortAndLongFlags(t *testing.T) {
	_, root, _, _ := fixtureRoots(t)

	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"list",
		"--sessions-root", root,
		"-f", "csv",
		"--column", "session_id",
		"--no-header",
		"--id-prefix", "99999999-9999-9999-9999",
		"-l", "1",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list mixed flags execute: %v", err)
	}

	out := strings.TrimSpace(stdout.String())
	if out != idCSV {
		t.Fatalf("expected %s, got %q", idCSV, out)
	}
}

func TestList_FlagConflict_LastValueWins(t *testing.T) {
	_, root, _, _ := fixtureRoots(t)

	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"list",
		"--sessions-root", root,
		"--format", "csv",
		"--column", "session_id",
		"--no-header",
		"--id", idCSV,
		"-i", idColumns,
		"--limit", "1",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list conflict execute: %v", err)
	}

	out := strings.TrimSpace(stdout.String())
	if out != idColumns {
		t.Fatalf("expected last -i value %s, got %q", idColumns, out)
	}
}

func TestList_ShortFlagHHelpAndHealth(t *testing.T) {
	_, root, _, _ := fixtureRoots(t)

	helpCmd := newIsolatedRootCmd(t, root)
	helpOut := &bytes.Buffer{}
	helpErr := &bytes.Buffer{}
	helpCmd.SetOut(helpOut)
	helpCmd.SetErr(helpErr)
	helpCmd.SetArgs([]string{"list", "-h"})
	if err := helpCmd.Execute(); err != nil {
		t.Fatalf("list -h execute: %v", err)
	}
	if !strings.Contains(helpOut.String(), "Usage:") {
		t.Fatalf("expected help usage output, got: %q", helpOut.String())
	}

	healthCmd := newIsolatedRootCmd(t, root)
	healthOut := &bytes.Buffer{}
	healthErr := &bytes.Buffer{}
	healthCmd.SetOut(healthOut)
	healthCmd.SetErr(healthErr)
	healthCmd.SetArgs([]string{
		"list",
		"--sessions-root", root,
		"-H", "corrupted",
		"-f", "csv",
		"--column", "health",
		"--no-header",
		"-l", "1",
	})
	if err := healthCmd.Execute(); err != nil {
		t.Fatalf("list -H execute: %v", err)
	}
	if !strings.Contains(healthOut.String(), "CORRUPTED") {
		t.Fatalf("expected corrupted row with -H, got: %q", healthOut.String())
	}
}

func TestList_NoHeaderAndColumn(t *testing.T) {
	_, root, _, _ := fixtureRoots(t)

	cmd := newIsolatedRootCmd(t, root)
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
		"--id", idColumns,
		"--limit", "1",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list execute: %v", err)
	}
	out := strings.TrimSpace(stdout.String())
	if strings.Contains(out, "session_id,health") {
		t.Fatalf("unexpected header in output: %q", out)
	}
	if !strings.Contains(out, idColumns+",OK") {
		t.Fatalf("unexpected row output: %q", out)
	}
}

func TestList_HeadWidth(t *testing.T) {
	_, root, _, _ := fixtureRoots(t)

	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"list",
		"--sessions-root", root,
		"--id", idHeadWidth,
		"--limit", "1",
		"--head-width", "10",
		"--color", "never",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list execute: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, idHeadWidth[:12]) || !strings.Contains(out, "this is a ...") {
		t.Fatalf("expected truncated head in output: %q", out)
	}
}

func TestList_SortAndOrder(t *testing.T) {
	_, root, _, _ := fixtureRoots(t)

	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"list",
		"--sessions-root", root,
		"--format", "csv",
		"--column", "session_id",
		"--id-prefix", "00000000-0000-0000-0000-0000000000",
		"--sort", "session_id",
		"--order", "asc",
		"--limit", "1",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list execute: %v", err)
	}
	out := strings.TrimSpace(stdout.String())
	if !strings.Contains(out, "00000000-0000-0000-0000-000000000000") {
		t.Fatalf("expected ascending first id, got: %q", out)
	}
}

func TestList_UsesFixtureDefaultSessionsRoot(t *testing.T) {
	_, root, _, _ := fixtureRoots(t)

	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"list", "--limit", "0", "--color", "never"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list execute: %v", err)
	}
	if !strings.Contains(stdout.String(), idDefault[:12]) {
		t.Fatalf("expected fixture data in output: %q", stdout.String())
	}
}

func TestList_UsesConfigSessionsRootWhenFlagMissing(t *testing.T) {
	workspace, root, _, _ := fixtureRoots(t)
	cfgPath := filepath.Join(workspace, "config.json")
	cfg := []byte(`{"sessions_root":"` + root + `"}`)
	if err := os.WriteFile(cfgPath, cfg, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("CSM_CONFIG", cfgPath)
	t.Setenv("SESSIONS_ROOT", "")

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"list", "--limit", "1", "--color", "never"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list execute: %v", err)
	}
	if !strings.Contains(stdout.String(), "showing 1 of") {
		t.Fatalf("unexpected list output: %q", stdout.String())
	}
}

func TestDoctor_StrictFailsOnWarnings(t *testing.T) {
	_, root, _, _ := fixtureRoots(t)
	t.Setenv("SESSIONS_ROOT", root)

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"doctor", "--strict"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected strict doctor failure")
	}
}

func TestDelete_DryRunWritesAuditAndKeepsFile(t *testing.T) {
	workspace, root, _, logFile := fixtureRoots(t)
	sessionPath := filepath.Join(workspace, "sessions", "2026", "03", "02", "rollout-delete-dry.jsonl")

	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"delete", "--sessions-root", root, "--id", idDeleteDry, "--dry-run", "--log-file", logFile})

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
	workspace, root, _, logFile := fixtureRoots(t)
	p := filepath.Join(workspace, "sessions", "2026", "03", "02", "rollout-delete-confirm.jsonl")

	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"delete", "--sessions-root", root, "--id", idDeleteNeed, "--dry-run=false", "--interactive-confirm=false", "--log-file", logFile})

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
	workspace, root, _, logFile := fixtureRoots(t)
	p1 := filepath.Join(workspace, "sessions", "2026", "03", "02", "rollout-delete-prefix-1.jsonl")
	p2 := filepath.Join(workspace, "sessions", "2026", "03", "02", "rollout-delete-prefix-2.jsonl")

	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
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
	workspace, root, trashRoot, logFile := fixtureRoots(t)
	filename := "rollout-delete-soft.jsonl"
	src := filepath.Join(workspace, "sessions", "2026", "03", "02", filename)

	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"delete",
		"--sessions-root", root,
		"--trash-root", trashRoot,
		"--log-file", logFile,
		"--id", idSoftDelete,
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
	if !strings.Contains(out, "batch_id=") {
		t.Fatalf("expected batch_id in output: %s", out)
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	line := strings.TrimSpace(string(data))
	var rec map[string]any
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		t.Fatalf("invalid audit json: %v", err)
	}
	if strings.TrimSpace(fmt.Sprint(rec["batch_id"])) == "" {
		t.Fatalf("expected batch_id in audit record: %#v", rec)
	}
}

func TestDelete_RealHardDeleteRemovesFile(t *testing.T) {
	workspace, root, _, logFile := fixtureRoots(t)
	src := filepath.Join(workspace, "sessions", "2026", "03", "02", "rollout-delete-hard.jsonl")

	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"delete",
		"--sessions-root", root,
		"--id", idHardDelete,
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
	workspace, root, _, logFile := fixtureRoots(t)
	p1 := filepath.Join(workspace, "sessions", "2026", "03", "02", "rollout-batch-1.jsonl")
	p2 := filepath.Join(workspace, "sessions", "2026", "03", "02", "rollout-batch-2.jsonl")
	p3 := filepath.Join(workspace, "sessions", "2026", "03", "02", "rollout-batch-3.jsonl")

	cmd := newIsolatedRootCmd(t, root)
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
	_, root, _, logFile := fixtureRoots(t)
	cmd := newIsolatedRootCmd(t, root)
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
	_, root, trashRoot, logFile := fixtureRoots(t)

	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"delete",
		"--sessions-root", root,
		"--trash-root", trashRoot,
		"--id", idPreview,
		"--dry-run=false",
		"--confirm",
		"--interactive-confirm=false",
		"--yes",
		"--log-file", logFile,
		"--preview", "sample",
		"--preview-limit", "1",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("delete execute: %v", err)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "preview action=soft-delete matched=1") {
		t.Fatalf("expected preview output, got: %q", errOut)
	}
	if !strings.Contains(errOut, "mode=sample") {
		t.Fatalf("expected sample mode output, got: %q", errOut)
	}
}

func TestDelete_RealDeletePreviewNoneSuppressesOutput(t *testing.T) {
	_, root, trashRoot, logFile := fixtureRoots(t)

	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"delete",
		"--sessions-root", root,
		"--trash-root", trashRoot,
		"--id", idPreview,
		"--dry-run=false",
		"--confirm",
		"--interactive-confirm=false",
		"--yes",
		"--log-file", logFile,
		"--preview", "none",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("delete execute: %v", err)
	}
	if strings.Contains(stderr.String(), "preview action=") {
		t.Fatalf("expected no preview output, got: %q", stderr.String())
	}
}

func TestDelete_RealDeletePreviewFullShowsAll(t *testing.T) {
	_, root, trashRoot, logFile := fixtureRoots(t)

	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"delete",
		"--sessions-root", root,
		"--trash-root", trashRoot,
		"--id-prefix", "33333333-3333-3333-3333",
		"--dry-run=false",
		"--confirm",
		"--interactive-confirm=false",
		"--yes",
		"--log-file", logFile,
		"--preview", "full",
		"--preview-limit", "1",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("delete execute: %v", err)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "mode=full") {
		t.Fatalf("expected full mode output, got: %q", errOut)
	}
	if strings.Contains(errOut, "... and") {
		t.Fatalf("full mode should not truncate preview, got: %q", errOut)
	}
	if !strings.Contains(errOut, shortID("33333333-3333-3333-3333-333333333333")) || !strings.Contains(errOut, shortID("33333333-3333-3333-3333-aaaaaaaaaaaa")) {
		t.Fatalf("full mode should show all matched IDs, got: %q", errOut)
	}
}

func TestDelete_InvalidPreviewMode(t *testing.T) {
	_, root, trashRoot, logFile := fixtureRoots(t)

	cmd := newIsolatedRootCmd(t, root)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"delete",
		"--sessions-root", root,
		"--trash-root", trashRoot,
		"--id", idPreview,
		"--dry-run=false",
		"--confirm",
		"--yes",
		"--interactive-confirm=false",
		"--log-file", logFile,
		"--preview", "bad",
	})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected invalid preview mode error")
	}
	if !strings.Contains(err.Error(), "invalid --preview") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRestore_DryRunDoesNotMoveFile(t *testing.T) {
	workspace, root, trashRoot, logFile := fixtureRoots(t)
	trashed := filepath.Join(workspace, "trash", "sessions", "2026", "03", "02", "rollout-r1.jsonl")

	cmd := newIsolatedRootCmd(t, root)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"restore",
		"--sessions-root", root,
		"--trash-root", trashRoot,
		"--id", idRestoreR1,
		"--dry-run",
		"--log-file", logFile,
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("restore execute: %v", err)
	}
	if _, err := os.Stat(trashed); err != nil {
		t.Fatalf("trashed file should remain in dry-run: %v", err)
	}
	dst := filepath.Join(workspace, "sessions", "2026", "03", "02", "rollout-r1.jsonl")
	if _, err := os.Stat(dst); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("destination should not exist in dry-run: %v", err)
	}
}

func TestRestore_RealMovesFileBack(t *testing.T) {
	workspace, root, trashRoot, logFile := fixtureRoots(t)
	trashed := filepath.Join(workspace, "trash", "sessions", "2026", "03", "02", "rollout-r2.jsonl")

	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"restore",
		"--sessions-root", root,
		"--trash-root", trashRoot,
		"--id", idRestoreR2,
		"--dry-run=false",
		"--confirm",
		"--interactive-confirm=false",
		"--yes",
		"--log-file", logFile,
		"--preview", "sample",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("restore execute: %v", err)
	}
	if !strings.Contains(stderr.String(), "preview action=restore matched=1") {
		t.Fatalf("expected restore preview output, got: %q", stderr.String())
	}
	if _, err := os.Stat(trashed); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("trashed file should be moved out: %v", err)
	}
	dst := filepath.Join(workspace, "sessions", "2026", "03", "02", "rollout-r2.jsonl")
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("destination should exist after restore: %v", err)
	}
}

func TestRestore_ByBatchIDRollsBackSoftDelete(t *testing.T) {
	workspace, root, trashRoot, logFile := fixtureRoots(t)
	filename := "rollout-delete-soft.jsonl"
	src := filepath.Join(workspace, "sessions", "2026", "03", "02", filename)
	trashed := filepath.Join(trashRoot, "sessions", "2026", "03", "02", filename)

	delCmd := newIsolatedRootCmd(t, root)
	delCmd.SetOut(&bytes.Buffer{})
	delCmd.SetErr(&bytes.Buffer{})
	delCmd.SetArgs([]string{
		"delete",
		"--sessions-root", root,
		"--trash-root", trashRoot,
		"--log-file", logFile,
		"--id", idSoftDelete,
		"--dry-run=false",
		"--confirm",
		"--interactive-confirm=false",
		"--yes",
	})
	if err := delCmd.Execute(); err != nil {
		t.Fatalf("delete execute: %v", err)
	}
	if _, err := os.Stat(trashed); err != nil {
		t.Fatalf("expected trashed file before rollback: %v", err)
	}

	logData, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(logData)), "\n")
	if len(lines) == 0 {
		t.Fatal("expected at least one log line")
	}
	var rec map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &rec); err != nil {
		t.Fatalf("invalid audit json: %v", err)
	}
	batchID := strings.TrimSpace(fmt.Sprint(rec["batch_id"]))
	if batchID == "" {
		t.Fatalf("missing batch_id in delete log: %#v", rec)
	}

	restoreCmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	restoreCmd.SetOut(stdout)
	restoreCmd.SetErr(stderr)
	restoreCmd.SetArgs([]string{
		"restore",
		"--sessions-root", root,
		"--trash-root", trashRoot,
		"--log-file", logFile,
		"--batch-id", batchID,
		"--dry-run=false",
		"--confirm",
		"--interactive-confirm=false",
		"--yes",
		"--preview", "none",
	})
	if err := restoreCmd.Execute(); err != nil {
		t.Fatalf("restore execute: %v", err)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("expected restored file at source path: %v", err)
	}
	if _, err := os.Stat(trashed); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("trashed file should be removed after rollback: %v", err)
	}
	if !strings.Contains(stdout.String(), "rollback_from_batch_id="+batchID) {
		t.Fatalf("expected rollback marker in output: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "batch_id=") {
		t.Fatalf("expected new restore batch_id in output: %q", stdout.String())
	}
}

func TestRestore_BatchIDNotFound(t *testing.T) {
	_, root, trashRoot, logFile := fixtureRoots(t)

	cmd := newIsolatedRootCmd(t, root)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"restore",
		"--sessions-root", root,
		"--trash-root", trashRoot,
		"--log-file", logFile,
		"--batch-id", "b-not-found",
	})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected missing batch_id error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRestore_BatchIDConflictsWithSelectors(t *testing.T) {
	_, root, trashRoot, logFile := fixtureRoots(t)

	cmd := newIsolatedRootCmd(t, root)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"restore",
		"--sessions-root", root,
		"--trash-root", trashRoot,
		"--log-file", logFile,
		"--batch-id", "b-abc",
		"--id", idRestoreR2,
	})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected batch-id selector conflict")
	}
	if !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRestore_RealPreviewNoneSuppressesOutput(t *testing.T) {
	_, root, trashRoot, logFile := fixtureRoots(t)

	cmd := newIsolatedRootCmd(t, root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"restore",
		"--sessions-root", root,
		"--trash-root", trashRoot,
		"--id", idRestoreR2,
		"--dry-run=false",
		"--confirm",
		"--interactive-confirm=false",
		"--yes",
		"--log-file", logFile,
		"--preview", "none",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("restore execute: %v", err)
	}
	if strings.Contains(stderr.String(), "preview action=") {
		t.Fatalf("expected no restore preview output, got: %q", stderr.String())
	}
}
