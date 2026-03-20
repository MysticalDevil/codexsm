package usecase

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/audit"
	"github.com/MysticalDevil/codexsm/session"
)

func TestRunDeleteAndRestoreActionApplyBatchDefaults(t *testing.T) {
	sel := session.Selector{ID: "a-1"}

	delOut, delErr := RunDeleteAction(DeleteActionInput{
		Sessions:        []session.Session{{SessionID: "a-1", Path: "/tmp/a-1.jsonl"}},
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
		Sessions:           []session.Session{{SessionID: "a-1", Path: "/tmp/trash/sessions/a-1.jsonl"}},
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

type stubAuditSink struct {
	batchID    string
	batchErr   error
	writeErr   error
	writtenLog string
	writtenRec audit.ActionRecord
}

func (s *stubAuditSink) NewBatchID() (string, error) {
	if s.batchErr != nil {
		return "", s.batchErr
	}

	if s.batchID == "" {
		return "b-test", nil
	}

	return s.batchID, nil
}

func (s *stubAuditSink) WriteActionLog(logFile string, rec audit.ActionRecord) error {
	s.writtenLog = logFile
	s.writtenRec = rec

	return s.writeErr
}

func TestRunDeleteAndRestoreActionUseInjectedExecutors(t *testing.T) {
	var delOpts session.DeleteOptions

	delExec := func(_ []session.Session, _ session.Selector, opts session.DeleteOptions) (session.DeleteSummary, error) {
		delOpts = opts
		return session.DeleteSummary{Action: "dry-run", Simulation: true}, nil
	}

	delOut, err := RunDeleteAction(DeleteActionInput{
		Sessions:        []session.Session{{SessionID: "x", Path: "/tmp/x.jsonl"}},
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

	if delOpts.MaxBatch != 77 {
		t.Fatalf("expected injected delete opts max-batch=77, got %d", delOpts.MaxBatch)
	}

	var restoreOpts session.RestoreOptions

	restoreExec := func(_ []session.Session, _ session.Selector, opts session.RestoreOptions) (session.RestoreSummary, error) {
		restoreOpts = opts
		return session.RestoreSummary{Action: "restore-dry-run", Simulation: true}, nil
	}

	restoreOut, err := RunRestoreAction(RestoreActionInput{
		Sessions:           []session.Session{{SessionID: "x", Path: "/tmp/trash/sessions/x.jsonl"}},
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

	if restoreOpts.MaxBatch != 66 {
		t.Fatalf("expected injected restore opts max-batch=66, got %d", restoreOpts.MaxBatch)
	}
}

func TestRunDeleteActionWritesAuditViaSink(t *testing.T) {
	delExec := func(_ []session.Session, _ session.Selector, _ session.DeleteOptions) (session.DeleteSummary, error) {
		return session.DeleteSummary{Action: "dry-run", Simulation: true}, nil
	}
	sink := &stubAuditSink{batchID: "b-del"}

	out, err := RunDeleteAction(DeleteActionInput{
		Sessions:        []session.Session{{SessionID: "a-1", Path: "/tmp/a-1.jsonl"}},
		Selector:        session.Selector{ID: "a-1"},
		DryRun:          true,
		Confirm:         true,
		Yes:             true,
		MaxBatch:        1,
		MaxBatchChanged: true,
		Executor:        delExec,
		LogFile:         "/tmp/actions.log",
		NewBatchID:      sink.NewBatchID,
		WriteActionLog:  sink.WriteActionLog,
		Now:             time.Date(2026, 3, 14, 1, 2, 3, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("RunDeleteAction with audit sink: %v", err)
	}

	if out.BatchID != "b-del" {
		t.Fatalf("unexpected delete batch id: %q", out.BatchID)
	}

	if out.LogError != nil {
		t.Fatalf("unexpected delete log error: %v", out.LogError)
	}

	if sink.writtenLog != "/tmp/actions.log" || sink.writtenRec.BatchID != "b-del" {
		t.Fatalf("unexpected sink write: log=%q rec=%+v", sink.writtenLog, sink.writtenRec)
	}
}

func TestRunRestoreActionPropagatesAuditWriteError(t *testing.T) {
	restoreExec := func(_ []session.Session, _ session.Selector, _ session.RestoreOptions) (session.RestoreSummary, error) {
		return session.RestoreSummary{Action: "restore-dry-run", Simulation: true}, nil
	}
	sink := &stubAuditSink{batchID: "b-res", writeErr: fmt.Errorf("write failed")}

	out, err := RunRestoreAction(RestoreActionInput{
		Sessions:           []session.Session{{SessionID: "a-1", Path: "/tmp/trash/sessions/a-1.jsonl"}},
		Selector:           session.Selector{ID: "a-1"},
		DryRun:             true,
		Confirm:            true,
		Yes:                true,
		MaxBatch:           1,
		MaxBatchChanged:    true,
		AllowEmptySelector: false,
		Executor:           restoreExec,
		LogFile:            "/tmp/actions.log",
		NewBatchID:         sink.NewBatchID,
		WriteActionLog:     sink.WriteActionLog,
		Now:                time.Date(2026, 3, 14, 2, 3, 4, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("RunRestoreAction with audit write error should keep run err nil, got %v", err)
	}

	if out.BatchID != "b-res" {
		t.Fatalf("unexpected restore batch id: %q", out.BatchID)
	}

	if out.LogError == nil || !strings.Contains(out.LogError.Error(), "write failed") {
		t.Fatalf("expected restore log error, got %v", out.LogError)
	}
}
