package usecase

import (
	"github.com/MysticalDevil/codexsm/internal/restoreexec"
	"github.com/MysticalDevil/codexsm/session"
)

const (
	DefaultMaxBatchTUIReal   = 100
	DefaultMaxBatchTUIDryRun = 1000
)

type DeleteActionInput struct {
	Candidates      []session.Session
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
}

type DeleteActionResult struct {
	Summary         session.DeleteSummary
	AppliedMaxBatch int
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
	sum, err := executor.Execute(in.Candidates, in.Selector, session.DeleteOptions{
		DryRun:       in.DryRun,
		Confirm:      in.Confirm,
		Yes:          in.Yes,
		Hard:         in.Hard,
		MaxBatch:     applied,
		SessionsRoot: in.SessionsRoot,
		TrashRoot:    in.TrashRoot,
	})
	return DeleteActionResult{Summary: sum, AppliedMaxBatch: applied}, err
}

type RestoreActionInput struct {
	Candidates         []session.Session
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
}

// RestoreSummary is restore execution summary.
type RestoreSummary = restoreexec.Summary

type RestoreActionResult struct {
	Summary         RestoreSummary
	AppliedMaxBatch int
}

// RestoreExecutor executes restore workflow over selected sessions.
type RestoreExecutor interface {
	Execute(candidates []session.Session, sel session.Selector, opts restoreexec.Options) (restoreexec.Summary, error)
}

// SessionRestoreExecutor is the default restore executor.
type SessionRestoreExecutor struct{}

// Execute runs the restore operation.
func (SessionRestoreExecutor) Execute(candidates []session.Session, sel session.Selector, opts restoreexec.Options) (restoreexec.Summary, error) {
	return restoreexec.Execute(candidates, sel, opts)
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
	sum, err := executor.Execute(in.Candidates, in.Selector, restoreexec.Options{
		DryRun:             in.DryRun,
		Confirm:            in.Confirm,
		Yes:                in.Yes,
		AllowEmptySelector: in.AllowEmptySelector,
		MaxBatch:           applied,
		SessionsRoot:       in.SessionsRoot,
		TrashSessionsRoot:  in.TrashSessionsRoot,
	})
	return RestoreActionResult{Summary: sum, AppliedMaxBatch: applied}, err
}
