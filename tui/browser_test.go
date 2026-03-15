package tui

import (
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"

	"github.com/MysticalDevil/codexsm/internal/core"
	"github.com/MysticalDevil/codexsm/internal/testsupport"
	"github.com/MysticalDevil/codexsm/session"
	"github.com/MysticalDevil/codexsm/session/scanner"
	"github.com/MysticalDevil/codexsm/tui/preview"
	"github.com/MysticalDevil/codexsm/usecase"
)

var (
	ansiSeqRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	widthCond = runewidth.NewCondition()
)

func init() {
	widthCond.EastAsianWidth = false
}

func stripANSIForTest(s string) string {
	return ansiSeqRe.ReplaceAllString(s, "")
}

func displayWidthForTest(s string) int {
	return widthCond.StringWidth(s)
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
	if !strings.Contains(out, "Required at least: 65x24") {
		t.Fatalf("expected min-size warning, got: %q", out)
	}

	if strings.Contains(out, "q quit") {
		t.Fatalf("did not expect extra bottom quit hint, got: %q", out)
	}

	if strings.Contains(out, "[KEYS]") {
		t.Fatalf("did not expect keybar in min-size warning, got: %q", out)
	}

	maxWidth := Compute(m.width, m.height).TotalW
	for _, line := range strings.Split(stripANSIForTest(out), "\n") {
		if got := displayWidthForTest(line); got > maxWidth {
			t.Fatalf("min-size warning line exceeds width=%d, got=%d line=%q", maxWidth, got, line)
		}
	}
}

func TestTUIViewCompactModeAtMinimumWidth(t *testing.T) {
	m := tuiModel{
		width:   81,
		height:  30,
		focus:   focusTree,
		groupBy: "host",
		sessions: []session.Session{
			{SessionID: "a", UpdatedAt: time.Now(), HostDir: "/tmp/a", Health: session.HealthOK},
			{SessionID: "b", UpdatedAt: time.Now().Add(-time.Minute), HostDir: "/tmp/b", Health: session.HealthMissingMeta},
		},
		previewCache: make(map[string][]string),
	}
	m.rebuildTree()

	out := stripANSIForTest(m.View())
	if strings.Contains(out, "Terminal too small") {
		t.Fatalf("did not expect min-size warning at compact minimum width, got: %q", out)
	}

	if !strings.Contains(out, "SES [C]") {
		t.Fatalf("expected compact indicator in tree title, got: %q", out)
	}

	maxWidth := Compute(m.width, m.height).TotalW
	for _, line := range strings.Split(out, "\n") {
		if got := displayWidthForTest(line); got > maxWidth {
			t.Fatalf("compact view line exceeds width=%d, got=%d line=%q", maxWidth, got, line)
		}
	}

	joined := strings.Join(strings.Split(out, "\n"), "\n")

	if strings.Contains(joined, " ├─ ") || strings.Contains(joined, " └─ ") {
		t.Fatalf("compact tree should not use connector symbols, got: %q", joined)
	}

	if strings.Contains(joined, "▾ ") || strings.Contains(joined, "▸ ") {
		t.Fatalf("compact tree should not show group arrow symbols, got: %q", joined)
	}
}

func TestTUIViewUltraModeAtNarrowWidth(t *testing.T) {
	m := tuiModel{
		width:   79,
		height:  30,
		focus:   focusTree,
		groupBy: "host",
		sessions: []session.Session{
			{SessionID: "a", UpdatedAt: time.Now(), HostDir: "/tmp/a", Health: session.HealthOK},
			{SessionID: "b", UpdatedAt: time.Now().Add(-time.Minute), HostDir: "/tmp/b", Health: session.HealthMissingMeta},
		},
		previewCache: make(map[string][]string),
	}
	m.rebuildTree()

	out := stripANSIForTest(m.View())
	if strings.Contains(out, "Terminal too small") {
		t.Fatalf("did not expect min-size warning at ultra width, got: %q", out)
	}

	if !strings.Contains(out, "SESSIONS (By Host)") {
		t.Fatalf("expected full tree title in ultra tree pane, got: %q", out)
	}

	maxWidth := Compute(m.width, m.height).TotalW
	for _, line := range strings.Split(out, "\n") {
		if got := displayWidthForTest(line); got > maxWidth {
			t.Fatalf("ultra view line exceeds width=%d, got=%d line=%q", maxWidth, got, line)
		}
	}
}

func TestTUIUltraTabSwitchesPaneAndBlocksActions(t *testing.T) {
	m := tuiModel{
		width:   79,
		height:  30,
		focus:   focusTree,
		groupBy: "host",
		sessions: []session.Session{
			{SessionID: "a", UpdatedAt: time.Now(), HostDir: "/tmp/a", Health: session.HealthOK},
		},
		previewCache: make(map[string][]string),
	}
	m.rebuildTree()

	if m.ultraPane != ultraPaneTree {
		t.Fatalf("expected ultra pane tree by default, got %v", m.ultraPane)
	}

	m.handleKey("tab")

	if m.ultraPane != ultraPanePreview || m.focus != focusPreview {
		t.Fatalf("expected ultra pane preview after tab, pane=%v focus=%v", m.ultraPane, m.focus)
	}

	m.handleKey("d")

	if !strings.Contains(m.status, "Actions disabled in ultra mode") {
		t.Fatalf("expected action-blocked status in ultra mode, got: %q", m.status)
	}
}

func TestWrapAndTruncateDisplayWidth(t *testing.T) {
	v := "这是一个非常非常长的中文字符串用于测试宽度处理"

	lines := wrapText(v, 12)
	for _, line := range lines {
		if got := displayWidthForTest(line); got > 12 {
			t.Fatalf("wrapped line exceeds width: %q width=%d", line, got)
		}
	}

	tr := truncateDisplay(v, 10)
	if got := displayWidthForTest(tr); got > 10 {
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

func TestPreviewForShowsFriendlyOversizeWarning(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")

	root := filepath.Join(workspace, "tmp")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir tmp: %v", err)
	}

	path := filepath.Join(root, "oversize-warning.jsonl")
	longText := strings.Repeat("x", 1024*1024+128)

	content := `{"type":"session_meta","payload":{"id":"x","timestamp":"2026-03-02T09:44:00.024Z"}}` + "\n" +
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"` + longText + `"}]}}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write preview fixture: %v", err)
	}

	m := tuiModel{
		previewCache: make(map[string][]string),
	}
	out := m.previewFor(path, 64, 20)

	joined := strings.Join(out, "\n")
	if !strings.Contains(stripANSIForTest(joined), "preview unavailable: a session entry exceeds the safe preview limit") {
		t.Fatalf("expected friendly oversize warning, got: %q", joined)
	}

	if strings.Contains(stripANSIForTest(joined), "token too long") {
		t.Fatalf("did not expect raw scanner error, got: %q", joined)
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
						if got := displayWidthForTest(stripANSIForTest(line)); got > width {
							t.Fatalf("preview line exceeds width=%d, got=%d line=%q", width, got, line)
						}
					}
				})
			}
		})
	}
}

func TestPreviewForUsesSessionStyleCacheKey(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")

	root := filepath.Join(workspace, "tmp")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir tmp: %v", err)
	}

	p := filepath.Join(root, "cache-key.jsonl")

	content := `{"type":"session_meta","payload":{"id":"x","timestamp":"2026-03-02T09:44:00.024Z"}}` + "\n" +
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"cache key check"}]}}` + "\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write preview fixture: %v", err)
	}

	info, err := os.Stat(p)
	if err != nil {
		t.Fatalf("stat preview fixture: %v", err)
	}

	m := tuiModel{previewCache: make(map[string][]string)}
	_ = m.previewFor(p, 24, 20)

	wantKey := preview.CacheKeyForSession(p, 24, info.Size(), info.ModTime().UnixNano())
	if _, ok := m.previewCache[wantKey]; !ok {
		t.Fatalf("expected preview cache key %q not found; keys=%v", wantKey, maps.Keys(m.previewCache))
	}
}

func TestClassifyAngleTag(t *testing.T) {
	tests := []struct {
		tag  string
		want preview.AngleTagTone
	}{
		{tag: "<turn_aborted>", want: preview.AngleTagToneDanger},
		{tag: "</turn_aborted>", want: preview.AngleTagToneDanger},
		{tag: "<environment_context>", want: preview.AngleTagToneSystem},
		{tag: "<collaboration_mode>", want: preview.AngleTagToneSystem},
		{tag: "<session_meta>", want: preview.AngleTagToneLifecycle},
		{tag: "<operation_done>", want: preview.AngleTagToneSuccess},
		{tag: "<generic_tag>", want: preview.AngleTagToneDefault},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.tag, func(t *testing.T) {
			if got := preview.ClassifyAngleTag(tt.tag); got != tt.want {
				t.Fatalf("preview.ClassifyAngleTag(%q)=%v, want=%v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestNormalizeTUIGroupBy(t *testing.T) {
	okCases := []string{"month", "day", "host", ""}
	for _, in := range okCases {
		got, err := normalizeTUIGroupBy(in)
		if err != nil {
			t.Fatalf("normalizeTUIGroupBy(%q) error: %v", in, err)
		}

		if in == "" && got != "host" {
			t.Fatalf("expected default host, got %q", got)
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
		{mode: "host", expectGroupNodes: true},
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
				if n.Kind == treeItemMonth {
					groupCount++
				}

				if n.Kind == treeItemSession {
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

func TestRebuildTreeHostGroupingDoesNotDuplicateHeaders(t *testing.T) {
	m := tuiModel{
		groupBy: "host",
		home:    "",
		sessions: []session.Session{
			{SessionID: "s1", UpdatedAt: time.Date(2026, 3, 5, 11, 0, 0, 0, time.Local), HostDir: "/host/a"},
			{SessionID: "s2", UpdatedAt: time.Date(2026, 3, 5, 10, 0, 0, 0, time.Local), HostDir: "/host/b"},
			{SessionID: "s3", UpdatedAt: time.Date(2026, 3, 5, 9, 0, 0, 0, time.Local), HostDir: "/host/a"},
			{SessionID: "s4", UpdatedAt: time.Date(2026, 3, 5, 8, 0, 0, 0, time.Local), HostDir: "/host/b"},
		},
	}
	m.rebuildTree()

	groupHeaderCount := 0
	seenHeaders := map[string]struct{}{}

	for _, item := range m.tree {
		if item.Kind != treeItemMonth {
			continue
		}

		groupHeaderCount++

		if _, exists := seenHeaders[item.Month]; exists {
			t.Fatalf("duplicate group header found for host %q", item.Month)
		}

		seenHeaders[item.Month] = struct{}{}
	}

	if groupHeaderCount != 2 {
		t.Fatalf("expected 2 host group headers, got %d", groupHeaderCount)
	}
}

func TestRebuildTreeRespectsCollapsedGroups(t *testing.T) {
	m := tuiModel{
		groupBy: "host",
		home:    "",
		sessions: []session.Session{
			{SessionID: "s1", UpdatedAt: time.Date(2026, 3, 5, 11, 0, 0, 0, time.Local), HostDir: "/host/a"},
			{SessionID: "s2", UpdatedAt: time.Date(2026, 3, 5, 10, 0, 0, 0, time.Local), HostDir: "/host/b"},
			{SessionID: "s3", UpdatedAt: time.Date(2026, 3, 5, 9, 0, 0, 0, time.Local), HostDir: "/host/a"},
		},
		collapsedGroups: map[string]bool{"/host/a": true},
	}
	m.rebuildTree()

	groupCount := 0
	sessionCount := 0
	hasCollapsedHeader := false

	for _, item := range m.tree {
		if item.Kind == treeItemMonth {
			groupCount++

			if item.Month == "/host/a" && strings.HasPrefix(item.Label, "▸ ") {
				hasCollapsedHeader = true
			}
		}

		if item.Kind == treeItemSession {
			sessionCount++

			if item.Month == "/host/a" {
				t.Fatalf("collapsed group should not render sessions: %+v", item)
			}
		}
	}

	if groupCount != 2 {
		t.Fatalf("expected 2 group headers, got %d", groupCount)
	}

	if sessionCount != 1 {
		t.Fatalf("expected only expanded-group sessions, got %d", sessionCount)
	}

	if !hasCollapsedHeader {
		t.Fatal("expected collapsed group header marker")
	}
}

func TestPreviewHostPath(t *testing.T) {
	tests := []struct {
		name  string
		host  string
		width int
	}{
		{name: "short_keep", host: "~/work/a", width: 24},
		{name: "home_tail", host: "~/ai-workspace/codex-session-manager", width: 20},
		{name: "abs_tail", host: "/var/db/repos/mystical-overlay/app-misc/spacedrive-bin", width: 20},
		{name: "very_narrow", host: "/very/long/path/with/many/segments", width: 8},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := previewHostPath(tc.host, tc.width)
			if got == "" {
				t.Fatal("previewHostPath returned empty")
			}

			if w := displayWidthForTest(got); w > tc.width {
				t.Fatalf("width overflow: got=%q width=%d limit=%d", got, w, tc.width)
			}
		})
	}
}

func TestTruncateMiddleDisplay(t *testing.T) {
	v := "/very/long/path/to/project/codex-session-manager"

	got := truncateMiddleDisplay(v, 20)
	if w := displayWidthForTest(got); w > 20 {
		t.Fatalf("truncateMiddleDisplay overflow: %q width=%d", got, w)
	}

	if !strings.Contains(got, "...") {
		t.Fatalf("expected middle ellipsis, got: %q", got)
	}
}

func TestRenderKeysLine(t *testing.T) {
	short := renderKeysLine(24, tuiTheme{Name: "tokyonight", Colors: cloneColorMap(builtinThemes["tokyonight"])})
	if w := displayWidthForTest(stripANSIForTest(short)); w > 24 {
		t.Fatalf("short keys line overflow: width=%d", w)
	}

	long := renderKeysLine(200, tuiTheme{Name: "tokyonight", Colors: cloneColorMap(builtinThemes["tokyonight"])})
	if !strings.Contains(stripANSIForTest(long), "[KEYS]") {
		t.Fatalf("expected KEYS header in long line, got: %q", long)
	}

	if !strings.Contains(stripANSIForTest(long), "z/Z fold") {
		t.Fatalf("expected fold key hint in long line, got: %q", long)
	}
}

func TestRenderKeysLineUsesAdaptiveVariants(t *testing.T) {
	theme := tuiTheme{Name: "tokyonight", Colors: cloneColorMap(builtinThemes["tokyonight"])}

	compact := stripANSIForTest(renderKeysLine(52, theme))
	if !strings.Contains(compact, "[KEYS]") || !strings.Contains(compact, "q quit") {
		t.Fatalf("expected compact keys variant, got: %q", compact)
	}

	if strings.Contains(compact, "Ctrl+d/u preview") {
		t.Fatalf("did not expect long keys variant in compact width, got: %q", compact)
	}

	for _, width := range []int{135, 136, 137, 138} {
		line := stripANSIForTest(renderKeysLine(width, theme))
		if got := displayWidthForTest(line); got > width {
			t.Fatalf("adaptive keys overflow at width=%d: got=%d line=%q", width, got, line)
		}
	}
}

func TestGroupKeyForSession(t *testing.T) {
	m := tuiModel{home: "/home/omega"}

	s := session.Session{
		SessionID: "x",
		UpdatedAt: time.Date(2026, 3, 5, 10, 0, 0, 0, time.Local),
		Health:    session.HealthOK,
		HostDir:   "/home/omega/work/project",
	}
	if got := m.groupKeyForSession(s, "month"); got == "" || !strings.Contains(got, "2026-03") {
		t.Fatalf("month group key unexpected: %q", got)
	}

	if got := m.groupKeyForSession(s, "day"); got == "" || !strings.Contains(got, "2026-03-05") {
		t.Fatalf("day group key unexpected: %q", got)
	}

	if got := m.groupKeyForSession(s, "host"); got == "" || !strings.Contains(got, "~/work/project") {
		t.Fatalf("host group key unexpected: %q", got)
	}
}

func TestTUIHandleKeyToggleGroupCollapse(t *testing.T) {
	m := tuiModel{
		sessions: []session.Session{
			{SessionID: "a1", UpdatedAt: time.Now(), HostDir: "/tmp/a"},
			{SessionID: "a2", UpdatedAt: time.Now().Add(-time.Minute), HostDir: "/tmp/a"},
			{SessionID: "b1", UpdatedAt: time.Now().Add(-2 * time.Minute), HostDir: "/tmp/b"},
		},
		collapsedGroups: make(map[string]bool),
		previewCache:    make(map[string][]string),
		focus:           focusTree,
		groupBy:         "host",
	}
	m.rebuildTree()

	beforeVisible := 0

	for _, item := range m.tree {
		if item.Kind == treeItemSession {
			beforeVisible++
		}
	}

	m.handleKey("z")

	if !m.collapsedGroups["/tmp/a"] {
		t.Fatalf("expected /tmp/a to be collapsed, got %#v", m.collapsedGroups)
	}

	afterVisible := 0

	for _, item := range m.tree {
		if item.Kind == treeItemSession {
			afterVisible++
		}
	}

	if afterVisible >= beforeVisible {
		t.Fatalf("expected fewer visible sessions after collapse: before=%d after=%d", beforeVisible, afterVisible)
	}

	m.handleKey("Z")

	if len(m.collapsedGroups) != 0 {
		t.Fatalf("expected all groups expanded, got %#v", m.collapsedGroups)
	}

	expandedVisible := 0

	for _, item := range m.tree {
		if item.Kind == treeItemSession {
			expandedVisible++
		}
	}

	if expandedVisible != beforeVisible {
		t.Fatalf("expected visible session count restored: before=%d expanded=%d", beforeVisible, expandedVisible)
	}
}

func TestTUIHandleKeyCanFocusGroupRows(t *testing.T) {
	m := tuiModel{
		sessions: []session.Session{
			{SessionID: "a1", UpdatedAt: time.Now(), HostDir: "/tmp/a"},
			{SessionID: "a2", UpdatedAt: time.Now().Add(-time.Minute), HostDir: "/tmp/a"},
		},
		previewCache: make(map[string][]string),
		focus:        focusTree,
		groupBy:      "host",
	}
	m.rebuildTree()

	if m.cursor <= 0 {
		t.Fatalf("expected initial cursor at session row, got %d", m.cursor)
	}

	m.handleKey("k")

	if m.cursor != 0 {
		t.Fatalf("expected cursor to move to group row, got %d", m.cursor)
	}

	if m.tree[m.cursor].Kind != treeItemMonth {
		t.Fatalf("expected group row kind at cursor, got %+v", m.tree[m.cursor])
	}

	m.handleKey("j")

	if m.tree[m.cursor].Kind != treeItemSession {
		t.Fatalf("expected cursor to move back to session row, got %+v", m.tree[m.cursor])
	}
}

func TestTUIHandleKeyToggleGroupCollapseOnGroupRow(t *testing.T) {
	m := tuiModel{
		sessions: []session.Session{
			{SessionID: "a1", UpdatedAt: time.Now(), HostDir: "/tmp/a"},
			{SessionID: "a2", UpdatedAt: time.Now().Add(-time.Minute), HostDir: "/tmp/a"},
			{SessionID: "b1", UpdatedAt: time.Now().Add(-2 * time.Minute), HostDir: "/tmp/b"},
		},
		collapsedGroups: make(map[string]bool),
		previewCache:    make(map[string][]string),
		focus:           focusTree,
		groupBy:         "host",
	}
	m.rebuildTree()

	m.handleKey("k")

	if m.cursor != 0 || m.tree[m.cursor].Kind != treeItemMonth {
		t.Fatalf("expected cursor on first group row, got cursor=%d item=%+v", m.cursor, m.tree[m.cursor])
	}

	group := m.tree[m.cursor].Month
	m.handleKey("z")

	if !m.collapsedGroups[group] {
		t.Fatalf("expected group %q to be collapsed, got %#v", group, m.collapsedGroups)
	}

	if m.cursor != 0 || m.tree[m.cursor].Kind != treeItemMonth || m.tree[m.cursor].Month != group {
		t.Fatalf("expected cursor to stay on collapsed group row, got cursor=%d item=%+v", m.cursor, m.tree[m.cursor])
	}
}

func TestDetailRowsColorsHealthValue(t *testing.T) {
	m := tuiModel{width: 120, home: "/home/omega"}
	okSession := session.Session{
		SessionID: "ok-id",
		UpdatedAt: time.Date(2026, 3, 8, 10, 0, 0, 0, time.Local),
		Health:    session.HealthOK,
		HostDir:   "/home/omega/work/project",
	}
	badSession := session.Session{
		SessionID: "bad-id",
		UpdatedAt: okSession.UpdatedAt,
		Health:    session.HealthCorrupted,
		HostDir:   okSession.HostDir,
	}

	_, okRow := m.detailRows(okSession, 96)

	_, badRow := m.detailRows(badSession, 96)
	if !strings.Contains(okRow, "OK") || !strings.Contains(badRow, "CORRUPTED") {
		t.Fatalf("expected health text in rows, got ok=%q bad=%q", okRow, badRow)
	}

	if got := m.healthColorHex(session.HealthOK); got != m.colorHex("status_ok") {
		t.Fatalf("healthColorHex(ok) mismatch: %q", got)
	}

	if got := m.healthColorHex(session.HealthCorrupted); got != m.colorHex("status_risk") {
		t.Fatalf("healthColorHex(corrupted) mismatch: %q", got)
	}
}

func TestDetailRowsUsesCompactColumnsWhenNarrow(t *testing.T) {
	m := tuiModel{width: 120, home: "/home/omega"}
	selected := session.Session{
		SessionID: "ok-id",
		UpdatedAt: time.Date(2026, 3, 8, 10, 0, 0, 0, time.Local),
		Health:    session.HealthOK,
		HostDir:   "/home/omega/work/project",
	}

	header, values := m.detailRows(selected, 52)
	cleanHeader := stripANSIForTest(header)
	cleanValues := stripANSIForTest(values)

	if strings.Contains(cleanHeader, "UPDATED") || strings.Contains(cleanHeader, "SIZE") {
		t.Fatalf("expected compact info header, got: %q", cleanHeader)
	}

	if !strings.Contains(cleanHeader, "ID") || !strings.Contains(cleanHeader, "HOST") {
		t.Fatalf("expected ID/HOST in compact header, got: %q", cleanHeader)
	}

	if !strings.Contains(cleanValues, "ok-id") && !strings.Contains(cleanValues, "ok") {
		t.Fatalf("expected compact values content, got: %q", cleanValues)
	}
}

func TestTUIViewAndHelpersCoverage(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")

	root := filepath.Join(workspace, "tmp")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir tmp: %v", err)
	}

	p := filepath.Join(root, "view.jsonl")

	content := `{"type":"session_meta","payload":{"id":"x","timestamp":"2026-03-02T09:44:00.024Z"}}` + "\n" +
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"hello <environment_context> world"}]}}` + "\n" +
		`{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok <turn_aborted> done"}]}}` + "\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write preview fixture: %v", err)
	}

	m := tuiModel{
		sessions: []session.Session{
			{
				SessionID: "019c-test-id",
				UpdatedAt: time.Date(2026, 3, 5, 12, 0, 0, 0, time.Local),
				Health:    session.HealthOK,
				HostDir:   "/home/omega/work/project",
				Path:      p,
			},
		},
		width:        136,
		height:       36,
		home:         "/home/omega",
		sessionsRoot: filepath.Join(workspace, "sessions"),
		previewCache: make(map[string][]string),
		groupBy:      "host",
		focus:        focusTree,
	}
	m.rebuildTree()

	out := m.View()
	if !strings.Contains(out, "SESSIONS") || !strings.Contains(out, "PREVIEW") {
		t.Fatalf("unexpected view output: %q", out)
	}

	if !strings.Contains(stripANSIForTest(out), "[KEYS]") {
		t.Fatalf("expected keys bar in view: %q", out)
	}

	// Helper coverage paths.
	if got := buildPreviewScrollBar(0, 1, 10, 16); got == "" {
		t.Fatal("empty preview scroll bar")
	}

	if got := fitCell("abc", 8); displayWidthForTest(got) != 8 {
		t.Fatalf("fitCell width unexpected: %q", got)
	}

	if got := fitCellMiddle("/very/long/path/to/project", 12); displayWidthForTest(got) != 12 {
		t.Fatalf("fitCellMiddle width unexpected: %q", got)
	}

	if step := m.previewPageStep(); step <= 0 {
		t.Fatalf("previewPageStep should be positive, got %d", step)
	}

	if !usecase.IsPreviewNoiseText("filesystem sandboxing note") {
		t.Fatal("expected preview noise detection")
	}
}

func TestTUIViewKeysBarWidthMatchesMainArea(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	sessionsRoot := filepath.Join(workspace, "sessions")

	sessions, err := scanner.ScanSessions(sessionsRoot)
	if err != nil {
		t.Fatalf("load sessions: %v", err)
	}

	if len(sessions) == 0 {
		t.Fatal("expected rich fixture sessions")
	}

	m := tuiModel{
		sessions:     sessions,
		width:        136,
		height:       36,
		home:         "/home/omega",
		sessionsRoot: sessionsRoot,
		previewCache: make(map[string][]string),
		groupBy:      "host",
		focus:        focusTree,
	}
	m.rebuildTree()
	out := stripANSIForTest(m.View())
	lines := strings.Split(out, "\n")

	keysIdx := -1

	for i := range lines {
		if strings.Contains(lines[i], "[KEYS]") {
			keysIdx = i
			break
		}
	}

	if keysIdx <= 1 || keysIdx+1 >= len(lines) {
		t.Fatalf("keys bar lines not found in expected location, idx=%d total=%d", keysIdx, len(lines))
	}

	mainWidth := 0
	for i := 0; i < keysIdx-1; i++ {
		if w := displayWidthForTest(lines[i]); w > mainWidth {
			mainWidth = w
		}
	}

	keysTopW := displayWidthForTest(lines[keysIdx-1])
	keysMidW := displayWidthForTest(lines[keysIdx])

	keysBotW := displayWidthForTest(lines[keysIdx+1])
	if keysTopW != keysMidW || keysMidW != keysBotW {
		t.Fatalf("keys bar width mismatch: top=%d mid=%d bot=%d", keysTopW, keysMidW, keysBotW)
	}

	if keysMidW != mainWidth {
		t.Fatalf("keys bar width (%d) must match main area width (%d)", keysMidW, mainWidth)
	}
}

func TestTUIViewShowsPendingConfirmInKeysBar(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	sessionsRoot := filepath.Join(workspace, "sessions")

	sessions, err := scanner.ScanSessions(sessionsRoot)
	if err != nil {
		t.Fatalf("scan sessions: %v", err)
	}

	if len(sessions) == 0 {
		t.Fatal("expected rich fixture sessions")
	}

	m := tuiModel{
		sessions:      sessions,
		width:         136,
		height:        32,
		previewCache:  make(map[string][]string),
		groupBy:       "host",
		focus:         focusTree,
		pendingAction: "delete",
		pendingID:     sessions[0].SessionID,
	}
	m.rebuildTree()

	out := stripANSIForTest(m.View())
	if !strings.Contains(out, "PENDING DELETE") {
		t.Fatalf("expected pending banner in keys bar, got: %q", out)
	}

	if !strings.Contains(out, "Press Y to confirm, N to cancel") {
		t.Fatalf("expected confirm hint in keys bar, got: %q", out)
	}
}

func TestTUIViewShowsPendingGroupDeleteInKeysBar(t *testing.T) {
	m := tuiModel{
		width:         136,
		height:        32,
		previewCache:  make(map[string][]string),
		groupBy:       "host",
		focus:         focusTree,
		pendingAction: "delete-group",
		pendingGroup:  "/tmp/group-a",
		pendingCount:  4,
		pendingStep:   2,
	}

	out := stripANSIForTest(m.View())
	if !strings.Contains(out, "PENDING DELETE GROUP 2/3") {
		t.Fatalf("expected group pending banner in keys bar, got: %q", out)
	}

	if !strings.Contains(out, "sessions=4") {
		t.Fatalf("expected group session count in keys bar, got: %q", out)
	}
}

func TestTUIUpdateAndDryRunPreview(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	sessionsRoot := filepath.Join(workspace, "sessions")
	m := tuiModel{
		sessions: []session.Session{
			{
				SessionID: "11111111-1111-1111-1111-111111111111",
				UpdatedAt: time.Date(2026, 3, 2, 10, 0, 0, 0, time.Local),
				Health:    session.HealthOK,
				HostDir:   "/tmp/host",
				Path:      filepath.Join(sessionsRoot, "2026", "03", "02", "rollout-delete-dry.jsonl"),
			},
		},
		previewCache: make(map[string][]string),
		sessionsRoot: sessionsRoot,
	}
	m.rebuildTree()

	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	updated := model.(tuiModel)
	if updated.width != 100 || updated.height != 30 {
		t.Fatalf("window size not updated: %+v", updated)
	}

	updated.runDryRunPreview()

	if !strings.Contains(updated.status, "delete: action=dry-run") {
		t.Fatalf("unexpected dry-run status: %q", updated.status)
	}

	if _, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}); cmd == nil {
		t.Fatal("expected quit command on q")
	}
}

func TestResolveTUITheme(t *testing.T) {
	theme, err := resolveTUITheme("tokyonight", map[string]string{"keys_label": "#ffffff"}, "catppuccin", []string{"keys_key=#123456"})
	if err != nil {
		t.Fatalf("resolve theme: %v", err)
	}

	if theme.Name != "catppuccin" {
		t.Fatalf("theme name mismatch: %q", theme.Name)
	}

	if got := theme.hex("keys_label", ""); got != "#ffffff" {
		t.Fatalf("config override not applied: %q", got)
	}

	if got := theme.hex("keys_key", ""); got != "#123456" {
		t.Fatalf("flag override not applied: %q", got)
	}
}

func TestResolveTUIThemeInvalidOverride(t *testing.T) {
	if _, err := resolveTUITheme("", nil, "", []string{"broken"}); err == nil {
		t.Fatal("expected invalid theme override error")
	}
}

func TestTUIRequestDeletePendingAndConfirm(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	sessionsRoot := filepath.Join(workspace, "sessions")
	trashRoot := filepath.Join(workspace, "trash")
	logFile := filepath.Join(workspace, "logs", "actions.log")
	target := filepath.Join(workspace, "sessions", "2026", "03", "02", "rollout-delete-dry.jsonl")

	m := tuiModel{
		sessions: []session.Session{{
			SessionID: "11111111-1111-1111-1111-111111111111",
			Path:      target,
			UpdatedAt: time.Now(),
			SizeBytes: 1,
			Health:    session.HealthOK,
		}},
		sessionsRoot: sessionsRoot,
		trashRoot:    trashRoot,
		logFile:      logFile,
		dryRun:       false,
		confirm:      true,
		yes:          false,
		maxBatch:     10,
		source:       "sessions",
		previewCache: map[string][]string{},
	}
	m.rebuildTree()
	m.requestDelete()

	if m.pendingAction != "delete" {
		t.Fatalf("expected pending delete, got %q", m.pendingAction)
	}

	m.commitPendingAction()

	if m.pendingAction != "" {
		t.Fatalf("pending action not cleared: %q", m.pendingAction)
	}

	if !strings.Contains(m.status, "delete: action=") {
		t.Fatalf("unexpected status: %q", m.status)
	}
}

func TestTUIRequestDeleteGroupDryRunFromHeader(t *testing.T) {
	now := time.Now()
	m := tuiModel{
		groupBy: "host",
		sessions: []session.Session{
			{SessionID: "s1", Path: filepath.Join(t.TempDir(), "s1.jsonl"), UpdatedAt: now, HostDir: "/tmp/group-a", SizeBytes: 10},
			{SessionID: "s2", Path: filepath.Join(t.TempDir(), "s2.jsonl"), UpdatedAt: now.Add(-time.Minute), HostDir: "/tmp/group-a", SizeBytes: 20},
			{SessionID: "s3", Path: filepath.Join(t.TempDir(), "s3.jsonl"), UpdatedAt: now.Add(-2 * time.Minute), HostDir: "/tmp/group-b", SizeBytes: 30},
		},
		dryRun:       true,
		confirm:      true,
		source:       "sessions",
		maxBatch:     10,
		previewCache: map[string][]string{},
	}
	m.rebuildTree()

	if idx, ok := m.findGroupIndex("/tmp/group-a"); ok {
		m.cursor = idx
	}

	m.requestDelete()

	if !strings.Contains(m.status, "delete-group: action=dry-run matched=2") {
		t.Fatalf("unexpected status: %q", m.status)
	}
}

func TestTUIHandleKeyDeleteRunsGroupDeleteFromHeader(t *testing.T) {
	now := time.Now()
	m := tuiModel{
		groupBy: "host",
		sessions: []session.Session{
			{SessionID: "s1", Path: filepath.Join(t.TempDir(), "s1.jsonl"), UpdatedAt: now, HostDir: "/tmp/group-a", SizeBytes: 10},
			{SessionID: "s2", Path: filepath.Join(t.TempDir(), "s2.jsonl"), UpdatedAt: now.Add(-time.Minute), HostDir: "/tmp/group-a", SizeBytes: 20},
		},
		dryRun:       true,
		confirm:      true,
		source:       "sessions",
		maxBatch:     10,
		previewCache: map[string][]string{},
	}
	m.rebuildTree()

	if idx, ok := m.findGroupIndex("/tmp/group-a"); ok {
		m.cursor = idx
	}

	m.handleKey("d")

	if !strings.Contains(m.status, "delete-group: action=dry-run matched=2") {
		t.Fatalf("unexpected status after handleKey: %q", m.status)
	}
}

func TestTUIRequestDeleteGroupRequiresThreeConfirms(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	sessionsRoot := filepath.Join(workspace, "sessions")
	trashRoot := filepath.Join(workspace, "trash")
	logFile := filepath.Join(workspace, "logs", "actions.log")
	now := time.Now()

	targetA := filepath.Join(workspace, "sessions", "2026", "03", "02", "rollout-delete-dry.jsonl")
	targetB := filepath.Join(workspace, "sessions", "2026", "03", "02", "rollout-delete-confirm.jsonl")
	targetC := filepath.Join(workspace, "sessions", "2026", "03", "02", "rollout-delete-hard.jsonl")

	m := tuiModel{
		groupBy: "host",
		sessions: []session.Session{
			{SessionID: "s1", Path: targetA, UpdatedAt: now, HostDir: "/tmp/group-a", SizeBytes: 1},
			{SessionID: "s2", Path: targetB, UpdatedAt: now.Add(-time.Minute), HostDir: "/tmp/group-a", SizeBytes: 1},
			{SessionID: "s3", Path: targetC, UpdatedAt: now.Add(-2 * time.Minute), HostDir: "/tmp/group-b", SizeBytes: 1},
		},
		sessionsRoot: sessionsRoot,
		trashRoot:    trashRoot,
		logFile:      logFile,
		dryRun:       false,
		confirm:      true,
		yes:          false,
		source:       "sessions",
		maxBatch:     10,
		previewCache: map[string][]string{},
	}
	m.rebuildTree()

	if idx, ok := m.findGroupIndex("/tmp/group-a"); ok {
		m.cursor = idx
	}

	m.requestDelete()

	if m.pendingAction != "delete-group" || m.pendingStep != 1 || m.pendingGroup != "/tmp/group-a" {
		t.Fatalf("unexpected pending group delete state: action=%q step=%d group=%q", m.pendingAction, m.pendingStep, m.pendingGroup)
	}

	m.commitPendingAction()

	if m.pendingStep != 2 {
		t.Fatalf("expected second confirm step, got %d", m.pendingStep)
	}

	if _, err := os.Stat(targetA); err != nil {
		t.Fatalf("expected targetA to remain before final confirm: %v", err)
	}

	m.commitPendingAction()

	if m.pendingStep != 3 {
		t.Fatalf("expected third confirm step, got %d", m.pendingStep)
	}

	if _, err := os.Stat(targetB); err != nil {
		t.Fatalf("expected targetB to remain before final confirm: %v", err)
	}

	m.commitPendingAction()

	if m.pendingAction != "" || m.pendingStep != 0 || m.pendingGroup != "" {
		t.Fatalf("expected pending state cleared after execution: action=%q step=%d group=%q", m.pendingAction, m.pendingStep, m.pendingGroup)
	}

	if len(m.sessions) != 1 || m.sessions[0].SessionID != "s3" {
		t.Fatalf("expected group sessions removed after delete, sessions=%+v", m.sessions)
	}

	if !strings.Contains(m.status, "delete-group: action=soft-delete matched=2") {
		t.Fatalf("unexpected status: %q", m.status)
	}
}

func TestTUICommitPendingGroupDeleteCancelsWhenGroupChanges(t *testing.T) {
	now := time.Now()
	m := tuiModel{
		groupBy: "host",
		sessions: []session.Session{
			{SessionID: "s1", UpdatedAt: now, HostDir: "/tmp/group-a"},
			{SessionID: "s2", UpdatedAt: now.Add(-time.Minute), HostDir: "/tmp/group-b"},
		},
		pendingAction: "delete-group",
		pendingGroup:  "/tmp/group-a",
		pendingCount:  1,
		pendingStep:   1,
		previewCache:  map[string][]string{},
	}
	m.rebuildTree()

	if idx, ok := m.findGroupIndex("/tmp/group-b"); ok {
		m.cursor = idx
	}

	m.commitPendingAction()

	if m.pendingAction != "" || m.pendingGroup != "" || m.pendingStep != 0 {
		t.Fatalf("pending group state not cleared: action=%q group=%q step=%d", m.pendingAction, m.pendingGroup, m.pendingStep)
	}

	if !strings.Contains(m.status, "group changed") {
		t.Fatalf("unexpected status: %q", m.status)
	}
}

func TestRemoveSelectedSessionKeepsCursorAndMovesToNext(t *testing.T) {
	now := time.Now()
	m := tuiModel{
		groupBy: "host",
		sessions: []session.Session{
			{SessionID: "s1", UpdatedAt: now, HostDir: "/tmp/h"},
			{SessionID: "s2", UpdatedAt: now.Add(-time.Minute), HostDir: "/tmp/h"},
			{SessionID: "s3", UpdatedAt: now.Add(-2 * time.Minute), HostDir: "/tmp/h"},
		},
		previewCache: map[string][]string{},
	}
	m.rebuildTree()

	findCursorByID := func(id string) int {
		for i, item := range m.tree {
			if item.Kind != treeItemSession || item.Index < 0 || item.Index >= len(m.sessions) {
				continue
			}

			if m.sessions[item.Index].SessionID == id {
				return i
			}
		}

		return -1
	}

	cur := findCursorByID("s2")
	if cur < 0 {
		t.Fatal("failed to locate s2 in tree")
	}

	m.cursor = cur
	m.removeSelectedSession()

	got, ok := m.selectedSession()
	if !ok {
		t.Fatal("expected selected session after delete")
	}

	if got.SessionID != "s3" {
		t.Fatalf("expected cursor to move to next session s3, got %q", got.SessionID)
	}
}

func TestTUIRequestHostMigrateDryRunMissingHost(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	sessionsRoot := filepath.Join(workspace, "sessions")
	missingHost := filepath.Join(workspace, "missing-host")
	otherHost := t.TempDir()

	m := tuiModel{
		sessions: []session.Session{
			{SessionID: "s1", Path: filepath.Join(workspace, "sessions", "2026", "03", "02", "rollout-delete-dry.jsonl"), UpdatedAt: time.Now(), HostDir: missingHost},
			{SessionID: "s2", Path: filepath.Join(workspace, "sessions", "2026", "03", "02", "rollout-delete-prefix-1.jsonl"), UpdatedAt: time.Now().Add(-time.Minute), HostDir: missingHost},
			{SessionID: "s3", Path: filepath.Join(workspace, "sessions", "2026", "03", "02", "rollout-delete-prefix-2.jsonl"), UpdatedAt: time.Now().Add(-2 * time.Minute), HostDir: otherHost},
		},
		sessionsRoot: sessionsRoot,
		trashRoot:    filepath.Join(workspace, "trash"),
		dryRun:       true,
		source:       "sessions",
		maxBatch:     10,
		previewCache: map[string][]string{},
	}
	m.rebuildTree()
	m.requestHostMigrate()

	if !strings.Contains(m.status, "migrate-host: action=dry-run matched=2") {
		t.Fatalf("unexpected status: %q", m.status)
	}
}

func TestTUIRequestHostMigrateRejectsExistingHost(t *testing.T) {
	host := t.TempDir()
	m := tuiModel{
		sessions: []session.Session{
			{SessionID: "s1", Path: filepath.Join(t.TempDir(), "noop.jsonl"), UpdatedAt: time.Now(), HostDir: host},
		},
		dryRun:       true,
		source:       "sessions",
		maxBatch:     10,
		previewCache: map[string][]string{},
	}
	m.rebuildTree()
	m.requestHostMigrate()

	if !strings.Contains(m.status, "Selected host path exists") {
		t.Fatalf("unexpected status: %q", m.status)
	}
}

func TestTUIRequestRestoreGuardPaths(t *testing.T) {
	t.Run("wrong source", func(t *testing.T) {
		m := tuiModel{
			sessions: []session.Session{{SessionID: "s1", UpdatedAt: time.Now()}},
			source:   "sessions",
		}
		m.rebuildTree()
		m.requestRestore()

		if !strings.Contains(m.status, "Current source is sessions") {
			t.Fatalf("unexpected status: %q", m.status)
		}
	})

	t.Run("no selection", func(t *testing.T) {
		m := tuiModel{source: "trash"}
		m.requestRestore()

		if !strings.Contains(m.status, "No session selected") {
			t.Fatalf("unexpected status: %q", m.status)
		}
	})

	t.Run("requires confirm", func(t *testing.T) {
		m := tuiModel{
			sessions: []session.Session{{SessionID: "s1", UpdatedAt: time.Now()}},
			source:   "trash",
			dryRun:   false,
			confirm:  false,
		}
		m.rebuildTree()
		m.requestRestore()

		if !strings.Contains(m.status, "Real restore requires --confirm") {
			t.Fatalf("unexpected status: %q", m.status)
		}
	})

	t.Run("sets pending action", func(t *testing.T) {
		m := tuiModel{
			sessions: []session.Session{{SessionID: "s1", UpdatedAt: time.Now()}},
			source:   "trash",
			dryRun:   false,
			confirm:  true,
			yes:      false,
		}
		m.rebuildTree()
		m.requestRestore()

		if m.pendingAction != "restore" || m.pendingID != "s1" {
			t.Fatalf("unexpected pending restore state: action=%q id=%q", m.pendingAction, m.pendingID)
		}

		if !strings.Contains(m.status, "Confirm restore") {
			t.Fatalf("unexpected status: %q", m.status)
		}
	})
}

func TestTUIRequestRestoreDryRunUpdatesStatus(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	trashSessionsRoot := filepath.Join(workspace, "trash", "sessions")

	items, err := scanner.ScanSessions(trashSessionsRoot)
	if err != nil {
		t.Fatalf("scan trash sessions: %v", err)
	}

	if len(items) == 0 {
		t.Fatal("expected trash fixture sessions")
	}

	m := tuiModel{
		sessions:     []session.Session{items[0]},
		source:       "trash",
		dryRun:       true,
		confirm:      true,
		yes:          true,
		maxBatch:     10,
		sessionsRoot: filepath.Join(workspace, "sessions"),
		trashRoot:    filepath.Join(workspace, "trash"),
		logFile:      filepath.Join(workspace, "logs", "actions.log"),
		previewCache: map[string][]string{},
	}
	m.rebuildTree()
	m.requestRestore()

	if !strings.Contains(m.status, "restore: action=restore-dry-run matched=1") {
		t.Fatalf("unexpected status: %q", m.status)
	}
}

func TestTUICommitPendingActionCancelsWhenSelectionChanges(t *testing.T) {
	now := time.Now()
	m := tuiModel{
		sessions: []session.Session{
			{SessionID: "s1", UpdatedAt: now},
			{SessionID: "s2", UpdatedAt: now.Add(-time.Minute)},
		},
		pendingAction: "delete",
		pendingID:     "s1",
		previewCache:  map[string][]string{},
	}
	m.rebuildTree()
	m.cursor = len(m.tree) - 1
	m.commitPendingAction()

	if m.pendingAction != "" || m.pendingID != "" {
		t.Fatalf("pending state not cleared: action=%q id=%q", m.pendingAction, m.pendingID)
	}

	if !strings.Contains(m.status, "selection changed") {
		t.Fatalf("unexpected status: %q", m.status)
	}
}

func TestTUICommitPendingHostMigrateCancelsWhenHostMissing(t *testing.T) {
	m := tuiModel{
		sessions:      []session.Session{{SessionID: "s1", UpdatedAt: time.Now()}},
		pendingAction: "migrate-host",
		pendingID:     "s1",
		previewCache:  map[string][]string{},
	}
	m.rebuildTree()
	m.commitPendingAction()

	if m.pendingAction != "" || m.pendingID != "" {
		t.Fatalf("pending state not cleared: action=%q id=%q", m.pendingAction, m.pendingID)
	}

	if !strings.Contains(m.status, "host missing") {
		t.Fatalf("unexpected status: %q", m.status)
	}
}

func TestTUICancelPendingActionClearsState(t *testing.T) {
	m := tuiModel{
		pendingAction: "restore",
		pendingStep:   2,
		pendingID:     "s1",
		pendingHost:   "/tmp/missing",
		pendingGroup:  "/tmp/group-a",
		pendingCount:  3,
	}
	m.cancelPendingAction()

	if m.pendingAction != "" || m.pendingID != "" || m.pendingHost != "" || m.pendingGroup != "" || m.pendingCount != 0 || m.pendingStep != 0 {
		t.Fatalf(
			"pending state not cleared: action=%q step=%d id=%q host=%q group=%q count=%d",
			m.pendingAction,
			m.pendingStep,
			m.pendingID,
			m.pendingHost,
			m.pendingGroup,
			m.pendingCount,
		)
	}

	if !strings.Contains(m.status, "Pending action cancelled") {
		t.Fatalf("unexpected status: %q", m.status)
	}
}

func TestRenderTreeLinesMarksHealthAndColorizedNames(t *testing.T) {
	missingHost := filepath.Join(t.TempDir(), "missing-host")
	m := tuiModel{
		groupBy: "none",
		sessions: []session.Session{
			{SessionID: "ok", UpdatedAt: time.Now(), Health: session.HealthOK, HostDir: missingHost},
			{SessionID: "warn", UpdatedAt: time.Now().Add(-time.Minute), Health: session.HealthMissingMeta},
			{SessionID: "err", UpdatedAt: time.Now().Add(-2 * time.Minute), Health: session.HealthCorrupted},
		},
		previewCache: map[string][]string{},
	}
	m.rebuildTree()
	lines := m.renderTreeLines(80, "#999999")
	raw := strings.Join(lines, "\n")

	joined := stripANSIForTest(raw)
	if !strings.Contains(joined, "! ok") {
		t.Fatalf("expected host-missing marker in tree lines: %q", joined)
	}

	if !strings.Contains(joined, "! warn") {
		t.Fatalf("expected unhealthy marker in tree lines: %q", joined)
	}

	if !strings.Contains(joined, "⚠ err") {
		t.Fatalf("expected error marker in tree lines: %q", joined)
	}

	if sym, color := m.healthSymbolAndColor(session.HealthMissingMeta); sym != "!" || color != m.colorHex("status_warn") {
		t.Fatalf("unexpected missing-meta visual: sym=%q color=%q", sym, color)
	}

	if sym, color := m.healthSymbolAndColor(session.HealthCorrupted); sym != "✖" || color != m.colorHex("status_risk") {
		t.Fatalf("unexpected corrupted visual: sym=%q color=%q", sym, color)
	}

	if sym, color, nonHealthy := m.treeHealthVisual(session.HealthOK, true); sym != "!" || color != m.colorHex("status_warn") || !nonHealthy {
		t.Fatalf("unexpected host-missing visual: sym=%q color=%q nonHealthy=%v", sym, color, nonHealthy)
	}

	if sym, color, nonHealthy := m.treeHealthVisual(session.HealthCorrupted, false); sym != "⚠" || color != m.colorHex("status_risk") || !nonHealthy {
		t.Fatalf("unexpected corrupted risk visual: sym=%q color=%q nonHealthy=%v", sym, color, nonHealthy)
	}
}

func TestRenderTreeLinesShowsPositionFooter(t *testing.T) {
	tmp := t.TempDir()
	hostA := filepath.Join(tmp, "a")
	hostB := filepath.Join(tmp, "b")

	if err := os.MkdirAll(hostA, 0o755); err != nil {
		t.Fatalf("mkdir hostA: %v", err)
	}

	if err := os.MkdirAll(hostB, 0o755); err != nil {
		t.Fatalf("mkdir hostB: %v", err)
	}

	m := tuiModel{
		groupBy: "host",
		sessions: []session.Session{
			{SessionID: "s1", UpdatedAt: time.Now(), HostDir: hostA, Health: session.HealthOK},
			{SessionID: "s2", UpdatedAt: time.Now().Add(-time.Minute), HostDir: hostA, Health: session.HealthMissingMeta},
			{SessionID: "s3", UpdatedAt: time.Now().Add(-2 * time.Minute), HostDir: hostB, Health: session.HealthCorrupted},
		},
		previewCache: map[string][]string{},
	}
	m.rebuildTree()

	content := m.renderTreePaneContent(60, 8, "#999999")

	lines := strings.Split(content, "\n")
	if len(lines) != 8 {
		t.Fatalf("expected fixed pane height lines=8, got %d", len(lines))
	}

	last := stripANSIForTest(lines[len(lines)-1])
	if !strings.Contains(last, "1/3") || !strings.Contains(last, "WARN: 2") || !strings.Contains(last, "RISK: 1") {
		t.Fatalf("expected footer in bottom line, got: %q", last)
	}

	m.cursor++
	m.skipToSelectable(1)

	content = m.renderTreePaneContent(60, 8, "#999999")
	lines = strings.Split(content, "\n")

	last = stripANSIForTest(lines[len(lines)-1])
	if !strings.Contains(last, "2/3") || !strings.Contains(last, "WARN: 2") || !strings.Contains(last, "RISK: 1") {
		t.Fatalf("expected updated footer after cursor move, got: %q", last)
	}
}

func TestRenderTreeLinesFooterCountsHostMissingAsWarn(t *testing.T) {
	tmp := t.TempDir()
	missingHost := filepath.Join(tmp, "missing-host")

	m := tuiModel{
		groupBy: "host",
		sessions: []session.Session{
			{SessionID: "s1", UpdatedAt: time.Now(), HostDir: missingHost, Health: session.HealthOK},
			{SessionID: "s2", UpdatedAt: time.Now().Add(-time.Minute), HostDir: tmp, Health: session.HealthOK},
		},
		previewCache: map[string][]string{},
	}
	m.rebuildTree()

	content := m.renderTreePaneContent(60, 8, "#999999")
	lines := strings.Split(content, "\n")

	last := stripANSIForTest(lines[len(lines)-1])
	if !strings.Contains(last, "1/2") || !strings.Contains(last, "WARN: 1") || !strings.Contains(last, "RISK: 0") {
		t.Fatalf("expected host-missing to be counted as warning, got: %q", last)
	}
}

func TestSortTUISessions_PrioritizesRisk(t *testing.T) {
	now := time.Now()
	items := []session.Session{
		{SessionID: "ok-new", UpdatedAt: now, Health: session.HealthOK},
		{SessionID: "missing-old", UpdatedAt: now.Add(-2 * time.Hour), Health: session.HealthMissingMeta},
		{SessionID: "corrupted-old", UpdatedAt: now.Add(-4 * time.Hour), Health: session.HealthCorrupted},
	}
	core.SortSessionsByRisk(items, nil, nil)

	if items[0].Health != session.HealthCorrupted {
		t.Fatalf("expected corrupted first, got %#v", items)
	}

	if items[1].Health != session.HealthMissingMeta {
		t.Fatalf("expected missing-meta second, got %#v", items)
	}
}
