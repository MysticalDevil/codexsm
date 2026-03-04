package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
	"github.com/MysticalDevil/codexsm/session"
	"github.com/spf13/cobra"
)

func TestListHelpers(t *testing.T) {
	cols, err := parseListColumns("", false, "table")
	if err != nil {
		t.Fatalf("parseListColumns default table: %v", err)
	}
	if len(cols) == 0 || cols[0].Key != "id" {
		t.Fatalf("unexpected default columns: %+v", cols)
	}
	if _, err := parseListColumns("bad", false, "table"); err == nil {
		t.Fatal("expected invalid column error")
	}

	home := "/tmp/home-sim"
	s := session.Session{
		SessionID: "11111111-1111-1111-1111-111111111111",
		CreatedAt: time.Date(2026, 3, 1, 1, 2, 3, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 2, 1, 2, 3, 0, time.UTC),
		SizeBytes: 1536,
		Path:      "/tmp/home-sim/p/a.jsonl",
		HostDir:   "/tmp/home-sim/work",
		Health:    session.HealthOK,
		Head:      "this is a long head line",
	}
	if got := listColumnValue("host", s, home, 8, true); got != "~/work" {
		t.Fatalf("unexpected host value: %q", got)
	}
	if got := listColumnValue("head", s, home, 8, true); got != "this is ..." {
		t.Fatalf("unexpected truncated head: %q", got)
	}
	if got := truncateDisplayText("abc", 0); got != "abc" {
		t.Fatalf("truncate with 0 should keep full text, got: %q", got)
	}
	if got := shortID(s.SessionID); got != "11111111-111" {
		t.Fatalf("unexpected short id: %q", got)
	}
	if got := formatBytesIEC(1536); got != "1.5KiB" {
		t.Fatalf("unexpected size: %q", got)
	}
	if !hasHealthColumn([]listColumn{{Key: "id"}, {Key: "health"}}) {
		t.Fatal("expected hasHealthColumn=true")
	}
	if stripANSI("\x1b[31mred\x1b[0m") != "red" {
		t.Fatal("stripANSI failed")
	}
}

func TestRenderAndDelimited(t *testing.T) {
	sessions := []session.Session{
		{
			SessionID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			UpdatedAt: time.Date(2026, 3, 2, 1, 2, 3, 0, time.Local),
			SizeBytes: 1024,
			Health:    session.HealthOK,
			Head:      "hello",
			HostDir:   "/var/tmp",
		},
	}
	cols := []listColumn{{Key: "id", Header: "ID"}, {Key: "health", Header: "HEALTH"}, {Key: "host", Header: "HOST"}}
	out, err := renderTable(sessions, 1, listRenderOptions{
		NoHeader:  false,
		ColorMode: "never",
		Out:       &bytes.Buffer{},
		Columns:   cols,
		HeadWidth: 10,
	})
	if err != nil {
		t.Fatalf("renderTable: %v", err)
	}
	if !strings.Contains(out, "ID") || !strings.Contains(out, "showing 1 of 1") {
		t.Fatalf("unexpected table output: %q", out)
	}

	buf := &bytes.Buffer{}
	if err := writeListDelimited(buf, sessions, ',', false, cols); err != nil {
		t.Fatalf("writeListDelimited csv: %v", err)
	}
	if !strings.Contains(buf.String(), "id,health,host") {
		t.Fatalf("missing csv header: %q", buf.String())
	}
}

func TestColorAndSelectorHelpers(t *testing.T) {
	if !shouldUseColor("always", &bytes.Buffer{}) {
		t.Fatal("always should enable color")
	}
	if shouldUseColor("never", &bytes.Buffer{}) {
		t.Fatal("never should disable color")
	}

	if _, err := buildSelector("", "", "bad", ""); err == nil {
		t.Fatal("expected older-than parse error")
	}
	if _, err := buildSelector("", "", "", "bad"); err == nil {
		t.Fatal("expected health parse error")
	}
	sel, err := buildSelector("id", "pre", "1h", "ok")
	if err != nil {
		t.Fatalf("buildSelector valid: %v", err)
	}
	if sel.ID != "id" || sel.IDPrefix != "pre" || !sel.HasOlderThan || !sel.HasHealth {
		t.Fatalf("unexpected selector: %+v", sel)
	}

	if _, err := parseHealth("bad"); err == nil {
		t.Fatal("expected parseHealth error")
	}
}

func TestErrorAndLoggingHelpers(t *testing.T) {
	base := errors.New("x")
	wrapped := WithExitCode(base, 9)
	var ex *ExitError
	if !errors.As(wrapped, &ex) {
		t.Fatal("expected ExitError")
	}
	if ex.ExitCode() != 9 {
		t.Fatalf("unexpected exit code: %d", ex.ExitCode())
	}
	if WithExitCode(nil, 1) != nil {
		t.Fatal("WithExitCode(nil) should be nil")
	}

	if _, err := parseLogLevel("bad"); err == nil {
		t.Fatal("expected parseLogLevel error")
	}
	if err := configureLogger("bad", "info", &bytes.Buffer{}); err == nil {
		t.Fatal("expected configureLogger format error")
	}
	if err := configureLogger("json", "debug", &bytes.Buffer{}); err != nil {
		t.Fatalf("configureLogger valid: %v", err)
	}
}

func TestDeleteRestoreHelperPaths(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBufferString("yes\n"))
	cmd.SetErr(&bytes.Buffer{})
	if ok, err := interactiveConfirmDelete(cmd, 1, false); err == nil || ok {
		t.Fatalf("expected non-interactive delete confirm error, got ok=%v err=%v", ok, err)
	}
	if ok, err := interactiveConfirmRestore(cmd, 1); err == nil || ok {
		t.Fatalf("expected non-interactive restore confirm error, got ok=%v err=%v", ok, err)
	}

	cmd.SetOut(&bytes.Buffer{})
	printDeleteSummary(cmd, session.DeleteSummary{
		Action:       "delete-dry-run",
		Simulation:   true,
		MatchedCount: 1,
		Results:      []session.DeleteResult{{SessionID: "s1", Path: "/tmp/a", Status: "simulated"}},
	})
	printDeletePreview(cmd, []session.Session{{SessionID: "s1", Path: "/tmp/a", SizeBytes: 5}}, false, 1)
	printRestoreSummary(cmd, restoreSummary{
		Action:       "restore-dry-run",
		Simulation:   true,
		MatchedCount: 1,
		Results:      []session.DeleteResult{{SessionID: "s1", Path: "/tmp/a", Status: "simulated"}},
	})
	if restoreActionName(true) != "restore-dry-run" || restoreActionName(false) != "restore" {
		t.Fatal("unexpected restoreActionName")
	}
}

func TestRestoreMoveFileAndCopy(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "tmp-files")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir test root: %v", err)
	}
	src := filepath.Join(root, "src.txt")
	dst := filepath.Join(root, "dst.txt")
	if err := os.WriteFile(src, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := moveFile(src, dst); err != nil {
		t.Fatalf("moveFile: %v", err)
	}
	if _, err := os.Stat(src); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("src should be moved, err=%v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil || string(data) != "abc" {
		t.Fatalf("dst content mismatch err=%v data=%q", err, string(data))
	}

	src2 := filepath.Join(root, "src2.txt")
	dst2 := filepath.Join(root, "dst2.txt")
	if err := os.WriteFile(src2, []byte("xyz"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFileForRestore(src2, dst2); err != nil {
		t.Fatalf("copyFileForRestore: %v", err)
	}
	data2, err := os.ReadFile(dst2)
	if err != nil || string(data2) != "xyz" {
		t.Fatalf("dst2 content mismatch err=%v data=%q", err, string(data2))
	}
}

func TestGroupRenderHelpers(t *testing.T) {
	stats := []groupStat{{Group: "ok", Count: 2, SizeBytes: 1024, Latest: "2026-03-02 10:00:00"}}
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
	if !groupLess(groupStat{Count: 1}, groupStat{Count: 2}, "count") {
		t.Fatal("groupLess count branch not working")
	}
}

func TestAdditionalListAndResolveCoverage(t *testing.T) {
	// writeWithPager should behave as passthrough when pager is disabled.
	src := "line1\nline2\n"
	out := &bytes.Buffer{}
	if err := writeWithPager(out, src, false, 10, true); err != nil {
		t.Fatalf("writeWithPager passthrough: %v", err)
	}
	if out.String() != src {
		t.Fatalf("unexpected passthrough output: %q", out.String())
	}

	// Cover colorized branch helper.
	colorized := colorizeRenderedTable("H\nrow\nshowing 1 of 1\n", []session.Session{{Health: session.HealthOK}}, false, true)
	if !strings.Contains(colorized, "\x1b[") {
		t.Fatalf("expected colored table output: %q", colorized)
	}

	// resolveOrDefault should use explicit value and fallback branch.
	got, err := resolveOrDefault("~/x", func() (string, error) { return "/unused", nil })
	if err != nil {
		t.Fatalf("resolveOrDefault explicit: %v", err)
	}
	if !strings.HasSuffix(got, "/x") {
		t.Fatalf("unexpected explicit resolved value: %q", got)
	}
	got, err = resolveOrDefault("", func() (string, error) { return "/fallback", nil })
	if err != nil || got != "/fallback" {
		t.Fatalf("resolveOrDefault fallback got=%q err=%v", got, err)
	}
}

func TestSortSessions(t *testing.T) {
	sessions := []session.Session{
		{
			SessionID: "b",
			UpdatedAt: time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC),
			CreatedAt: time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC),
			SizeBytes: 20,
			Health:    session.HealthCorrupted,
			Path:      "/tmp/b.jsonl",
		},
		{
			SessionID: "a",
			UpdatedAt: time.Date(2026, 3, 2, 11, 0, 0, 0, time.UTC),
			CreatedAt: time.Date(2026, 3, 2, 8, 0, 0, 0, time.UTC),
			SizeBytes: 10,
			Health:    session.HealthOK,
			Path:      "/tmp/a.jsonl",
		},
	}

	if err := sortSessions(sessions, "size", "asc"); err != nil {
		t.Fatalf("sortSessions size asc: %v", err)
	}
	if sessions[0].SessionID != "a" {
		t.Fatalf("unexpected size asc order: %+v", sessions)
	}

	if err := sortSessions(sessions, "health", "asc"); err != nil {
		t.Fatalf("sortSessions health asc: %v", err)
	}
	if sessions[0].Health != session.HealthOK {
		t.Fatalf("unexpected health asc order: %+v", sessions)
	}

	if err := sortSessions(sessions, "updated_at", "desc"); err != nil {
		t.Fatalf("sortSessions updated_at desc: %v", err)
	}
	if sessions[0].UpdatedAt.Before(sessions[1].UpdatedAt) {
		t.Fatalf("unexpected updated_at desc order: %+v", sessions)
	}

	if err := sortSessions(sessions, "invalid", "asc"); err == nil {
		t.Fatal("expected invalid sort error")
	}
	if err := sortSessions(sessions, "size", "invalid"); err == nil {
		t.Fatal("expected invalid order error")
	}
}
