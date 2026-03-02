package session

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDeleteSessionsDryRunNoSideEffects(t *testing.T) {
	root := t.TempDir()
	f := filepath.Join(root, "a.jsonl")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	candidates := []Session{{SessionID: "a", Path: f, SizeBytes: 1, UpdatedAt: time.Now()}}
	sel := Selector{ID: "a"}
	sum, err := DeleteSessions(candidates, sel, DeleteOptions{DryRun: true, SessionsRoot: root, TrashRoot: filepath.Join(root, "trash")})
	if err != nil {
		t.Fatalf("DeleteSessions dry-run: %v", err)
	}
	if sum.Skipped != 1 || sum.Succeeded != 0 {
		t.Fatalf("unexpected summary: %+v", sum)
	}
	if _, err := os.Stat(f); err != nil {
		t.Fatalf("file should remain after dry-run: %v", err)
	}
}

func TestDeleteSessionsSoftDelete(t *testing.T) {
	root := t.TempDir()
	sessionsRoot := filepath.Join(root, "sessions")
	trashRoot := filepath.Join(root, "trash")

	src := filepath.Join(sessionsRoot, "2026", "03", "02", "a.jsonl")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	candidates := []Session{{SessionID: "a", Path: src, SizeBytes: 1, UpdatedAt: time.Now()}}
	sel := Selector{ID: "a"}
	sum, err := DeleteSessions(candidates, sel, DeleteOptions{DryRun: false, Confirm: true, Yes: true, SessionsRoot: sessionsRoot, TrashRoot: trashRoot})
	if err != nil {
		t.Fatalf("DeleteSessions soft: %v", err)
	}
	if sum.Succeeded != 1 || sum.Failed != 0 {
		t.Fatalf("unexpected summary: %+v", sum)
	}
	if _, err := os.Stat(src); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("source should be moved, stat err=%v", err)
	}
	if len(sum.Results) != 1 || sum.Results[0].Destination == "" {
		t.Fatalf("missing destination: %+v", sum.Results)
	}
	if _, err := os.Stat(sum.Results[0].Destination); err != nil {
		t.Fatalf("destination missing: %v", err)
	}
}
