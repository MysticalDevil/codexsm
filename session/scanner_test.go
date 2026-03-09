package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
)

func TestScanSessionsHealthAndID(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "sessions")
	okFile := filepath.Join(root, "2026", "03", "02", "rollout-scanner-ok.jsonl")
	missingMeta := filepath.Join(root, "missing.jsonl")
	corrupted := filepath.Join(root, "bad.jsonl")

	list, err := ScanSessions(root)
	if err != nil {
		t.Fatalf("ScanSessions: %v", err)
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
			if s.Head != "hello codex session manager" {
				t.Fatalf("unexpected session head: %q", s.Head)
			}
			if s.HostDir != "/workspace/proj" {
				t.Fatalf("unexpected host dir: %q", s.HostDir)
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

func TestScanSessionsHeadSkipsInstructionNoise(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "sessions")
	p := filepath.Join(root, "2026", "03", "03", "rollout-noise.jsonl")

	list, err := ScanSessions(root)
	if err != nil {
		t.Fatalf("ScanSessions: %v", err)
	}

	found := false
	for _, s := range list {
		if s.Path != p {
			continue
		}
		found = true
		if s.Head != "default list output should hide filename" {
			t.Fatalf("unexpected head: %q", s.Head)
		}
		break
	}
	if !found {
		t.Fatalf("noise fixture not found: %s", p)
	}
}

func TestScanSessionsMarksOverlongMetaLineCorrupted(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "oversize.jsonl")
	overlong := `{"type":"session_meta","payload":{"id":"x","timestamp":"` + strings.Repeat("1", maxSessionMetaLineBytes) + `"}}`
	if err := os.WriteFile(p, []byte(overlong), 0o644); err != nil {
		t.Fatalf("write oversize fixture: %v", err)
	}

	list, err := ScanSessions(root)
	if err != nil {
		t.Fatalf("ScanSessions: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 scanned session, got %d", len(list))
	}
	if list[0].Health != HealthCorrupted {
		t.Fatalf("expected corrupted health for oversize meta line, got %s", list[0].Health)
	}
}
