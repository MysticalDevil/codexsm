package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanSessionsHealthAndID(t *testing.T) {
	root := t.TempDir()

	okFile := filepath.Join(root, "2026", "03", "02", "rollout-2026-03-02T17-44-00-019cadee-e315-7b91-8b5d-c0b52770cca6.jsonl")
	if err := os.MkdirAll(filepath.Dir(okFile), 0o755); err != nil {
		t.Fatal(err)
	}
	okContent := `{"timestamp":"2026-03-02T09:44:02.106Z","type":"session_meta","payload":{"id":"019cadee-e315-7b91-8b5d-c0b52770cca6","timestamp":"2026-03-02T09:44:00.024Z"}}` + "\n"
	if err := os.WriteFile(okFile, []byte(okContent), 0o644); err != nil {
		t.Fatal(err)
	}

	missingMeta := filepath.Join(root, "missing.jsonl")
	if err := os.WriteFile(missingMeta, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	corrupted := filepath.Join(root, "bad.jsonl")
	if err := os.WriteFile(corrupted, []byte("{not-json\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	list, err := ScanSessions(root)
	if err != nil {
		t.Fatalf("ScanSessions: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(list))
	}

	foundOK := false
	foundMissing := false
	foundCorrupted := false
	for _, s := range list {
		switch s.Path {
		case okFile:
			foundOK = true
			if s.Health != HealthOK {
				t.Fatalf("ok file health=%s", s.Health)
			}
			if s.SessionID != "019cadee-e315-7b91-8b5d-c0b52770cca6" {
				t.Fatalf("unexpected session id: %s", s.SessionID)
			}
		case missingMeta:
			foundMissing = true
			if s.Health != HealthMissingMeta {
				t.Fatalf("missing file health=%s", s.Health)
			}
		case corrupted:
			foundCorrupted = true
			if s.Health != HealthCorrupted {
				t.Fatalf("bad file health=%s", s.Health)
			}
		}
	}
	if !foundOK || !foundMissing || !foundCorrupted {
		t.Fatalf("did not find all expected files")
	}
}
