package list_test

import (
	"bytes"
	"encoding/json/v2"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/MysticalDevil/codexsm/cli"
	"github.com/MysticalDevil/codexsm/internal/testsupport"
	"github.com/MysticalDevil/codexsm/session"
)

func TestListOffsetAndLimit(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "sessions")

	run := func(offset int) []session.Session {
		t.Helper()

		cmd := cli.NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{
			"list",
			"--sessions-root", root,
			"--format", "json",
			"--limit", "1",
			"--offset", strconv.Itoa(offset),
		})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("list execute offset=%d: %v", offset, err)
		}

		var rows []session.Session
		if err := json.Unmarshal(stdout.Bytes(), &rows); err != nil {
			t.Fatalf("unmarshal list json offset=%d: %v output=%q", offset, err, stdout.String())
		}

		return rows
	}

	first := run(0)

	second := run(1)
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("expected one row for each offset, got first=%d second=%d", len(first), len(second))
	}

	if first[0].Path == second[0].Path {
		t.Fatalf("expected different row across offsets, got same path=%v", first[0].Path)
	}
}

func TestListOffsetNegativeReturnsError(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "sessions")
	cmd := cli.NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"list", "--sessions-root", root, "--offset", "-1"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected negative offset error")
	}
}
