package core

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MysticalDevil/codexsm/session"
	"github.com/MysticalDevil/codexsm/session/scanner"
)

type QueryResult struct {
	Total int
	Items []session.Session
}

// QuerySessions scans root via repo, applies selector, then performs
// normalized sorting and pagination.
func QuerySessions(repo SessionRepository, root string, spec QuerySpec) (QueryResult, error) {
	if repo == nil {
		repo = scanner.ScanSessions
	}

	if spec.Offset < 0 {
		return QueryResult{}, errInvalidOffset(spec.Offset)
	}

	sortSpec, err := normalizeSortSpec(spec)
	if err != nil {
		return QueryResult{}, err
	}

	items, err := repo(root)
	if err != nil {
		return QueryResult{}, err
	}

	now := spec.Now
	if now.IsZero() {
		now = time.Now()
	}

	filtered := session.FilterSessions(items, spec.Selector, now)
	sortSessions(filtered, sortSpec)

	total := len(filtered)
	if spec.Offset > 0 {
		if spec.Offset >= total {
			return QueryResult{Total: total, Items: []session.Session{}}, nil
		}

		filtered = filtered[spec.Offset:]
	}

	if spec.Limit > 0 && len(filtered) > spec.Limit {
		filtered = filtered[:spec.Limit]
	}

	return QueryResult{
		Total: total,
		Items: filtered,
	}, nil
}

func sortSessions(items []session.Session, spec SortSpec) {
	if len(items) <= 1 {
		return
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
		switch spec.Field {
		case SortFieldUpdatedAt:
			return a.UpdatedAt.Compare(b.UpdatedAt)
		case SortFieldCreatedAt:
			return a.CreatedAt.Compare(b.CreatedAt)
		case SortFieldSize:
			if a.SizeBytes < b.SizeBytes {
				return -1
			}

			if a.SizeBytes > b.SizeBytes {
				return 1
			}

			return 0
		case SortFieldHealth:
			ra := healthRank(a.Health)

			rb := healthRank(b.Health)
			if ra < rb {
				return -1
			}

			if ra > rb {
				return 1
			}

			return 0
		case SortFieldID:
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

		if spec.Order == SortOrderDesc {
			c = -c
		}

		return c
	})
}

func errInvalidOffset(v int) error {
	return &queryErr{msg: "invalid --offset value", value: v}
}

type queryErr struct {
	msg   string
	value int
}

func (e *queryErr) Error() string {
	return e.msg + " " + strconv.Itoa(e.value)
}
