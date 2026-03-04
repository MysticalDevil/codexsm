package audit

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MysticalDevil/codex-sm/session"
)

func TestWriteActionLog(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "logs", "actions.log")
	rec := ActionRecord{
		Timestamp:  time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC),
		Action:     "soft-delete",
		Simulation: true,
		Selector: session.Selector{
			IDPrefix:     "019c",
			OlderThan:    30 * time.Minute,
			HasOlderThan: true,
			Health:       session.HealthOK,
			HasHealth:    true,
		},
		MatchedCount:  1,
		AffectedBytes: 123,
		Sessions:      []SessionRef{{SessionID: "s1", Path: "/tmp/s1.jsonl"}},
	}

	if err := WriteActionLog(logFile, rec); err != nil {
		t.Fatalf("WriteActionLog #1: %v", err)
	}
	if err := WriteActionLog(logFile, rec); err != nil {
		t.Fatalf("WriteActionLog #2: %v", err)
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(lines))
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatalf("unmarshal first log line: %v", err)
	}

	sel, ok := got["selector"].(map[string]any)
	if !ok {
		t.Fatalf("selector missing or wrong type: %#v", got["selector"])
	}
	if sel["older_than"] != "30m0s" {
		t.Fatalf("older_than should be duration string, got: %#v", sel["older_than"])
	}
	if sel["id_prefix"] != "019c" {
		t.Fatalf("id_prefix mismatch: %#v", sel["id_prefix"])
	}
}
