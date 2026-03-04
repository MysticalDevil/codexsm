package session

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
)

func TestDeleteSessionsDryRunNoSideEffects(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	sessionsRoot := filepath.Join(workspace, "sessions")
	trashRoot := filepath.Join(workspace, "trash")
	f := filepath.Join(sessionsRoot, "2026", "03", "02", "rollout-delete-dry.jsonl")

	candidates := []Session{{SessionID: "11111111-1111-1111-1111-111111111111", Path: f, SizeBytes: 1, UpdatedAt: time.Now()}}
	sel := Selector{ID: "11111111-1111-1111-1111-111111111111"}
	sum, err := DeleteSessions(candidates, sel, DeleteOptions{DryRun: true, SessionsRoot: sessionsRoot, TrashRoot: trashRoot})
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
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	sessionsRoot := filepath.Join(workspace, "sessions")
	trashRoot := filepath.Join(workspace, "trash")
	src := filepath.Join(sessionsRoot, "2026", "03", "02", "rollout-delete-soft.jsonl")

	candidates := []Session{{SessionID: "44444444-4444-4444-4444-444444444444", Path: src, SizeBytes: 1, UpdatedAt: time.Now()}}
	sel := Selector{ID: "44444444-4444-4444-4444-444444444444"}
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
