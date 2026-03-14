package usecase

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/internal/restoreexec"
	"github.com/MysticalDevil/codexsm/internal/testsupport"
	"github.com/MysticalDevil/codexsm/session"
)

func TestSelectDeleteCandidates(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "sessions")

	_, err := SelectDeleteCandidates(DeleteCandidatesInput{
		SessionsRoot: root,
		Selector:     session.Selector{},
		Now:          time.Now(),
	})
	if err == nil || !strings.Contains(err.Error(), "requires at least one selector") {
		t.Fatalf("expected selector error, got: %v", err)
	}

	res, err := SelectDeleteCandidates(DeleteCandidatesInput{
		SessionsRoot: root,
		Selector: session.Selector{
			ID: "11111111-1111-1111-1111-111111111111",
		},
		Now: time.Now(),
	})
	if err != nil {
		t.Fatalf("SelectDeleteCandidates: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(res.Candidates))
	}
	if res.AffectedBytes <= 0 {
		t.Fatalf("expected affected bytes > 0, got %d", res.AffectedBytes)
	}
}

func TestSelectRestoreCandidates(t *testing.T) {
	trashSessionsRoot := t.TempDir()
	writeSessionFixture(t, trashSessionsRoot, "a-1", "/tmp/a")
	writeSessionFixture(t, trashSessionsRoot, "b-2", "/tmp/b")

	_, err := SelectRestoreCandidates(RestoreCandidatesInput{
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

	_, err = SelectRestoreCandidates(RestoreCandidatesInput{
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

	_, err = SelectRestoreCandidates(RestoreCandidatesInput{
		TrashSessionsRoot: trashSessionsRoot,
		Selector:          session.Selector{},
		BatchID:           "",
		Now:               time.Now(),
	})
	if err == nil || !strings.Contains(err.Error(), "requires at least one selector") {
		t.Fatalf("expected missing selector error, got: %v", err)
	}
}

func TestEffectiveMaxBatch(t *testing.T) {
	if got := EffectiveMaxBatch(false, 777, true); got != DefaultMaxBatchDryRun {
		t.Fatalf("unexpected dry-run default max-batch: %d", got)
	}
	if got := EffectiveMaxBatch(false, 777, false); got != DefaultMaxBatchReal {
		t.Fatalf("unexpected real default max-batch: %d", got)
	}
	if got := EffectiveMaxBatch(true, 123, true); got != 123 {
		t.Fatalf("expected configured max-batch override, got %d", got)
	}
}

func TestEffectiveMaxBatchWithDefaults(t *testing.T) {
	if got := EffectiveMaxBatchWithDefaults(false, 999, false, 100, 1000); got != 100 {
		t.Fatalf("expected real default, got %d", got)
	}
	if got := EffectiveMaxBatchWithDefaults(false, 999, true, 100, 1000); got != 1000 {
		t.Fatalf("expected dry-run default, got %d", got)
	}
	if got := EffectiveMaxBatchWithDefaults(true, 321, true, 100, 1000); got != 321 {
		t.Fatalf("expected explicit max-batch, got %d", got)
	}
}

func TestRunDeleteAndRestoreActionApplyBatchDefaults(t *testing.T) {
	sel := session.Selector{ID: "a-1"}
	delOut, delErr := RunDeleteAction(DeleteActionInput{
		Candidates:      []session.Session{{SessionID: "a-1", Path: "/tmp/a-1.jsonl"}},
		Selector:        sel,
		DryRun:          true,
		Confirm:         true,
		Yes:             true,
		SessionsRoot:    "/tmp/sessions",
		TrashRoot:       "/tmp/trash",
		MaxBatch:        1,
		MaxBatchChanged: false,
		RealDefault:     100,
		DryRunDefault:   1000,
	})
	if delErr != nil {
		t.Fatalf("RunDeleteAction(dry-run): %v", delErr)
	}
	if delOut.AppliedMaxBatch != 1000 {
		t.Fatalf("unexpected delete applied max batch: %d", delOut.AppliedMaxBatch)
	}

	restoreOut, restoreErr := RunRestoreAction(RestoreActionInput{
		Candidates:         []session.Session{{SessionID: "a-1", Path: "/tmp/trash/sessions/a-1.jsonl"}},
		Selector:           sel,
		DryRun:             true,
		Confirm:            true,
		Yes:                true,
		SessionsRoot:       "/tmp/sessions",
		TrashSessionsRoot:  "/tmp/trash/sessions",
		MaxBatch:           1,
		MaxBatchChanged:    false,
		RealDefault:        100,
		DryRunDefault:      1000,
		AllowEmptySelector: false,
	})
	if restoreErr != nil {
		t.Fatalf("RunRestoreAction(dry-run): %v", restoreErr)
	}
	if restoreOut.AppliedMaxBatch != 1000 {
		t.Fatalf("unexpected restore applied max batch: %d", restoreOut.AppliedMaxBatch)
	}
}

type stubDeleteExecutor struct {
	summary session.DeleteSummary
	err     error
	opts    session.DeleteOptions
}

func (s *stubDeleteExecutor) Execute(_ []session.Session, _ session.Selector, opts session.DeleteOptions) (session.DeleteSummary, error) {
	s.opts = opts
	return s.summary, s.err
}

type stubRestoreExecutor struct {
	summary restoreexec.Summary
	err     error
	opts    restoreexec.Options
}

func (s *stubRestoreExecutor) Execute(_ []session.Session, _ session.Selector, opts restoreexec.Options) (restoreexec.Summary, error) {
	s.opts = opts
	return s.summary, s.err
}

func TestRunDeleteAndRestoreActionUseInjectedExecutors(t *testing.T) {
	delExec := &stubDeleteExecutor{
		summary: session.DeleteSummary{Action: "dry-run", Simulation: true},
	}
	delOut, err := RunDeleteAction(DeleteActionInput{
		Candidates:      []session.Session{{SessionID: "x", Path: "/tmp/x.jsonl"}},
		Selector:        session.Selector{ID: "x"},
		DryRun:          true,
		Confirm:         true,
		Yes:             true,
		MaxBatch:        77,
		MaxBatchChanged: true,
		Executor:        delExec,
	})
	if err != nil {
		t.Fatalf("RunDeleteAction(injected): %v", err)
	}
	if delOut.Summary.Action != "dry-run" {
		t.Fatalf("unexpected delete summary: %+v", delOut.Summary)
	}
	if delExec.opts.MaxBatch != 77 {
		t.Fatalf("expected injected delete opts max-batch=77, got %d", delExec.opts.MaxBatch)
	}

	restoreExec := &stubRestoreExecutor{
		summary: restoreexec.Summary{Action: "restore-dry-run", Simulation: true},
	}
	restoreOut, err := RunRestoreAction(RestoreActionInput{
		Candidates:         []session.Session{{SessionID: "x", Path: "/tmp/trash/sessions/x.jsonl"}},
		Selector:           session.Selector{ID: "x"},
		DryRun:             true,
		Confirm:            true,
		Yes:                true,
		MaxBatch:           66,
		MaxBatchChanged:    true,
		AllowEmptySelector: true,
		Executor:           restoreExec,
	})
	if err != nil {
		t.Fatalf("RunRestoreAction(injected): %v", err)
	}
	if restoreOut.Summary.Action != "restore-dry-run" {
		t.Fatalf("unexpected restore summary: %+v", restoreOut.Summary)
	}
	if restoreExec.opts.MaxBatch != 66 {
		t.Fatalf("expected injected restore opts max-batch=66, got %d", restoreExec.opts.MaxBatch)
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
