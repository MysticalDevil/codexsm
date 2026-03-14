package usecase

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
	"github.com/MysticalDevil/codexsm/session"
)

func TestSelectDeleteSessions(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "sessions")

	_, err := SelectDeleteSessions(DeleteSelectInput{
		SessionsRoot: root,
		Selector:     session.Selector{},
		Now:          time.Now(),
	})
	if err == nil || !strings.Contains(err.Error(), "requires at least one selector") {
		t.Fatalf("expected selector error, got: %v", err)
	}

	res, err := SelectDeleteSessions(DeleteSelectInput{
		SessionsRoot: root,
		Selector: session.Selector{
			ID: "11111111-1111-1111-1111-111111111111",
		},
		Now: time.Now(),
	})
	if err != nil {
		t.Fatalf("SelectDeleteSessions: %v", err)
	}

	if len(res.Sessions) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(res.Sessions))
	}

	if res.AffectedBytes <= 0 {
		t.Fatalf("expected affected bytes > 0, got %d", res.AffectedBytes)
	}
}
