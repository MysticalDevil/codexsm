package usecase

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
	"github.com/MysticalDevil/codexsm/session"
)

func TestSortSessions(t *testing.T) {
	items := []session.Session{
		{
			SessionID: "b",
			UpdatedAt: time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC),
			CreatedAt: time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC),
			SizeBytes: 20,
			Health:    session.HealthCorrupted,
			Path:      "/tmp/b.jsonl",
		},
		{
			SessionID: "a",
			UpdatedAt: time.Date(2026, 3, 2, 11, 0, 0, 0, time.UTC),
			CreatedAt: time.Date(2026, 3, 2, 8, 0, 0, 0, time.UTC),
			SizeBytes: 10,
			Health:    session.HealthOK,
			Path:      "/tmp/a.jsonl",
		},
	}

	if err := SortSessions(items, "size", "asc"); err != nil {
		t.Fatalf("SortSessions size asc: %v", err)
	}
	if items[0].SessionID != "a" {
		t.Fatalf("unexpected size asc order: %+v", items)
	}

	if err := SortSessions(items, "health", "asc"); err != nil {
		t.Fatalf("SortSessions health asc: %v", err)
	}
	if items[0].Health != session.HealthOK {
		t.Fatalf("unexpected health asc order: %+v", items)
	}

	if err := SortSessions(items, "updated_at", "desc"); err != nil {
		t.Fatalf("SortSessions updated_at desc: %v", err)
	}
	if items[0].UpdatedAt.Before(items[1].UpdatedAt) {
		t.Fatalf("unexpected updated_at desc order: %+v", items)
	}

	if err := SortSessions(items, "invalid", "asc"); err == nil {
		t.Fatal("expected invalid sort error")
	}
	if err := SortSessions(items, "size", "invalid"); err == nil {
		t.Fatal("expected invalid order error")
	}
}

func TestBuildGroupStats(t *testing.T) {
	items := []session.Session{
		{
			SessionID: "a",
			UpdatedAt: time.Date(2026, 3, 2, 11, 0, 0, 0, time.UTC),
			SizeBytes: 10,
			Health:    session.HealthOK,
		},
		{
			SessionID: "b",
			UpdatedAt: time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC),
			SizeBytes: 20,
			Health:    session.HealthCorrupted,
		},
		{
			SessionID: "c",
			UpdatedAt: time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC),
			SizeBytes: 30,
			Health:    session.HealthOK,
		},
	}

	stats, err := BuildGroupStats(items, "health", "count", "desc")
	if err != nil {
		t.Fatalf("BuildGroupStats: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(stats))
	}
	if stats[0].Group != "ok" || stats[0].Count != 2 {
		t.Fatalf("unexpected top group: %+v", stats[0])
	}

	if _, err := BuildGroupStats(items, "bad", "count", "desc"); err == nil {
		t.Fatal("expected invalid --by error")
	}
	if _, err := BuildGroupStats(items, "day", "bad", "desc"); err == nil {
		t.Fatal("expected invalid --sort error")
	}
	if _, err := BuildGroupStats(items, "day", "count", "bad"); err == nil {
		t.Fatal("expected invalid --order error")
	}
}

func TestListAndPreviewUsecases(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "sessions")

	result, err := ListSessions(ListInput{
		SessionsRoot: root,
		Selector:     session.Selector{},
		SortBy:       "updated_at",
		Order:        "desc",
		Limit:        3,
	})
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if result.Total == 0 || len(result.Items) == 0 {
		t.Fatalf("expected listed sessions, got total=%d items=%d", result.Total, len(result.Items))
	}
}

func TestExtractPreviewMessages(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.jsonl")
	lines := []string{
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"text":"hello world"}]}}`,
		`{"type":"response_item","payload":{"type":"message","role":"assistant","text":"ok"}}`,
		`{"type":"response_item","payload":{"type":"message","role":"assistant","text":"filesystem sandboxing note"}}`,
	}
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	items, err := ExtractPreviewMessages(p, 10)
	if err != nil {
		t.Fatalf("ExtractPreviewMessages: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("unexpected extracted count: %d", len(items))
	}
	if items[0].Role != "user" || items[0].Text != "hello world" {
		t.Fatalf("unexpected first message: %+v", items[0])
	}
}

func TestExtractPreviewMessages_LongLine(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "long.jsonl")
	payload := strings.Repeat("A", defaultPreviewMaxLineSize+1)
	line := `{"type":"response_item","payload":{"type":"message","role":"user","text":"` + payload + `"}}`
	if err := os.WriteFile(p, []byte(line+"\n"), 0o644); err != nil {
		t.Fatalf("write long fixture: %v", err)
	}
	_, err := ExtractPreviewMessages(p, 10)
	if !errors.Is(err, ErrPreviewEntryTooLong) {
		t.Fatalf("expected ErrPreviewEntryTooLong, got %v", err)
	}
}

func TestDoctorRisk(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "sessions")
	rep, err := DoctorRisk(DoctorRiskInput{
		SessionsRoot:   root,
		SampleLimit:    2,
		IntegrityCheck: true,
	})
	if err != nil {
		t.Fatalf("DoctorRisk: %v", err)
	}
	if rep.SessionsTotal == 0 || rep.RiskTotal == 0 {
		t.Fatalf("unexpected report: %+v", rep)
	}
	if rep.SampleLimit != 2 {
		t.Fatalf("expected sample limit=2, got %+v", rep)
	}
	if len(rep.Samples) > 2 {
		t.Fatalf("expected <=2 samples, got %d", len(rep.Samples))
	}
}

func TestCheckSessionHostPaths(t *testing.T) {
	sessionsRoot := t.TempDir()
	existingHost := t.TempDir()
	missingHost := filepath.Join(t.TempDir(), "missing-host-dir")

	writeDoctorSessionFixture(t, sessionsRoot, "s1", existingHost)
	writeDoctorSessionFixture(t, sessionsRoot, "s2", missingHost)

	got := CheckSessionHostPaths(DoctorHostPathInput{
		SessionsRoot: sessionsRoot,
		CompactPath:  func(v string, _ int) string { return v },
	})
	if got.Level != DoctorWarn {
		t.Fatalf("expected warn, got %s detail=%q", got.Level, got.Detail)
	}
	if !strings.Contains(got.Detail, "recommended_actions:") {
		t.Fatalf("expected action block in detail, got: %q", got.Detail)
	}
}

func writeDoctorSessionFixture(t *testing.T, sessionsRoot, id, host string) {
	t.Helper()
	dir := filepath.Join(sessionsRoot, "2026", "03", "08")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir sessions fixture: %v", err)
	}
	path := filepath.Join(dir, id+".jsonl")
	line := `{"type":"session_meta","payload":{"id":"` + id + `","cwd":"` + host + `","timestamp":"` + time.Now().UTC().Format(time.RFC3339Nano) + `"}}` + "\n"
	if err := os.WriteFile(path, []byte(line), 0o644); err != nil {
		t.Fatalf("write session fixture: %v", err)
	}
}
