package usecase

import (
	"time"

	"github.com/MysticalDevil/codexsm/audit"
	"github.com/MysticalDevil/codexsm/session"
)

const (
	DefaultMaxBatchTUIReal   = 100
	DefaultMaxBatchTUIDryRun = 1000
)

type DeleteActionInput struct {
	Sessions        []session.Session
	Selector        session.Selector
	DryRun          bool
	Confirm         bool
	Yes             bool
	Hard            bool
	SessionsRoot    string
	TrashRoot       string
	MaxBatch        int
	MaxBatchChanged bool
	RealDefault     int
	DryRunDefault   int
	Executor        DeleteExecutor
	LogFile         string
	AuditSink       AuditSink
	Now             time.Time
}

type DeleteActionResult struct {
	Summary         session.DeleteSummary
	AppliedMaxBatch int
	BatchID         string
	LogError        error
}

// DeleteExecutor executes delete workflow over selected sessions.
type DeleteExecutor interface {
	Execute(candidates []session.Session, sel session.Selector, opts session.DeleteOptions) (session.DeleteSummary, error)
}

// SessionDeleteExecutor is the default delete executor.
type SessionDeleteExecutor struct{}

// Execute runs the session delete operation.
func (SessionDeleteExecutor) Execute(candidates []session.Session, sel session.Selector, opts session.DeleteOptions) (session.DeleteSummary, error) {
	return session.DeleteSessions(candidates, sel, opts)
}

// AuditSink appends action records and can allocate operation batch IDs.
type AuditSink interface {
	NewBatchID() (string, error)
	WriteActionLog(logFile string, rec audit.ActionRecord) error
}

func RunDeleteAction(in DeleteActionInput) (DeleteActionResult, error) {
	applied := EffectiveMaxBatchWithDefaults(
		in.MaxBatchChanged,
		in.MaxBatch,
		in.DryRun,
		in.RealDefault,
		in.DryRunDefault,
	)

	executor := in.Executor
	if executor == nil {
		executor = SessionDeleteExecutor{}
	}

	sum, err := executor.Execute(in.Sessions, in.Selector, session.DeleteOptions{
		DryRun:       in.DryRun,
		Confirm:      in.Confirm,
		Yes:          in.Yes,
		Hard:         in.Hard,
		MaxBatch:     applied,
		SessionsRoot: in.SessionsRoot,
		TrashRoot:    in.TrashRoot,
	})

	out := DeleteActionResult{Summary: sum, AppliedMaxBatch: applied}
	if in.AuditSink == nil || in.LogFile == "" {
		return out, err
	}

	if len(in.Sessions) > 0 {
		batchID, batchErr := in.AuditSink.NewBatchID()
		if batchErr != nil {
			return out, batchErr
		}

		out.BatchID = batchID
	}

	ts := in.Now
	if ts.IsZero() {
		ts = time.Now()
	}

	rec := audit.BuildActionRecord(
		out.BatchID,
		ts.UTC(),
		sum.Action,
		sum.Simulation,
		in.Selector,
		in.Sessions,
		sum.AffectedBytes,
		sum.Results,
		sum.ErrorSummary,
	)
	out.LogError = in.AuditSink.WriteActionLog(in.LogFile, rec)

	return out, err
}

type RestoreActionInput struct {
	Sessions           []session.Session
	Selector           session.Selector
	DryRun             bool
	Confirm            bool
	Yes                bool
	SessionsRoot       string
	TrashSessionsRoot  string
	MaxBatch           int
	MaxBatchChanged    bool
	RealDefault        int
	DryRunDefault      int
	AllowEmptySelector bool
	Executor           RestoreExecutor
	LogFile            string
	AuditSink          AuditSink
	Now                time.Time
}

// RestoreSummary is restore execution summary.
type RestoreSummary = session.RestoreSummary

type RestoreActionResult struct {
	Summary         RestoreSummary
	AppliedMaxBatch int
	BatchID         string
	LogError        error
}

// RestoreExecutor executes restore workflow over selected sessions.
type RestoreExecutor interface {
	Execute(candidates []session.Session, sel session.Selector, opts session.RestoreOptions) (session.RestoreSummary, error)
}

// SessionRestoreExecutor is the default restore executor.
type SessionRestoreExecutor struct{}

// Execute runs the restore operation.
func (SessionRestoreExecutor) Execute(candidates []session.Session, sel session.Selector, opts session.RestoreOptions) (session.RestoreSummary, error) {
	return session.RestoreSessions(candidates, sel, opts)
}

func RunRestoreAction(in RestoreActionInput) (RestoreActionResult, error) {
	applied := EffectiveMaxBatchWithDefaults(
		in.MaxBatchChanged,
		in.MaxBatch,
		in.DryRun,
		in.RealDefault,
		in.DryRunDefault,
	)

	executor := in.Executor
	if executor == nil {
		executor = SessionRestoreExecutor{}
	}

	sum, err := executor.Execute(in.Sessions, in.Selector, session.RestoreOptions{
		DryRun:             in.DryRun,
		Confirm:            in.Confirm,
		Yes:                in.Yes,
		AllowEmptySelector: in.AllowEmptySelector,
		MaxBatch:           applied,
		SessionsRoot:       in.SessionsRoot,
		TrashSessionsRoot:  in.TrashSessionsRoot,
	})

	out := RestoreActionResult{Summary: sum, AppliedMaxBatch: applied}
	if in.AuditSink == nil || in.LogFile == "" {
		return out, err
	}

	if len(in.Sessions) > 0 {
		batchID, batchErr := in.AuditSink.NewBatchID()
		if batchErr != nil {
			return out, batchErr
		}

		out.BatchID = batchID
	}

	ts := in.Now
	if ts.IsZero() {
		ts = time.Now()
	}

	rec := audit.BuildActionRecord(
		out.BatchID,
		ts.UTC(),
		sum.Action,
		sum.Simulation,
		in.Selector,
		in.Sessions,
		sum.AffectedBytes,
		sum.Results,
		sum.ErrorSummary,
	)
	out.LogError = in.AuditSink.WriteActionLog(in.LogFile, rec)

	return out, err
}
