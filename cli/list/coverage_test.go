package list

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/session"
)

func TestRenderTableIncludesHeaderAndFooter(t *testing.T) {
	items := []session.Session{
		{
			SessionID: "a-1",
			UpdatedAt: time.Now(),
			Health:    session.HealthOK,
			Path:      "/tmp/a.jsonl",
			HostDir:   "/tmp",
		},
	}

	text, err := RenderTable(items, 1, RenderOptions{
		NoHeader:  false,
		ColorMode: "never",
		Out:       &bytes.Buffer{},
		Columns:   []Column{{Key: "id", Header: "ID"}, {Key: "health", Header: "HEALTH"}},
		HeadWidth: 24,
	})
	if err != nil {
		t.Fatalf("render table: %v", err)
	}

	if !strings.Contains(text, "ID") {
		t.Fatalf("expected header in output: %q", text)
	}

	if !strings.Contains(text, "showing 1 of 1") {
		t.Fatalf("expected footer in output: %q", text)
	}
}

func TestColorizeRenderedTableReturnsStyledText(t *testing.T) {
	input := "ID\nrow\nshowing 1 of 1\n"
	out := ColorizeRenderedTable(input, []session.Session{{Health: session.HealthCorrupted}}, false, true)
	if out == "" {
		t.Fatal("expected colored output")
	}
}

func TestWriteWithPagerDisabledWritesContent(t *testing.T) {
	var out bytes.Buffer
	content := "alpha\nbeta\n"

	if err := WriteWithPager(&out, content, false, 10, true); err != nil {
		t.Fatalf("write with pager: %v", err)
	}

	if out.String() != content {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestWriteWithPagerInteractiveAll(t *testing.T) {
	devNull, err := os.OpenFile("/dev/null", os.O_WRONLY, 0)
	if err != nil {
		t.Skipf("open /dev/null: %v", err)
	}
	defer devNull.Close()

	// /dev/null must appear as a char device for pager path.
	if fi, err := devNull.Stat(); err != nil || (fi.Mode()&os.ModeCharDevice) == 0 {
		t.Skip("terminal-like char device unavailable")
	}

	root := t.TempDir()
	inPath := filepath.Join(root, "pager-input.txt")
	if err := os.WriteFile(inPath, []byte("a\n"), 0o644); err != nil {
		t.Fatalf("write pager input: %v", err)
	}

	in, err := os.Open(inPath)
	if err != nil {
		t.Fatalf("open pager input: %v", err)
	}
	defer in.Close()

	oldStdin := os.Stdin
	os.Stdin = in
	defer func() { os.Stdin = oldStdin }()

	text := strings.Join([]string{
		"ID\tHEALTH",
		"row1",
		"row2",
		"row3",
		"showing 3 of 3",
		"",
	}, "\n")

	if err := WriteWithPager(devNull, text, true, 1, true); err != nil {
		t.Fatalf("write with pager interactive all: %v", err)
	}
}
