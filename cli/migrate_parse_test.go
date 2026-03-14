package cli

import "testing"

func TestParseSinceTime(t *testing.T) {
	if _, ok, err := parseSinceTime("   "); err != nil || ok {
		t.Fatalf("empty since should be unset, ok=%v err=%v", ok, err)
	}

	if _, ok, err := parseSinceTime("2026-03-10"); err != nil || !ok {
		t.Fatalf("date since should parse, ok=%v err=%v", ok, err)
	}

	if _, _, err := parseSinceTime("bad-time"); err == nil {
		t.Fatal("expected parse error")
	}
}
