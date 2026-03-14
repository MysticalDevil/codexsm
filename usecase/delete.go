package usecase

import (
	"errors"
	"time"

	"github.com/MysticalDevil/codexsm/internal/core"
	"github.com/MysticalDevil/codexsm/session"
)

type DeleteSelectInput struct {
	SessionsRoot string
	Selector     session.Selector
	Now          time.Time
	Repository   core.SessionRepository
}

type DeleteSelectResult struct {
	Sessions      []session.Session
	AffectedBytes int64
}

func SelectDeleteSessions(in DeleteSelectInput) (DeleteSelectResult, error) {
	if !in.Selector.HasAnyFilter() {
		return DeleteSelectResult{}, errors.New("delete requires at least one selector (--id/--id-prefix/--host-contains/--path-contains/--head-contains/--older-than/--health)")
	}

	q, err := core.QuerySessions(in.Repository, in.SessionsRoot, core.QuerySpec{
		Selector: in.Selector,
		Now:      in.Now,
	})
	if err != nil {
		return DeleteSelectResult{}, err
	}

	var affected int64
	for _, s := range q.Items {
		affected += s.SizeBytes
	}

	return DeleteSelectResult{
		Sessions:      q.Items,
		AffectedBytes: affected,
	}, nil
}
