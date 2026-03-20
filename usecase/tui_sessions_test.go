package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/session"
)

func repoWith(items []session.Session, err error) func(string) ([]session.Session, error) {
	return func(_ string) ([]session.Session, error) {
		if err != nil {
			return nil, err
		}

		out := make([]session.Session, len(items))
		copy(out, items)

		return out, nil
	}
}

func healthRiskEvaluator(s session.Session, _ session.IntegrityChecker) session.Risk {
	switch s.Health {
	case session.HealthCorrupted:
		return session.Risk{Level: session.RiskHigh}
	case session.HealthMissingMeta:
		return session.Risk{Level: session.RiskMedium}
	case session.HealthOK:
		return session.Risk{Level: session.RiskNone}
	}

	return session.Risk{Level: session.RiskNone}
}

func TestLoadTUISessionsRiskSortAndLimits(t *testing.T) {
	now := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)
	items := []session.Session{
		{SessionID: "ok-new", UpdatedAt: now.Add(3 * time.Minute), Health: session.HealthOK},
		{SessionID: "high-old", UpdatedAt: now.Add(-5 * time.Minute), Health: session.HealthCorrupted},
		{SessionID: "mid-mid", UpdatedAt: now.Add(-1 * time.Minute), Health: session.HealthMissingMeta},
		{SessionID: "ok-old", UpdatedAt: now.Add(-3 * time.Minute), Health: session.HealthOK},
	}

	out, err := LoadTUISessions(LoadTUISessionsInput{
		SessionsRoot: "/tmp/sessions",
		ScanLimit:    3,
		ViewLimit:    2,
		Repository:   repoWith(items, nil),
		Evaluator:    healthRiskEvaluator,
	})
	if err != nil {
		t.Fatalf("LoadTUISessions: %v", err)
	}

	if out.Total != 4 {
		t.Fatalf("unexpected total: %d", out.Total)
	}

	if len(out.Items) != 2 {
		t.Fatalf("unexpected item count: %d", len(out.Items))
	}

	if out.Items[0].SessionID != "high-old" {
		t.Fatalf("expected high risk first, got %q", out.Items[0].SessionID)
	}

	if out.Items[1].SessionID != "mid-mid" {
		t.Fatalf("expected medium risk second, got %q", out.Items[1].SessionID)
	}
}

func TestLoadTUISessionsRepositoryError(t *testing.T) {
	want := errors.New("scan failed")

	_, err := LoadTUISessions(LoadTUISessionsInput{
		SessionsRoot: "/tmp/sessions",
		Repository:   repoWith(nil, want),
	})
	if !errors.Is(err, want) {
		t.Fatalf("expected repository error, got %v", err)
	}
}
