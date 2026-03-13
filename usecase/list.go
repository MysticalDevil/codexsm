package usecase

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/MysticalDevil/codexsm/internal/core"
	"github.com/MysticalDevil/codexsm/session"
)

type ListInput struct {
	SessionsRoot string
	Selector     session.Selector
	Now          time.Time
	SortBy       string
	Order        string
	Limit        int
	Repository   core.SessionRepository
}

type ListResult struct {
	Total int
	Items []session.Session
}

func ListSessions(in ListInput) (ListResult, error) {
	repo := in.Repository
	if repo == nil {
		repo = core.ScannerRepository{}
	}
	items, err := repo.ScanSessions(in.SessionsRoot)
	if err != nil {
		return ListResult{}, err
	}
	now := in.Now
	if now.IsZero() {
		now = time.Now()
	}
	filtered := session.FilterSessions(items, in.Selector, now)
	if err := SortSessions(filtered, in.SortBy, in.Order); err != nil {
		return ListResult{}, err
	}
	total := len(filtered)
	if in.Limit > 0 && len(filtered) > in.Limit {
		filtered = filtered[:in.Limit]
	}
	return ListResult{Total: total, Items: filtered}, nil
}

func SortSessions(items []session.Session, sortBy, order string) error {
	if len(items) <= 1 {
		return nil
	}

	by := strings.ToLower(strings.TrimSpace(sortBy))
	if by == "" {
		by = "updated_at"
	}
	switch by {
	case "updated_at", "created_at", "size", "health", "id", "session_id":
	default:
		return fmt.Errorf("invalid --sort value %q", sortBy)
	}

	desc := true
	switch strings.ToLower(strings.TrimSpace(order)) {
	case "", "desc":
		desc = true
	case "asc":
		desc = false
	default:
		return fmt.Errorf("invalid --order value %q", order)
	}

	healthRank := func(h session.Health) int {
		switch h {
		case session.HealthOK:
			return 0
		case session.HealthMissingMeta:
			return 1
		case session.HealthCorrupted:
			return 2
		default:
			return 3
		}
	}

	compare := func(a, b session.Session) int {
		switch by {
		case "updated_at":
			return a.UpdatedAt.Compare(b.UpdatedAt)
		case "created_at":
			return a.CreatedAt.Compare(b.CreatedAt)
		case "size":
			if a.SizeBytes < b.SizeBytes {
				return -1
			}
			if a.SizeBytes > b.SizeBytes {
				return 1
			}
			return 0
		case "health":
			ra := healthRank(a.Health)
			rb := healthRank(b.Health)
			if ra < rb {
				return -1
			}
			if ra > rb {
				return 1
			}
			return 0
		case "id", "session_id":
			return strings.Compare(a.SessionID, b.SessionID)
		default:
			return 0
		}
	}

	slices.SortStableFunc(items, func(a, b session.Session) int {
		c := compare(a, b)
		if c == 0 {
			c = strings.Compare(a.SessionID, b.SessionID)
		}
		if c == 0 {
			c = strings.Compare(a.Path, b.Path)
		}
		if desc {
			c = -c
		}
		return c
	})
	return nil
}
