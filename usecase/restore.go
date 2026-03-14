package usecase

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MysticalDevil/codexsm/audit"
	"github.com/MysticalDevil/codexsm/internal/core"
	"github.com/MysticalDevil/codexsm/session"
)

type RestoreSelectInput struct {
	TrashSessionsRoot string
	Selector          session.Selector
	BatchID           string
	LogFile           string
	Now               time.Time
	Repository        core.SessionRepository
	IDsForBatch       func(logFile, batchID string) ([]string, error)
}

type RestoreSelectResult struct {
	Sessions      []session.Session
	AffectedBytes int64
}

func SelectRestoreSessions(in RestoreSelectInput) (RestoreSelectResult, error) {
	batchID := strings.TrimSpace(in.BatchID)
	if batchID != "" && in.Selector.HasAnyFilter() {
		return RestoreSelectResult{}, fmt.Errorf("restore --batch-id cannot be combined with selector flags")
	}

	if batchID != "" {
		loadIDs := in.IDsForBatch
		if loadIDs == nil {
			loadIDs = audit.SessionIDsForBatchRollback
		}

		ids, err := loadIDs(in.LogFile, batchID)
		if err != nil {
			return RestoreSelectResult{}, err
		}

		idSet := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			idSet[id] = struct{}{}
		}

		q, err := core.QuerySessions(in.Repository, in.TrashSessionsRoot, core.QuerySpec{
			Now: in.Now,
		})
		if err != nil {
			return RestoreSelectResult{}, err
		}

		candidates := make([]session.Session, 0, len(q.Items))

		var affected int64

		for _, s := range q.Items {
			if _, ok := idSet[s.SessionID]; !ok {
				continue
			}

			candidates = append(candidates, s)
			affected += s.SizeBytes
		}

		if len(candidates) == 0 {
			return RestoreSelectResult{}, fmt.Errorf("batch id %q has no sessions currently restorable from trash", batchID)
		}

		return RestoreSelectResult{
			Sessions:      candidates,
			AffectedBytes: affected,
		}, nil
	}

	if !in.Selector.HasAnyFilter() {
		return RestoreSelectResult{}, errors.New("restore requires at least one selector (--id/--id-prefix/--host-contains/--path-contains/--head-contains/--older-than/--health or --batch-id)")
	}

	q, err := core.QuerySessions(in.Repository, in.TrashSessionsRoot, core.QuerySpec{
		Selector: in.Selector,
		Now:      in.Now,
	})
	if err != nil {
		return RestoreSelectResult{}, err
	}

	var affected int64
	for _, s := range q.Items {
		affected += s.SizeBytes
	}

	return RestoreSelectResult{
		Sessions:      q.Items,
		AffectedBytes: affected,
	}, nil
}
