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
	okContent := `{"timestamp":"2026-03-02T09:44:02.106Z","type":"session_meta","payload":{"id":"019cadee-e315-7b91-8b5d-c0b52770cca6","timestamp":"2026-03-02T09:44:00.024Z","cwd":"/workspace/proj"}}` + "\n" +
		`{"timestamp":"2026-03-02T09:44:03.106Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"  hello   codex session manager  "}]}}` + "\n"
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
	root := t.TempDir()
	p := filepath.Join(root, "2026", "03", "03", "rollout-noise.jsonl")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `{"type":"session_meta","payload":{"id":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","timestamp":"2026-03-03T09:44:00.024Z"}}` + "\n" +
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"# AGENTS.md instructions for /workspace/project"}]}}` + "\n" +
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"默认list输出中不用输出文件名"}]}}` + "\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	list, err := ScanSessions(root)
	if err != nil {
		t.Fatalf("ScanSessions: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 session, got %d", len(list))
	}
	if list[0].Head != "默认list输出中不用输出文件名" {
		t.Fatalf("unexpected head: %q", list[0].Head)
	}
}
