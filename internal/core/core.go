package core

import (
	"fmt"
	"strings"
	"time"

	"github.com/MysticalDevil/codexsm/session"
)

type SortField string

const (
	SortFieldUpdatedAt SortField = "updated_at"
	SortFieldCreatedAt SortField = "created_at"
	SortFieldSize      SortField = "size"
	SortFieldHealth    SortField = "health"
	SortFieldID        SortField = "id"
)

type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

// SortSpec is the normalized internal sort model for session queries.
type SortSpec struct {
	Field SortField
	Order SortOrder
}

// QuerySpec describes shared session-query constraints used by usecases.
type QuerySpec struct {
	Selector session.Selector
	Offset   int
	Limit    int
	Sort     SortSpec

	// Legacy string fields are accepted for CLI-facing compatibility and
	// normalized into Sort when Sort is not provided.
	SortBy string
	Order  string
	Now    time.Time
}

func ParseSortSpec(sortBy, order string) (SortSpec, error) {
	by := strings.ToLower(strings.TrimSpace(sortBy))
	if by == "" {
		by = string(SortFieldUpdatedAt)
	}

	var field SortField

	switch by {
	case "updated_at":
		field = SortFieldUpdatedAt
	case "created_at":
		field = SortFieldCreatedAt
	case "size":
		field = SortFieldSize
	case "health":
		field = SortFieldHealth
	case "id", "session_id":
		field = SortFieldID
	default:
		return SortSpec{}, fmt.Errorf("invalid --sort value %q", sortBy)
	}

	ord := strings.ToLower(strings.TrimSpace(order))
	if ord == "" {
		ord = string(SortOrderDesc)
	}

	var normalized SortOrder

	switch ord {
	case "asc":
		normalized = SortOrderAsc
	case "desc":
		normalized = SortOrderDesc
	default:
		return SortSpec{}, fmt.Errorf("invalid --order value %q", order)
	}

	return SortSpec{Field: field, Order: normalized}, nil
}

func normalizeSortSpec(spec QuerySpec) (SortSpec, error) {
	if spec.Sort.Field != "" {
		field := spec.Sort.Field
		switch field {
		case SortFieldUpdatedAt, SortFieldCreatedAt, SortFieldSize, SortFieldHealth, SortFieldID:
		default:
			return SortSpec{}, fmt.Errorf("invalid --sort value %q", string(field))
		}

		order := spec.Sort.Order
		if order == "" {
			order = SortOrderDesc
		}

		switch order {
		case SortOrderAsc, SortOrderDesc:
		default:
			return SortSpec{}, fmt.Errorf("invalid --order value %q", string(order))
		}

		return SortSpec{Field: field, Order: order}, nil
	}

	return ParseSortSpec(spec.SortBy, spec.Order)
}

// SessionRepository provides access to raw session rows.
type SessionRepository func(root string) ([]session.Session, error)

// RiskEvaluator evaluates one session and returns the highest-priority risk.
type RiskEvaluator func(session.Session, session.IntegrityChecker) session.Risk
