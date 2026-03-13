package core

import (
	"time"

	"github.com/MysticalDevil/codexsm/session"
)

// QuerySpec describes shared session-query constraints used by usecases.
type QuerySpec struct {
	Selector session.Selector
	SortBy   string
	Order    string
	Limit    int
	Now      time.Time
}

// SessionRepository provides access to raw session rows.
type SessionRepository interface {
	ScanSessions(root string) ([]session.Session, error)
}

// ScannerRepository is the default repository backed by session.ScanSessions.
type ScannerRepository struct{}

// ScanSessions implements SessionRepository.
func (ScannerRepository) ScanSessions(root string) ([]session.Session, error) {
	return session.ScanSessions(root)
}

// RiskEvaluator evaluates one session and returns the highest-priority risk.
type RiskEvaluator interface {
	Evaluate(session.Session, session.IntegrityChecker) session.Risk
}

// SessionRiskEvaluator is the default evaluator backed by session.EvaluateRisk.
type SessionRiskEvaluator struct{}

// Evaluate implements RiskEvaluator.
func (SessionRiskEvaluator) Evaluate(s session.Session, checker session.IntegrityChecker) session.Risk {
	return session.EvaluateRisk(s, checker)
}

