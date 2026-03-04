package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mattn/go-runewidth"

	"github.com/MysticalDevil/codex-sm/internal/testsupport"
	"github.com/MysticalDevil/codex-sm/session"
)

var ansiSeqRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSIForTest(s string) string {
	return ansiSeqRe.ReplaceAllString(s, "")
}

func TestTUIHandleKeySwitchAndScroll(t *testing.T) {
	m := tuiModel{
		sessions: []session.Session{
			{SessionID: "a", UpdatedAt: time.Now()},
			{SessionID: "b", UpdatedAt: time.Now().Add(-time.Minute)},
		},
		previewCache: make(map[string][]string),
		focus:        focusTree,
	}
	m.rebuildTree()

	if quit := m.handleKey("tab"); quit {
		t.Fatal("tab should not quit")
	}
	if m.focus != focusPreview {
		t.Fatalf("expected focusPreview, got %v", m.focus)
	}

	m.handleKey("j")
	if m.previewOffset != 1 {
		t.Fatalf("expected previewOffset=1, got %d", m.previewOffset)
	}

	m.handleKey("t")
	if m.focus != focusTree {
		t.Fatalf("expected focusTree, got %v", m.focus)
	}
	start := m.cursor
	m.handleKey("j")
	if m.cursor == start {
		t.Fatalf("expected tree cursor to move, cursor=%d", m.cursor)
	}
}

func TestTUIViewMinSizeWarning(t *testing.T) {
	m := tuiModel{width: 80, height: 20}
	out := m.View()
	if !strings.Contains(out, "Required at least: 100x24") {
		t.Fatalf("expected min-size warning, got: %q", out)
	}
	if !strings.Contains(out, "KEYS") {
		t.Fatalf("expected KEYS bar, got: %q", out)
	}
}

func TestWrapAndTruncateDisplayWidth(t *testing.T) {
	v := "这是一个非常非常长的中文字符串用于测试宽度处理"
	lines := wrapText(v, 12)
	for _, line := range lines {
		if got := runewidth.StringWidth(line); got > 12 {
			t.Fatalf("wrapped line exceeds width: %q width=%d", line, got)
		}
	}

	tr := truncateDisplay(v, 10)
	if got := runewidth.StringWidth(tr); got > 10 {
		t.Fatalf("truncated text exceeds width: %q width=%d", tr, got)
	}
}

func TestPreviewForLimitsPerMessageToTwoLines(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "tmp")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir tmp: %v", err)
	}
	p := filepath.Join(root, "preview.jsonl")
	content := `{"type":"session_meta","payload":{"id":"x","timestamp":"2026-03-02T09:44:00.024Z"}}` + "\n" +
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen sixteen seventeen eighteen nineteen twenty twentyone twentytwo twentythree twentyfour"}]}}` + "\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write preview fixture: %v", err)
	}

	m := tuiModel{
		previewCache: make(map[string][]string),
	}
	out := m.previewFor(p, 12, 20)
	if len(out) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(out))
	}
	// One message should be capped to 2 rendered lines in preview output.
	if len(out) > 2 {
		t.Fatalf("expected message preview max 2 lines, got %d lines: %#v", len(out), out)
	}
	if !strings.Contains(out[len(out)-1], "...") {
		t.Fatalf("expected ellipsis on truncated second line, got: %#v", out)
	}
}

func TestPreviewForShowsRolePerMessage(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "tmp")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir tmp: %v", err)
	}
	p := filepath.Join(root, "roles.jsonl")
	content := `{"type":"session_meta","payload":{"id":"x","timestamp":"2026-03-02T09:44:00.024Z"}}` + "\n" +
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"u1 u2 u3 u4 u5 u6 u7 u8 u9 u10 u11 u12 u13"}]}}` + "\n" +
		`{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"a1 a2 a3 a4 a5 a6 a7 a8 a9 a10 a11 a12 a13"}]}}` + "\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write preview fixture: %v", err)
	}

	m := tuiModel{
		previewCache: make(map[string][]string),
	}
	out := m.previewFor(p, 24, 20)
	clean := make([]string, 0, len(out))
	for _, v := range out {
		clean = append(clean, stripANSIForTest(v))
	}
	got := strings.Join(clean, "\n")
	if !strings.Contains(got, "\n U ") && !strings.HasPrefix(got, " U ") {
		t.Fatalf("expected user role marker per message, got: %q", got)
	}
	if !strings.Contains(got, "\n A ") && !strings.HasPrefix(got, " A ") {
		t.Fatalf("expected assistant role marker per message, got: %q", got)
	}
}

func TestPreviewForLineWidthBound(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{
			name: "english",
			text: "abcdefghijklmnopqrstuvwxyz ABCDEFGHIJKLMNOPQRSTUVWXYZ 0123456789 this is a very long english sentence for preview width bound checks",
		},
		{
			name: "chinese",
			text: "这是一段非常非常长的中文文本用于测试预览窗口在窄宽度和宽宽度下的边界换行以及强制截断逻辑是否稳定",
		},
		{
			name: "mixed",
			text: "build 成功后 run integration tests，然后检查 preview width bound 与 scroll 状态是否一致，最后输出 report",
		},
	}
	widths := []int{10, 12, 24, 32, 48}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			for _, width := range widths {
				width := width
				t.Run("w"+strconv.Itoa(width), func(t *testing.T) {
					workspace := testsupport.PrepareFixtureSandbox(t, "rich")
					root := filepath.Join(workspace, "tmp")
					if err := os.MkdirAll(root, 0o755); err != nil {
						t.Fatalf("mkdir tmp: %v", err)
					}
					p := filepath.Join(root, "width-"+tc.name+".jsonl")
					content := `{"type":"session_meta","payload":{"id":"x","timestamp":"2026-03-02T09:44:00.024Z"}}` + "\n" +
						`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"` + tc.text + `"}]}}` + "\n"
					if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
						t.Fatalf("write preview fixture: %v", err)
					}

					m := tuiModel{
						previewCache: make(map[string][]string),
					}
					out := m.previewFor(p, width, 20)
					for _, line := range out {
						if got := runewidth.StringWidth(stripANSIForTest(line)); got > width {
							t.Fatalf("preview line exceeds width=%d, got=%d line=%q", width, got, line)
						}
					}
				})
			}
		})
	}
}

func TestClassifyAngleTag(t *testing.T) {
	tests := []struct {
		tag  string
		want angleTagTone
	}{
		{tag: "<turn_aborted>", want: angleTagToneDanger},
		{tag: "</turn_aborted>", want: angleTagToneDanger},
		{tag: "<environment_context>", want: angleTagToneSystem},
		{tag: "<collaboration_mode>", want: angleTagToneSystem},
		{tag: "<session_meta>", want: angleTagToneLifecycle},
		{tag: "<operation_done>", want: angleTagToneSuccess},
		{tag: "<generic_tag>", want: angleTagToneDefault},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.tag, func(t *testing.T) {
			if got := classifyAngleTag(tt.tag); got != tt.want {
				t.Fatalf("classifyAngleTag(%q)=%v, want=%v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestNormalizeTUIGroupBy(t *testing.T) {
	okCases := []string{"month", "day", "health", "host", "none", ""}
	for _, in := range okCases {
		got, err := normalizeTUIGroupBy(in)
		if err != nil {
			t.Fatalf("normalizeTUIGroupBy(%q) error: %v", in, err)
		}
		if in == "" && got != "month" {
			t.Fatalf("expected default month, got %q", got)
		}
	}
	if _, err := normalizeTUIGroupBy("invalid"); err == nil {
		t.Fatal("expected error for invalid group-by")
	}
}

func TestRebuildTreeGroupingModes(t *testing.T) {
	sessions := []session.Session{
		{SessionID: "s1", UpdatedAt: time.Date(2026, 3, 5, 10, 0, 0, 0, time.Local), Health: session.HealthOK, HostDir: "/tmp/a"},
		{SessionID: "s2", UpdatedAt: time.Date(2026, 3, 5, 11, 0, 0, 0, time.Local), Health: session.HealthCorrupted, HostDir: "/tmp/b"},
		{SessionID: "s3", UpdatedAt: time.Date(2026, 2, 1, 9, 0, 0, 0, time.Local), Health: session.HealthOK, HostDir: "/tmp/a"},
	}

	tests := []struct {
		mode             string
		expectGroupNodes bool
	}{
		{mode: "month", expectGroupNodes: true},
		{mode: "day", expectGroupNodes: true},
		{mode: "health", expectGroupNodes: true},
		{mode: "host", expectGroupNodes: true},
		{mode: "none", expectGroupNodes: false},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.mode, func(t *testing.T) {
			m := tuiModel{
				sessions: sessions,
				groupBy:  tc.mode,
				home:     "",
			}
			m.rebuildTree()
			groupCount := 0
			sessionCount := 0
			for _, n := range m.tree {
				if n.kind == treeItemMonth {
					groupCount++
				}
				if n.kind == treeItemSession {
					sessionCount++
				}
			}
			if sessionCount != len(sessions) {
				t.Fatalf("session node count=%d, want=%d", sessionCount, len(sessions))
			}
			if tc.expectGroupNodes && groupCount == 0 {
				t.Fatalf("expected group nodes for mode=%s", tc.mode)
			}
			if !tc.expectGroupNodes && groupCount != 0 {
				t.Fatalf("expected no group nodes for mode=%s, got=%d", tc.mode, groupCount)
			}
		})
	}
}
