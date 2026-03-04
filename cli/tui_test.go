package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
	"github.com/MysticalDevil/codexsm/session"
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
			if w := runewidth.StringWidth(got); w > tc.width {
				t.Fatalf("width overflow: got=%q width=%d limit=%d", got, w, tc.width)
			}
		})
	}
}

func TestTruncateMiddleDisplay(t *testing.T) {
	v := "/very/long/path/to/project/codex-session-manager"
	got := truncateMiddleDisplay(v, 20)
	if w := runewidth.StringWidth(got); w > 20 {
		t.Fatalf("truncateMiddleDisplay overflow: %q width=%d", got, w)
	}
	if !strings.Contains(got, "...") {
		t.Fatalf("expected middle ellipsis, got: %q", got)
	}
}

func TestRenderKeysLine(t *testing.T) {
	short := renderKeysLine(24, tuiTheme{Name: "tokyonight", Colors: cloneColorMap(builtinThemes["tokyonight"])})
	if w := runewidth.StringWidth(stripANSIForTest(short)); w > 24 {
		t.Fatalf("short keys line overflow: width=%d", w)
	}
	long := renderKeysLine(200, tuiTheme{Name: "tokyonight", Colors: cloneColorMap(builtinThemes["tokyonight"])})
	if !strings.Contains(stripANSIForTest(long), "[KEYS]") {
		t.Fatalf("expected KEYS header in long line, got: %q", long)
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
	if got := m.groupKeyForSession(s, "health"); got != string(session.HealthOK) {
		t.Fatalf("health group key unexpected: %q", got)
	}
	if got := m.groupKeyForSession(s, "host"); got == "" || !strings.Contains(got, "~/work/project") {
		t.Fatalf("host group key unexpected: %q", got)
	}
	if got := m.groupKeyForSession(s, "none"); got != "" {
		t.Fatalf("none group key expected empty, got %q", got)
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
		width:        120,
		height:       36,
		home:         "/home/omega",
		sessionsRoot: filepath.Join(workspace, "sessions"),
		previewCache: make(map[string][]string),
		groupBy:      "month",
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
	if got := fitCell("abc", 8); runewidth.StringWidth(got) != 8 {
		t.Fatalf("fitCell width unexpected: %q", got)
	}
	if got := fitCellMiddle("/very/long/path/to/project", 12); runewidth.StringWidth(got) != 12 {
		t.Fatalf("fitCellMiddle width unexpected: %q", got)
	}
	if step := m.previewPageStep(); step <= 0 {
		t.Fatalf("previewPageStep should be positive, got %d", step)
	}
	if !isPreviewNoise("filesystem sandboxing note") {
		t.Fatal("expected preview noise detection")
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
