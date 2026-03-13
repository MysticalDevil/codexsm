package usecase

import (
	"time"

	"github.com/MysticalDevil/codexsm/internal/core"
	"github.com/MysticalDevil/codexsm/session"
)

// LoadTUISessionsInput describes TUI session-loading constraints.
type LoadTUISessionsInput struct {
	SessionsRoot string
	ScanLimit    int
	ViewLimit    int
	Now          time.Time
	Repository   core.SessionRepository
	Evaluator    core.RiskEvaluator
}

// LoadTUISessionsResult is the normalized TUI session set.
type LoadTUISessionsResult struct {
	Total int
	Items []session.Session
}

// LoadTUISessions loads sessions for TUI and applies risk-first ordering.
func LoadTUISessions(in LoadTUISessionsInput) (LoadTUISessionsResult, error) {
	q, err := core.QuerySessions(in.Repository, in.SessionsRoot, core.QuerySpec{
		Now: in.Now,
	})
	if err != nil {
		return LoadTUISessionsResult{}, err
	}
	items := append([]session.Session(nil), q.Items...)
	core.SortSessionsByRisk(items, in.Evaluator, nil)

	if in.ScanLimit > 0 && len(items) > in.ScanLimit {
		items = items[:in.ScanLimit]
	}
	if in.ViewLimit > 0 && len(items) > in.ViewLimit {
		items = items[:in.ViewLimit]
	}
	return LoadTUISessionsResult{
		Total: len(q.Items),
		Items: items,
	}, nil
}
