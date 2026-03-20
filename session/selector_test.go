package session

import (
	"testing"
	"time"
)

func TestFilterSessions(t *testing.T) {
	now := time.Now()
	sessions := []Session{
		{SessionID: "abc-1", UpdatedAt: now.Add(-48 * time.Hour), Health: HealthOK, HostDir: "/workspace/proj-a", Path: "/sessions/a.jsonl", Head: "selector fixture"},
		{SessionID: "abc-2", UpdatedAt: now.Add(-2 * time.Hour), Health: HealthCorrupted, HostDir: "/workspace/proj-b", Path: "/sessions/b.jsonl", Head: "other"},
		{SessionID: "xyz-1", UpdatedAt: now.Add(-96 * time.Hour), Health: HealthOK, HostDir: "/var/tmp/proj", Path: "/trash/c.jsonl", Head: "selector fixture"},
	}
	sel := Selector{
		IDPrefix:     "abc",
		HostContains: "proj-a",
		PathContains: "/sessions/",
		HeadContains: "SELECTOR",
		OlderThan:    24 * time.Hour,
		HasOlderThan: true,
		Health:       HealthOK,
		HasHealth:    true,
	}

	got := FilterSessions(sessions, sel, now)
	if len(got) != 1 {
		t.Fatalf("expected 1 got %d", len(got))
	}

	if got[0].SessionID != "abc-1" {
		t.Fatalf("unexpected id %s", got[0].SessionID)
	}
}

func TestSelectorHasAnyFilter_WithContainsFilters(t *testing.T) {
	if !(Selector{HostContains: "/workspace"}).HasAnyFilter() {
		t.Fatal("expected host filter to count as active selector")
	}

	if !(Selector{PathContains: "rollout"}).HasAnyFilter() {
		t.Fatal("expected path filter to count as active selector")
	}

	if !(Selector{HeadContains: "fixture"}).HasAnyFilter() {
		t.Fatal("expected head filter to count as active selector")
	}
}

func TestFilterSessions_MultilingualAndEmojiHeadContains(t *testing.T) {
	now := time.Now()
	sessions := []Session{
		{SessionID: "zh", UpdatedAt: now.Add(-time.Minute), Head: "请帮我实现会话恢复功能"},
		{SessionID: "en", UpdatedAt: now.Add(-2 * time.Minute), Head: "please implement retry logic"},
		{SessionID: "es", UpdatedAt: now.Add(-3 * time.Minute), Head: "por favor implementar filtro de sesiones"},
		{SessionID: "la", UpdatedAt: now.Add(-4 * time.Minute), Head: "salve quaeso sessiones refice"},
		{SessionID: "ja", UpdatedAt: now.Add(-5 * time.Minute), Head: "日本語のセッション表示を改善してください"},
		{SessionID: "ko", UpdatedAt: now.Add(-6 * time.Minute), Head: "세션 목록을 빠르게 개선해 주세요"},
		{SessionID: "ar", UpdatedAt: now.Add(-7 * time.Minute), Head: "يرجى تحسين فحص الجلسات"},
		{SessionID: "mix", UpdatedAt: now.Add(-8 * time.Minute), Head: "fix list command gracias 日本語 مرحبا 😄"},
	}

	cases := []struct {
		name string
		q    string
		id   string
	}{
		{name: "chinese", q: "实现", id: "zh"},
		{name: "english-fold", q: "RETRY LOGIC", id: "en"},
		{name: "spanish-fold", q: "IMPLEMENTAR", id: "es"},
		{name: "latin", q: "sessiones", id: "la"},
		{name: "japanese", q: "表示を改善", id: "ja"},
		{name: "korean", q: "세션", id: "ko"},
		{name: "arabic", q: "الجلسات", id: "ar"},
		{name: "emoji", q: "😄", id: "mix"},
		{name: "multilingual-mixed", q: "gracias 日本語", id: "mix"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := FilterSessions(sessions, Selector{HeadContains: tc.q}, now)
			if len(got) != 1 {
				t.Fatalf("expected one match for %q, got %d", tc.q, len(got))
			}

			if got[0].SessionID != tc.id {
				t.Fatalf("unexpected id for %q: %s", tc.q, got[0].SessionID)
			}
		})
	}
}

func TestFilterSessions_PreservesInputOrder(t *testing.T) {
	now := time.Now()
	sessions := []Session{
		{SessionID: "first", UpdatedAt: now.Add(-time.Hour)},
		{SessionID: "second", UpdatedAt: now.Add(-time.Minute)},
		{SessionID: "third", UpdatedAt: now.Add(-2 * time.Hour)},
	}

	got := FilterSessions(sessions, Selector{}, now)
	if len(got) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(got))
	}

	if got[0].SessionID != "first" || got[1].SessionID != "second" || got[2].SessionID != "third" {
		t.Fatalf("unexpected order: %+v", got)
	}
}
