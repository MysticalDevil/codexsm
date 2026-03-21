package del

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
	"github.com/MysticalDevil/codexsm/session"
	"github.com/MysticalDevil/codexsm/session/scanner"
)

func TestNewCommandDryRunByID(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	sessionsRoot := filepath.Join(workspace, "sessions")
	trashRoot := filepath.Join(workspace, "trash")
	logFile := filepath.Join(workspace, "logs", "actions.log")

	items, err := scanner.ScanSessions(sessionsRoot)
	if err != nil {
		t.Fatalf("scan sessions: %v", err)
	}

	var selected session.Session
	found := false
	for _, s := range items {
		if strings.TrimSpace(s.SessionID) == "" {
			continue
		}

		selected = s
		found = true
		break
	}

	if !found {
		t.Fatal("expected at least one selectable session")
	}

	cmd := NewCommand(
		func() (string, error) { return sessionsRoot, nil },
		func() (string, error) { return trashRoot, nil },
		func() (string, error) { return logFile, nil },
		nil,
		nil,
		time.Now,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--id", selected.SessionID, "--dry-run"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("delete dry-run execute: %v stderr=%q", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "action=dry-run") {
		t.Fatalf("expected dry-run summary, got: %q", stdout.String())
	}
}

func TestNewCommandRequiresSelector(t *testing.T) {
	cmd := NewCommand(
		func() (string, error) { return "/tmp/sessions", nil },
		func() (string, error) { return "/tmp/trash", nil },
		func() (string, error) { return "/tmp/actions.log", nil },
		nil,
		nil,
		time.Now,
	)

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--dry-run"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected selector validation error")
	}

	if !strings.Contains(err.Error(), "requires at least one selector") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewCommandRejectsInvalidPreviewMode(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	sessionsRoot := filepath.Join(workspace, "sessions")

	items, err := scanner.ScanSessions(sessionsRoot)
	if err != nil {
		t.Fatalf("scan sessions: %v", err)
	}

	var selected session.Session
	for _, s := range items {
		if strings.TrimSpace(s.SessionID) != "" {
			selected = s
			break
		}
	}

	if strings.TrimSpace(selected.SessionID) == "" {
		t.Fatal("expected selectable session")
	}

	cmd := NewCommand(
		func() (string, error) { return sessionsRoot, nil },
		func() (string, error) { return filepath.Join(workspace, "trash"), nil },
		func() (string, error) { return filepath.Join(workspace, "logs", "actions.log"), nil },
		nil,
		nil,
		time.Now,
	)

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--id", selected.SessionID, "--preview", "bad-mode"})

	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected invalid preview mode error")
	}

	if !strings.Contains(err.Error(), "invalid --preview") {
		t.Fatalf("unexpected error: %v", err)
	}
}
