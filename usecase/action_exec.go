package usecase

import (
	"github.com/MysticalDevil/codexsm/internal/deleteexec"
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
}

type DeleteActionResult struct {
	Summary         session.DeleteSummary
	AppliedMaxBatch int
}

func RunDeleteAction(in DeleteActionInput) (DeleteActionResult, error) {
	applied := EffectiveMaxBatchWithDefaults(
		in.MaxBatchChanged,
		in.MaxBatch,
		in.DryRun,
		in.RealDefault,
		in.DryRunDefault,
	)
	sum, err := deleteexec.Execute(in.Candidates, in.Selector, deleteexec.Options{
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
}

type RestoreActionResult struct {
	Summary         restoreexec.Summary
	AppliedMaxBatch int
}

func RunRestoreAction(in RestoreActionInput) (RestoreActionResult, error) {
	applied := EffectiveMaxBatchWithDefaults(
		in.MaxBatchChanged,
		in.MaxBatch,
		in.DryRun,
		in.RealDefault,
		in.DryRunDefault,
	)
	sum, err := restoreexec.Execute(in.Candidates, in.Selector, restoreexec.Options{
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
