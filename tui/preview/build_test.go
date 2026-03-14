package preview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassifyAngleTag(t *testing.T) {
	tests := []struct {
		tag  string
		want AngleTagTone
	}{
		{tag: "<turn_aborted>", want: AngleTagToneDanger},
		{tag: "</turn_aborted>", want: AngleTagToneDanger},
		{tag: "<environment_context>", want: AngleTagToneSystem},
		{tag: "<collaboration_mode>", want: AngleTagToneSystem},
		{tag: "<session_meta>", want: AngleTagToneLifecycle},
		{tag: "<operation_done>", want: AngleTagToneSuccess},
		{tag: "<generic_tag>", want: AngleTagToneDefault},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.tag, func(t *testing.T) {
			if got := ClassifyAngleTag(tt.tag); got != tt.want {
				t.Fatalf("ClassifyAngleTag(%q)=%v, want=%v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestCacheKeyForSession(t *testing.T) {
	k1 := CacheKeyForSession("/tmp/a.jsonl", 24, 100, 200)
	k2 := CacheKeyForSession("/tmp/a.jsonl", 48, 100, 200)
	k3 := CacheKeyForSession("/tmp/a.jsonl", 24, 101, 200)
	if k1 == k2 {
		t.Fatalf("expected width to affect cache key: %q", k1)
	}
	if k1 == k3 {
		t.Fatalf("expected size to affect cache key: %q", k1)
	}
}

func TestBuildLinesFriendlyOversizeWarning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "oversize.jsonl")
	longText := strings.Repeat("x", 1024*1024+128)
	content := `{"type":"session_meta","payload":{"id":"x","timestamp":"2026-03-02T09:44:00.024Z"}}` + "\n" +
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"` + longText + `"}]}}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write preview fixture: %v", err)
	}

	lines := BuildLines(path, 64, 20, ThemePalette{})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "preview unavailable: a session entry exceeds the safe preview limit") {
		t.Fatalf("expected friendly oversize warning, got: %q", joined)
	}
	if strings.Contains(joined, "token too long") {
		t.Fatalf("did not expect raw scanner error, got: %q", joined)
	}
}
