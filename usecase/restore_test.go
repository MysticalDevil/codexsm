package usecase

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/session"
)

func TestSelectRestoreSessions(t *testing.T) {
	trashSessionsRoot := t.TempDir()
	writeSessionFixture(t, trashSessionsRoot, "a-1", "/tmp/a")
	writeSessionFixture(t, trashSessionsRoot, "b-2", "/tmp/b")

	_, err := SelectRestoreSessions(RestoreSelectInput{
		TrashSessionsRoot: trashSessionsRoot,
		Selector:          session.Selector{},
		BatchID:           "b-1",
		LogFile:           "/tmp/log",
		IDsForBatch: func(_ string, _ string) ([]string, error) {
			return []string{"a-1"}, nil
		},
		Now: time.Now(),
	})
	if err != nil {
		t.Fatalf("batch-id restore candidates: %v", err)
	}

	_, err = SelectRestoreSessions(RestoreSelectInput{
		TrashSessionsRoot: trashSessionsRoot,
		Selector: session.Selector{
			ID: "a-1",
		},
		BatchID: "b-1",
		LogFile: "/tmp/log",
		IDsForBatch: func(_ string, _ string) ([]string, error) {
			return []string{"a-1"}, nil
		},
		Now: time.Now(),
	})
	if err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("expected conflict error, got: %v", err)
	}

	_, err = SelectRestoreSessions(RestoreSelectInput{
		TrashSessionsRoot: trashSessionsRoot,
		Selector:          session.Selector{},
		BatchID:           "",
		Now:               time.Now(),
	})
	if err == nil || !strings.Contains(err.Error(), "requires at least one selector") {
		t.Fatalf("expected missing selector error, got: %v", err)
	}
}

func writeSessionFixture(t *testing.T, sessionsRoot, id, host string) {
	t.Helper()

	dir := filepath.Join(sessionsRoot, "2026", "03", "08")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir sessions fixture: %v", err)
	}

	path := filepath.Join(dir, id+".jsonl")

	line := fmt.Sprintf(
		`{"type":"session_meta","payload":{"id":"%s","cwd":"%s","timestamp":"%s"}}`+"\n",
		id,
		host,
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err := os.WriteFile(path, []byte(line), 0o644); err != nil {
		t.Fatalf("write session fixture: %v", err)
	}
}
