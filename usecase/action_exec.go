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
	NewBatchID      NewBatchID
	WriteActionLog  WriteActionLog
	Now             time.Time
}

type DeleteActionResult struct {
	Summary         session.DeleteSummary
	AppliedMaxBatch int
	BatchID         string
	LogError        error
}

// DeleteExecutor executes delete workflow over selected sessions.
type DeleteExecutor func(candidates []session.Session, sel session.Selector, opts session.DeleteOptions) (session.DeleteSummary, error)

// NewBatchID allocates operation batch IDs for action logging.
type NewBatchID func() (string, error)

// WriteActionLog appends one action record into the action log file.
type WriteActionLog func(logFile string, rec audit.ActionRecord) error

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
		executor = session.DeleteSessions
	}

	sum, err := executor(in.Sessions, in.Selector, session.DeleteOptions{
		DryRun:       in.DryRun,
		Confirm:      in.Confirm,
		Yes:          in.Yes,
		Hard:         in.Hard,
		MaxBatch:     applied,
		SessionsRoot: in.SessionsRoot,
		TrashRoot:    in.TrashRoot,
	})

	out := DeleteActionResult{Summary: sum, AppliedMaxBatch: applied}
	if in.NewBatchID == nil || in.WriteActionLog == nil || in.LogFile == "" {
		return out, err
	}

	if len(in.Sessions) > 0 {
		batchID, batchErr := in.NewBatchID()
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
	out.LogError = in.WriteActionLog(in.LogFile, rec)

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
	NewBatchID         NewBatchID
	WriteActionLog     WriteActionLog
	Now                time.Time
}

type RestoreActionResult struct {
	Summary         session.RestoreSummary
	AppliedMaxBatch int
	BatchID         string
	LogError        error
}

// RestoreExecutor executes restore workflow over selected sessions.
type RestoreExecutor func(candidates []session.Session, sel session.Selector, opts session.RestoreOptions) (session.RestoreSummary, error)

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
		executor = session.RestoreSessions
	}

	sum, err := executor(in.Sessions, in.Selector, session.RestoreOptions{
		DryRun:             in.DryRun,
		Confirm:            in.Confirm,
		Yes:                in.Yes,
		AllowEmptySelector: in.AllowEmptySelector,
		MaxBatch:           applied,
		SessionsRoot:       in.SessionsRoot,
		TrashSessionsRoot:  in.TrashSessionsRoot,
	})

	out := RestoreActionResult{Summary: sum, AppliedMaxBatch: applied}
	if in.NewBatchID == nil || in.WriteActionLog == nil || in.LogFile == "" {
		return out, err
	}

	if len(in.Sessions) > 0 {
		batchID, batchErr := in.NewBatchID()
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
	out.LogError = in.WriteActionLog(in.LogFile, rec)

	return out, err
}
