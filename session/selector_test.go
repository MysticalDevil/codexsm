package session

import (
	"testing"
	"time"
)

func TestFilterSessions(t *testing.T) {
	now := time.Now()
	sessions := []Session{
		{SessionID: "abc-1", UpdatedAt: now.Add(-48 * time.Hour), Health: HealthOK},
		{SessionID: "abc-2", UpdatedAt: now.Add(-2 * time.Hour), Health: HealthCorrupted},
		{SessionID: "xyz-1", UpdatedAt: now.Add(-96 * time.Hour), Health: HealthOK},
	}
	sel := Selector{IDPrefix: "abc", OlderThan: 24 * time.Hour, HasOlderThan: true, Health: HealthOK, HasHealth: true}
	got := FilterSessions(sessions, sel, now)
	if len(got) != 1 {
		t.Fatalf("expected 1 got %d", len(got))
	}
	if got[0].SessionID != "abc-1" {
		t.Fatalf("unexpected id %s", got[0].SessionID)
	}
}
