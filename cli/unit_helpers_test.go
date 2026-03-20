package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	del "github.com/MysticalDevil/codexsm/cli/delete"
	"github.com/MysticalDevil/codexsm/cli/list"
	"github.com/MysticalDevil/codexsm/cli/restore"
	cliutil "github.com/MysticalDevil/codexsm/cli/util"
	"github.com/MysticalDevil/codexsm/internal/core"
	"github.com/MysticalDevil/codexsm/internal/ops"
	"github.com/MysticalDevil/codexsm/internal/testsupport"
	"github.com/MysticalDevil/codexsm/session"
	"github.com/MysticalDevil/codexsm/util"
	"github.com/spf13/cobra"
)

func TestListHelpers(t *testing.T) {
	cols, err := list.ParseColumns("", false, "table")
	if err != nil {
		t.Fatalf("parseListColumns default table: %v", err)
	}

	if len(cols) == 0 || cols[0].Key != "id" {
		t.Fatalf("unexpected default columns: %+v", cols)
	}

	if _, err := list.ParseColumns("bad", false, "table"); err == nil {
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
	if got := list.ColumnValue("host", s, home, 8, true); got != "~/work" {
		t.Fatalf("unexpected host value: %q", got)
	}

	if got := list.ColumnValue("head", s, home, 8, true); got != "this is ..." {
		t.Fatalf("unexpected truncated head: %q", got)
	}

	if got := list.TruncateDisplayText("abc", 0); got != "abc" {
		t.Fatalf("truncate with 0 should keep full text, got: %q", got)
	}

	if got := core.ShortID(s.SessionID); got != "11111111-111" {
		t.Fatalf("unexpected short id: %q", got)
	}

	if got := core.FormatBytesIEC(1536); got != "1.5KiB" {
		t.Fatalf("unexpected size: %q", got)
	}

	if !list.HasHealthColumn([]list.Column{{Key: "id"}, {Key: "health"}}) {
		t.Fatal("expected hasHealthColumn=true")
	}

	if cliutil.StripANSI("\x1b[31mred\x1b[0m") != "red" {
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
	cols := []list.Column{{Key: "id", Header: "ID"}, {Key: "health", Header: "HEALTH"}, {Key: "host", Header: "HOST"}}

	out, err := list.RenderTable(sessions, 1, list.RenderOptions{
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
	if err := list.WriteDelimited(buf, sessions, ',', false, cols); err != nil {
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

	if _, err := cliutil.BuildSelector("", "", "", "", "", "bad", ""); err == nil {
		t.Fatal("expected older-than parse error")
	}

	if _, err := cliutil.BuildSelector("", "", "", "", "", "", "bad"); err == nil {
		t.Fatal("expected health parse error")
	}

	sel, err := cliutil.BuildSelector("id", "pre", " host ", " /sessions ", " fixture ", "1h", "ok")
	if err != nil {
		t.Fatalf("buildSelector valid: %v", err)
	}

	if sel.ID != "id" || sel.IDPrefix != "pre" || sel.HostContains != "host" || sel.PathContains != "/sessions" || sel.HeadContains != "fixture" || !sel.HasOlderThan || !sel.HasHealth {
		t.Fatalf("unexpected selector: %+v", sel)
	}

	if _, err := cliutil.ParseHealth("bad"); err == nil {
		t.Fatal("expected parseHealth error")
	}

	if got, err := ops.ParsePreviewMode("sample"); err != nil || got != ops.PreviewSample {
		t.Fatalf("ParsePreviewMode(sample) got=%q err=%v", got, err)
	}

	if got, err := ops.ParsePreviewMode("full"); err != nil || got != ops.PreviewFull {
		t.Fatalf("ParsePreviewMode(full) got=%q err=%v", got, err)
	}

	if got, err := ops.ParsePreviewMode("none"); err != nil || got != ops.PreviewNone {
		t.Fatalf("ParsePreviewMode(none) got=%q err=%v", got, err)
	}

	if _, err := ops.ParsePreviewMode("bad"); err == nil {
		t.Fatal("expected ParsePreviewMode error")
	}
}

func TestErrorAndLoggingHelpers(t *testing.T) {
	base := errors.New("x")
	wrapped := cliutil.WithExitCode(base, 9)

	var ex *cliutil.ExitError
	if !errors.As(wrapped, &ex) {
		t.Fatal("expected ExitError")
	}

	if ex.ExitCode() != 9 {
		t.Fatalf("unexpected exit code: %d", ex.ExitCode())
	}

	if cliutil.WithExitCode(nil, 1) != nil {
		t.Fatal("WithExitCode(nil) should be nil")
	}

	if (&cliutil.ExitError{}).ExitCode() != 1 {
		t.Fatal("zero ExitError should default to code 1")
	}

	if (&cliutil.ExitError{Code: -2, Err: base}).ExitCode() != 1 {
		t.Fatal("negative ExitError code should default to 1")
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

	errBuf := &bytes.Buffer{}
	cmd.SetErr(errBuf)

	if ok, err := del.InteractiveConfirmDelete(cmd, 1, false); err == nil || ok {
		t.Fatalf("expected non-interactive delete confirm error, got ok=%v err=%v", ok, err)
	}

	if ok, err := restore.InteractiveConfirmRestore(cmd, 1); err == nil || ok {
		t.Fatalf("expected non-interactive restore confirm error, got ok=%v err=%v", ok, err)
	}

	cmd.SetOut(&bytes.Buffer{})
	del.PrintDeleteSummary(cmd, session.DeleteSummary{
		Action:       "delete-dry-run",
		Simulation:   true,
		MatchedCount: 1,
		Results:      []session.DeleteResult{{SessionID: "s1", Path: "/tmp/a", Status: "simulated"}},
	})
	del.PrintDeletePreview(cmd, []session.Session{{SessionID: "s1", Path: "/tmp/a", SizeBytes: 5}}, false, ops.PreviewSample, 1)
	restore.PrintRestorePreview(cmd, []session.Session{{SessionID: "s1", Path: "/tmp/a", SizeBytes: 5}}, ops.PreviewSample, 1)
	restore.PrintRestoreSummary(cmd, session.RestoreSummary{
		Action:       "restore-dry-run",
		Simulation:   true,
		MatchedCount: 1,
		Results:      []session.DeleteResult{{SessionID: "s1", Path: "/tmp/a", Status: "simulated"}},
	})
}

func TestDeleteRestorePreviewHelpersEdgeCases(t *testing.T) {
	items := []session.Session{
		{SessionID: "s1", Path: "/tmp/a", SizeBytes: 5},
		{SessionID: "s2", Path: "/tmp/b", SizeBytes: 7},
	}

	t.Run("delete preview full ignores negative limit", func(t *testing.T) {
		cmd := &cobra.Command{}
		errBuf := &bytes.Buffer{}
		cmd.SetErr(errBuf)
		del.PrintDeletePreview(cmd, items, true, ops.PreviewFull, -1)

		got := errBuf.String()
		if !strings.Contains(got, "preview action=hard-delete matched=2") {
			t.Fatalf("unexpected delete preview header: %q", got)
		}

		if !strings.Contains(got, core.ShortID("s1")) || !strings.Contains(got, core.ShortID("s2")) {
			t.Fatalf("expected all preview rows in full mode, got: %q", got)
		}
	})

	t.Run("delete preview sample clamps negative limit", func(t *testing.T) {
		cmd := &cobra.Command{}
		errBuf := &bytes.Buffer{}
		cmd.SetErr(errBuf)
		del.PrintDeletePreview(cmd, items, false, ops.PreviewSample, -3)

		got := errBuf.String()
		if strings.Contains(got, core.ShortID("s1")) || strings.Contains(got, core.ShortID("s2")) {
			t.Fatalf("did not expect item rows with negative sample limit, got: %q", got)
		}

		if !strings.Contains(got, "... and 2 more") {
			t.Fatalf("expected remainder notice, got: %q", got)
		}
	})

	t.Run("restore preview full ignores negative limit", func(t *testing.T) {
		cmd := &cobra.Command{}
		errBuf := &bytes.Buffer{}
		cmd.SetErr(errBuf)
		restore.PrintRestorePreview(cmd, items, ops.PreviewFull, -1)

		got := errBuf.String()
		if !strings.Contains(got, "preview action=restore matched=2") {
			t.Fatalf("unexpected restore preview header: %q", got)
		}

		if !strings.Contains(got, core.ShortID("s1")) || !strings.Contains(got, core.ShortID("s2")) {
			t.Fatalf("expected all restore preview rows in full mode, got: %q", got)
		}
	})

	t.Run("restore preview sample clamps negative limit", func(t *testing.T) {
		cmd := &cobra.Command{}
		errBuf := &bytes.Buffer{}
		cmd.SetErr(errBuf)
		restore.PrintRestorePreview(cmd, items, ops.PreviewSample, -1)

		got := errBuf.String()
		if strings.Contains(got, core.ShortID("s1")) || strings.Contains(got, core.ShortID("s2")) {
			t.Fatalf("did not expect restore item rows with negative sample limit, got: %q", got)
		}

		if !strings.Contains(got, "... and 2 more") {
			t.Fatalf("expected restore remainder notice, got: %q", got)
		}
	})
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

	if err := util.MoveFile(src, dst); err != nil {
		t.Fatalf("MoveFile: %v", err)
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

	if err := util.CopyFile(src2, dst2); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}

	data2, err := os.ReadFile(dst2)
	if err != nil || string(data2) != "xyz" {
		t.Fatalf("dst2 content mismatch err=%v data=%q", err, string(data2))
	}
}

func TestAdditionalListAndResolveCoverage(t *testing.T) {
	// writeWithPager should behave as passthrough when pager is disabled.
	src := "line1\nline2\n"

	out := &bytes.Buffer{}
	if err := list.WriteWithPager(out, src, false, 10, true); err != nil {
		t.Fatalf("writeWithPager passthrough: %v", err)
	}

	if out.String() != src {
		t.Fatalf("unexpected passthrough output: %q", out.String())
	}

	// Cover colorized branch helper.
	colorized := list.ColorizeRenderedTable("H\nrow\nshowing 1 of 1\n", []session.Session{{Health: session.HealthOK}}, false, true)
	if !strings.Contains(colorized, "\x1b[") {
		t.Fatalf("expected colored table output: %q", colorized)
	}

	// resolveOrDefault should use explicit value and fallback branch.
	got, err := cliutil.ResolveOrDefault("~/x", func() (string, error) { return "/unused", nil })
	if err != nil {
		t.Fatalf("resolveOrDefault explicit: %v", err)
	}

	if !strings.HasSuffix(got, "/x") {
		t.Fatalf("unexpected explicit resolved value: %q", got)
	}

	got, err = cliutil.ResolveOrDefault("", func() (string, error) { return "/fallback", nil })
	if err != nil || got != "/fallback" {
		t.Fatalf("resolveOrDefault fallback got=%q err=%v", got, err)
	}
}
