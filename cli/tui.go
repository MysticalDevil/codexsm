package cli

import (
	"bufio"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/spf13/cobra"

	"github.com/MysticalDevil/codex-sm/config"
	"github.com/MysticalDevil/codex-sm/session"
)

type treeItemKind int

const (
	treeItemMonth treeItemKind = iota
	treeItemSession
)

type treeItem struct {
	kind   treeItemKind
	label  string
	month  string
	index  int
	indent int
}

type tuiFocus int

const (
	focusTree tuiFocus = iota
	focusPreview
)

type tuiModel struct {
	sessions      []session.Session
	tree          []treeItem
	cursor        int
	offset        int
	previewOffset int
	width         int
	height        int
	home          string
	sessionsRoot  string
	status        string
	previewCache  map[string][]string
	lastPath      string
	focus         tuiFocus
}

const (
	tokyoBg        = "#1a1b26"
	tokyoFg        = "#c0caf5"
	tokyoBlue      = "#7aa2f7"
	tokyoCyan      = "#7dcfff"
	tokyoMagenta   = "#bb9af7"
	tokyoComment   = "#565f89"
	tokyoSelection = "#283457"

	tuiMinWidth  = 100
	tuiMinHeight = 24
)

func newTUICmd() *cobra.Command {
	var (
		sessionsRoot string
		limit        int
	)

	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive TUI session browser",
		Long: "Interactive session browser (optional).\n\n" +
			"Keys:\n" +
			"  j/k or Down/Up: move cursor\n" +
			"  g/G: first/last\n" +
			"  Ctrl+d / Ctrl+u: scroll preview\n" +
			"  d: dry-run delete preview for current session\n" +
			"  q: quit",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(sessionsRoot) == "" {
				v, err := config.DefaultSessionsRoot()
				if err != nil {
					return err
				}
				sessionsRoot = v
			} else {
				v, err := config.ResolvePath(sessionsRoot)
				if err != nil {
					return err
				}
				sessionsRoot = v
			}

			items, err := session.ScanSessions(sessionsRoot)
			if err != nil {
				return err
			}
			sortTUISessions(items)
			if limit > 0 && len(items) > limit {
				items = items[:limit]
			}

			home, _ := config.ResolvePath("~")
			m := tuiModel{
				sessions:     items,
				home:         home,
				sessionsRoot: sessionsRoot,
				status:       "Ready. Press q to quit.",
				previewCache: make(map[string][]string),
				focus:        focusTree,
			}
			m.rebuildTree()
			_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
			return err
		},
	}

	cmd.Flags().StringVar(&sessionsRoot, "sessions-root", "", "sessions root directory")
	cmd.Flags().IntVar(&limit, "limit", 100, "max sessions loaded into TUI (0 means unlimited)")
	return cmd
}

func (m tuiModel) Init() tea.Cmd {
	return nil
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampOffset()
		return m, nil
	case tea.KeyMsg:
		if m.handleKey(msg.String()) {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *tuiModel) handleKey(key string) bool {
	switch key {
	case "q", "ctrl+c":
		return true
	case "tab", "ctrl+i":
		if m.focus == focusTree {
			m.focus = focusPreview
		} else {
			m.focus = focusTree
		}
	case "shift+tab", "backtab":
		if m.focus == focusPreview {
			m.focus = focusTree
		} else {
			m.focus = focusPreview
		}
	case "right", "l":
		m.focus = focusPreview
	case "left", "h":
		m.focus = focusTree
	case "t", "1":
		m.focus = focusTree
	case "p", "2":
		m.focus = focusPreview
	case "j", "down":
		if m.focus == focusTree {
			if m.cursor < len(m.tree)-1 {
				m.cursor++
			}
			m.skipToSelectable(1)
			m.syncPreviewSelection()
			m.clampOffset()
		} else {
			m.previewOffset++
		}
	case "k", "up":
		if m.focus == focusTree {
			if m.cursor > 0 {
				m.cursor--
			}
			m.skipToSelectable(-1)
			m.syncPreviewSelection()
			m.clampOffset()
		} else {
			m.previewOffset--
			if m.previewOffset < 0 {
				m.previewOffset = 0
			}
		}
	case "g":
		if m.focus == focusTree {
			m.cursor = 0
			m.skipToSelectable(1)
			m.syncPreviewSelection()
			m.clampOffset()
		} else {
			m.previewOffset = 0
		}
	case "G":
		if m.focus == focusTree {
			if len(m.tree) > 0 {
				m.cursor = len(m.tree) - 1
			}
			m.skipToSelectable(-1)
			m.syncPreviewSelection()
			m.clampOffset()
		} else {
			m.previewOffset = 1 << 30 // clamped in View by preview length
		}
	case "ctrl+d":
		if m.focus == focusPreview {
			m.previewOffset += m.previewPageStep()
		}
	case "ctrl+u":
		if m.focus == focusPreview {
			m.previewOffset -= m.previewPageStep()
			if m.previewOffset < 0 {
				m.previewOffset = 0
			}
		}
	case "d":
		m.runDryRunPreview()
	}
	return false
}

func (m tuiModel) View() string {
	totalW := m.width
	totalH := m.height
	if totalW <= 0 {
		totalW = 120
	}
	if totalH <= 0 {
		totalH = 32
	}

	keysPanelStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(tokyoBlue)).
		Foreground(lipgloss.Color(tokyoCyan)).
		Padding(0, 1)
	keysOuterH := 3
	keysInnerW := max(20, totalW-keysPanelStyle.GetHorizontalFrameSize())
	keybarBody := "[KEYS] Tab/h/l t/p/1/2 switch pane | j/k scroll active pane | g/G top/bottom | Ctrl+d/u preview page | d dry-run | q quit"
	keybar := keysPanelStyle.
		Width(keysInnerW).
		Bold(true).
		Render(truncateDisplay(keybarBody, keysInnerW))

	if m.width > 0 && m.height > 0 && (m.width < tuiMinWidth || m.height < tuiMinHeight) {
		msg := fmt.Sprintf(
			"Terminal too small.\nRequired at least: %dx%d\nCurrent: %dx%d\nResize terminal and try again. Press q to quit.",
			tuiMinWidth, tuiMinHeight, m.width, m.height,
		)
		mainAreaH := max(6, totalH-keysOuterH)
		warn := lipgloss.NewStyle().
			Width(max(32, totalW-2)).
			Height(max(4, mainAreaH-2)).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(tokyoBlue)).
			Foreground(lipgloss.Color(tokyoFg)).
			Background(lipgloss.Color(tokyoBg)).
			Padding(1, 2).
			Render(msg)
		return strings.Join([]string{warn, keybar}, "\n")
	}

	mainAreaH := max(8, totalH-keysOuterH)

	if len(m.sessions) == 0 || len(m.tree) == 0 {
		empty := lipgloss.NewStyle().Foreground(lipgloss.Color(tokyoComment)).
			Render("No sessions found.")
		emptyPane := lipgloss.NewStyle().
			Width(max(32, totalW-2)).
			Height(max(4, mainAreaH-2)).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(tokyoBlue)).
			Foreground(lipgloss.Color(tokyoFg)).
			Background(lipgloss.Color(tokyoBg)).
			Padding(0, 1).
			Render(empty + "\n" + m.status)
		return strings.Join([]string{keybar, emptyPane}, "\n")
	}

	gapW := 1
	leftOuterW := int(float64(totalW) * 0.33)
	if leftOuterW < 28 {
		leftOuterW = 28
	}
	if leftOuterW > totalW-36-gapW {
		leftOuterW = max(28, totalW-36-gapW)
	}
	rightOuterW := totalW - leftOuterW - gapW
	if rightOuterW < 36 {
		rightOuterW = 36
		leftOuterW = max(28, totalW-rightOuterW-gapW)
	}
	if leftOuterW+gapW+rightOuterW > totalW {
		rightOuterW = max(36, totalW-leftOuterW-gapW)
	}

	infoOuterH := 4 // border + exactly 2 text rows, no blank
	if infoOuterH >= mainAreaH-4 {
		infoOuterH = max(3, mainAreaH/4)
	}
	previewOuterH := mainAreaH - infoOuterH
	if previewOuterH < 5 {
		previewOuterH = 5
		infoOuterH = max(3, mainAreaH-previewOuterH)
	}

	leftBase := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Foreground(lipgloss.Color(tokyoFg)).
		Background(lipgloss.Color(tokyoBg)).
		Padding(0, 1)
	rightBase := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Foreground(lipgloss.Color(tokyoFg)).
		Background(lipgloss.Color(tokyoBg)).
		Padding(0, 1)
	infoBase := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Foreground(lipgloss.Color(tokyoFg)).
		Background(lipgloss.Color(tokyoBg)).
		Padding(0, 1)

	leftW := max(12, leftOuterW-leftBase.GetHorizontalFrameSize())
	rightW := max(12, rightOuterW-rightBase.GetHorizontalFrameSize())

	leftTitleText := "SESSIONS (By Month)"
	rightTitleText := "PREVIEW"
	if m.focus == focusTree {
		leftTitleText += " *"
	} else {
		rightTitleText += " *"
	}
	leftTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tokyoCyan)).Render(leftTitleText)
	rightTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tokyoCyan)).Render(rightTitleText)

	leftLines := make([]string, 0, m.visibleRows()+1)
	previewLines := make([]string, 0, m.visibleRows()+1)
	infoLines := make([]string, 0, 4)
	leftLines = append(leftLines, leftTitle)
	previewLines = append(previewLines, rightTitle)

	start, end := m.visibleRange()
	for i := start; i < end; i++ {
		item := m.tree[i]
		line := item.label
		if item.indent > 0 {
			line = strings.Repeat("  ", item.indent) + line
		}
		line = truncateDisplay(line, leftW-4)
		if i == m.cursor {
			if m.focus == focusTree {
				line = lipgloss.NewStyle().Foreground(lipgloss.Color(tokyoBg)).Background(lipgloss.Color(tokyoCyan)).Bold(true).Render(line)
				line = lipgloss.NewStyle().Foreground(lipgloss.Color(tokyoCyan)).Render("▌") + " " + line
			} else {
				line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tokyoBlue)).Render("▏ " + line)
			}
		} else {
			if item.kind == treeItemMonth {
				line = lipgloss.NewStyle().Foreground(lipgloss.Color(tokyoMagenta)).Render(line)
			}
			line = "  " + line
		}
		leftLines = append(leftLines, line)
	}

	selected, ok := m.selectedSession()
	infoInnerH := max(1, infoOuterH-infoBase.GetVerticalFrameSize())
	if infoInnerH > 2 {
		infoInnerH = 2
	}
	previewInnerH := max(2, previewOuterH-rightBase.GetVerticalFrameSize())
	previewContentHeight := max(2, previewInnerH-3) // title + scroll + bar
	previewTextWidth := max(8, rightW-8)
	if ok {
		preview := m.previewFor(selected.Path, previewTextWidth, previewContentHeight)
		maybeMax := max(0, len(preview)-previewContentHeight)
		if m.previewOffset > maybeMax {
			m.previewOffset = maybeMax
		}
		start := m.previewOffset
		end := start + previewContentHeight
		if end > len(preview) {
			end = len(preview)
		}
		scrollInfo := fmt.Sprintf(" scroll %d-%d/%d ", start+1, end, len(preview))
		scrollStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(tokyoComment))
		barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(tokyoBlue))
		if m.focus == focusPreview {
			scrollStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tokyoCyan))
			barStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tokyoCyan))
		}
		previewLines = append(previewLines, scrollStyle.Render(truncateDisplay(scrollInfo, previewTextWidth)))
		previewLines = append(previewLines, barStyle.Render(" "+buildPreviewScrollBar(start, end, len(preview), max(10, previewTextWidth-2))))
		previewLines = append(previewLines, preview[start:end]...)
		h, v := m.detailRows(selected)
		infoLines = append(infoLines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tokyoMagenta)).Render(h))
		infoLines = append(infoLines, lipgloss.NewStyle().Foreground(lipgloss.Color(tokyoFg)).Render(v))
	} else {
		previewLines = append(previewLines, " Select a session node")
		infoLines = append(infoLines, lipgloss.NewStyle().Foreground(lipgloss.Color(tokyoComment)).Render("No session selected"))
	}

	leftBorder := tokyoBlue
	rightBorder := tokyoBlue
	if m.focus == focusTree {
		leftBorder = tokyoCyan
	} else {
		rightBorder = tokyoCyan
	}

	leftPane := leftBase.
		Width(leftW).
		Height(max(2, mainAreaH-leftBase.GetVerticalFrameSize())).
		BorderForeground(lipgloss.Color(leftBorder)).
		Render(strings.Join(leftLines, "\n"))

	previewPane := rightBase.
		Width(rightW).
		Height(previewInnerH).
		BorderForeground(lipgloss.Color(rightBorder)).
		Render(strings.Join(previewLines, "\n"))

	infoBorder := tokyoBlue
	if m.focus == focusPreview {
		infoBorder = tokyoCyan
	}
	infoPane := infoBase.
		Width(rightW).
		Height(infoInnerH).
		BorderForeground(lipgloss.Color(infoBorder)).
		Render(strings.Join(infoLines[:minInt(len(infoLines), 2)], "\n"))

	rightBlock := lipgloss.JoinVertical(lipgloss.Left, infoPane, previewPane)
	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, strings.Repeat(" ", gapW), rightBlock)

	return strings.Join([]string{
		mainArea,
		keybar,
	}, "\n")
}

func (m *tuiModel) runDryRunPreview() {
	selected, ok := m.selectedSession()
	if !ok {
		m.status = "No sessions to simulate."
		return
	}
	sum, err := session.DeleteSessions(
		[]session.Session{selected},
		session.Selector{ID: selected.SessionID},
		session.DeleteOptions{
			DryRun:       true,
			SessionsRoot: m.sessionsRoot,
			TrashRoot:    filepath.Join(filepath.Dir(m.sessionsRoot), "trash"),
		},
	)
	if err != nil {
		m.status = "dry-run failed: " + err.Error()
		return
	}
	m.status = fmt.Sprintf(
		"dry-run: action=%s matched=%d affected=%s",
		sum.Action,
		sum.MatchedCount,
		formatBytesIEC(sum.AffectedBytes),
	)
}

func (m *tuiModel) visibleRows() int {
	h := m.height
	if h <= 0 {
		return 12
	}
	rows := h - 8
	if rows < 5 {
		return 5
	}
	return rows
}

func (m *tuiModel) visibleRange() (int, int) {
	rows := m.visibleRows()
	start := m.offset
	if start < 0 {
		start = 0
	}
	end := start + rows
	if end > len(m.tree) {
		end = len(m.tree)
	}
	return start, end
}

func (m *tuiModel) clampOffset() {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	rows := m.visibleRows()
	if m.cursor >= m.offset+rows {
		m.offset = m.cursor - rows + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
	maxOffset := len(m.tree) - rows
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.offset > maxOffset {
		m.offset = maxOffset
	}
}

func sortTUISessions(items []session.Session) {
	slices.SortStableFunc(items, func(a, b session.Session) int {
		c := b.UpdatedAt.Compare(a.UpdatedAt)
		if c != 0 {
			return c
		}
		return strings.Compare(a.SessionID, b.SessionID)
	})
}

func (m *tuiModel) rebuildTree() {
	m.tree = make([]treeItem, 0, len(m.sessions)+16)
	currentMonth := ""
	for i, s := range m.sessions {
		month := s.UpdatedAt.Format("2006-01")
		if month == "0001-01" {
			month = "unknown"
		}
		if month != currentMonth {
			currentMonth = month
			m.tree = append(m.tree, treeItem{
				kind:   treeItemMonth,
				label:  "▾ " + month,
				month:  month,
				indent: 0,
			})
		}
		m.tree = append(m.tree, treeItem{
			kind:   treeItemSession,
			label:  "└─ " + shortID(s.SessionID),
			month:  month,
			index:  i,
			indent: 1,
		})
	}
	m.cursor = 0
	m.skipToSelectable(1)
	m.syncPreviewSelection()
}

func (m *tuiModel) skipToSelectable(step int) {
	if len(m.tree) == 0 {
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.tree) {
		m.cursor = len(m.tree) - 1
	}
	for m.cursor >= 0 && m.cursor < len(m.tree) {
		if m.tree[m.cursor].kind == treeItemSession {
			return
		}
		m.cursor += step
	}
	if step > 0 {
		m.cursor = len(m.tree) - 1
	} else {
		m.cursor = 0
	}
}

func (m *tuiModel) selectedSession() (session.Session, bool) {
	if len(m.tree) == 0 || m.cursor < 0 || m.cursor >= len(m.tree) {
		return session.Session{}, false
	}
	item := m.tree[m.cursor]
	if item.kind != treeItemSession || item.index < 0 || item.index >= len(m.sessions) {
		return session.Session{}, false
	}
	return m.sessions[item.index], true
}

func (m *tuiModel) detailRows(selected session.Session) (header string, values string) {
	host := compactHomePath(selected.HostDir, m.home)
	if strings.TrimSpace(host) == "" {
		host = "-"
	}
	contentWidth := max(40, m.width-4)
	cols := []struct {
		name string
		val  string
		w    int
	}{
		{name: "ID", val: shortID(selected.SessionID), w: 12},
		{name: "UPDATED", val: formatDisplayTime(selected.UpdatedAt), w: 19},
		{name: "SIZE", val: formatBytesIEC(selected.SizeBytes), w: 8},
		{name: "HEALTH", val: string(selected.Health), w: 12},
		{name: "HOST", val: host, w: max(14, minInt(24, contentWidth/5))},
	}

	var headerParts []string
	var valueParts []string
	for _, c := range cols {
		headerParts = append(headerParts, fitCell(c.name, c.w))
		valueParts = append(valueParts, fitCell(c.val, c.w))
	}
	return strings.Join(headerParts, "  "), strings.Join(valueParts, "  ")
}

func (m *tuiModel) previewFor(path string, width, lines int) []string {
	if width < 10 {
		width = 10
	}
	if lines < 5 {
		lines = 5
	}
	if cached, ok := m.previewCache[path]; ok {
		return cached
	}

	const maxPreviewLines = 600
	out := make([]string, 0, minInt(maxPreviewLines, lines*10))
	f, err := os.Open(path)
	if err != nil {
		out = append(out, " failed to open preview: "+err.Error())
		m.previewCache[path] = out
		return out
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() && len(out) < maxPreviewLines {
		line := strings.TrimSpace(sc.Text())
		if line == "" || !jsontext.Value([]byte(line)).IsValid() {
			continue
		}
		role, text := previewLine(line)
		if strings.TrimSpace(text) == "" {
			continue
		}
		if isPreviewNoise(text) {
			continue
		}
		prefix := "?"
		switch role {
		case "user":
			prefix = "U"
		case "assistant":
			prefix = "A"
		}
		first := true
		contentWidth := max(4, width-3) // " <role> " prefix takes 3 cells
		wrapped := wrapText(text, contentWidth)
		trimmed := wrapped
		truncated := false
		if len(trimmed) > 2 {
			trimmed = trimmed[:2]
			truncated = true
		}
		for i, chunk := range trimmed {
			if truncated && i == len(trimmed)-1 {
				chunk = withEllipsis(chunk, contentWidth)
			}
			p := " "
			if first {
				p = prefix
				first = false
			}
			row := fmt.Sprintf(" %s %s", p, chunk)
			out = append(out, truncateDisplay(row, width))
		}
	}
	if err := sc.Err(); err != nil {
		out = append(out, " preview read error: "+err.Error())
	}
	if len(out) == 0 {
		out = append(out, " no dialogue preview available")
	}
	m.previewCache[path] = out
	return out
}

func previewLine(line string) (role string, text string) {
	var item struct {
		Type    string `json:"type"`
		Payload struct {
			Type    string `json:"type"`
			Role    string `json:"role"`
			Text    string `json:"text"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"payload"`
	}
	if err := json.Unmarshal([]byte(line), &item); err != nil {
		return "", ""
	}
	if item.Type != "response_item" || item.Payload.Type != "message" {
		return "", ""
	}
	for _, c := range item.Payload.Content {
		if v := compactPreviewText(c.Text); v != "" {
			return item.Payload.Role, v
		}
	}
	return item.Payload.Role, compactPreviewText(item.Payload.Text)
}

func compactPreviewText(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	return strings.Join(strings.Fields(v), " ")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func fitCell(v string, width int) string {
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(v) > width {
		v = truncateDisplay(v, width)
	}
	w := runewidth.StringWidth(v)
	if w >= width {
		return v
	}
	return v + strings.Repeat(" ", width-w)
}

func wrapText(v string, width int) []string {
	if width <= 0 {
		return []string{v}
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return []string{""}
	}
	words := strings.Fields(v)
	if len(words) <= 1 {
		return wrapRunesByWidth(v, width)
	}
	out := make([]string, 0, len(words))
	line := words[0]
	for _, w := range words[1:] {
		candidate := line + " " + w
		if runewidth.StringWidth(candidate) <= width {
			line = candidate
			continue
		}
		out = append(out, line)
		if runewidth.StringWidth(w) > width {
			split := wrapRunesByWidth(w, width)
			if len(split) > 0 {
				out = append(out, split[:len(split)-1]...)
				line = split[len(split)-1]
				continue
			}
		}
		line = w
	}
	out = append(out, line)
	return out
}

func wrapRunesByWidth(v string, width int) []string {
	if width <= 0 {
		return []string{v}
	}
	var out []string
	var b strings.Builder
	current := 0
	for _, r := range v {
		rw := runewidth.RuneWidth(r)
		if rw <= 0 {
			rw = 1
		}
		if current+rw > width && b.Len() > 0 {
			out = append(out, b.String())
			b.Reset()
			current = 0
		}
		b.WriteRune(r)
		current += rw
	}
	if b.Len() > 0 {
		out = append(out, b.String())
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

func truncateDisplay(v string, width int) string {
	if width <= 0 {
		return v
	}
	if runewidth.StringWidth(v) <= width {
		return v
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}
	target := width - 3
	var b strings.Builder
	current := 0
	for _, r := range v {
		rw := runewidth.RuneWidth(r)
		if rw <= 0 {
			rw = 1
		}
		if current+rw > target {
			break
		}
		b.WriteRune(r)
		current += rw
	}
	b.WriteString("...")
	return b.String()
}

func withEllipsis(v string, width int) string {
	if width <= 0 {
		return ""
	}
	if width <= 3 {
		return truncateDisplay(v, width)
	}
	if runewidth.StringWidth(v)+3 <= width {
		return v + "..."
	}
	return truncateDisplay(v, width-3) + "..."
}

func (m *tuiModel) syncPreviewSelection() {
	selected, ok := m.selectedSession()
	if !ok {
		m.lastPath = ""
		m.previewOffset = 0
		return
	}
	if selected.Path != m.lastPath {
		m.lastPath = selected.Path
		m.previewOffset = 0
	}
}

func (m *tuiModel) previewPageStep() int {
	step := m.visibleRows() / 2
	if step < 1 {
		return 1
	}
	return step
}

func buildPreviewScrollBar(start, end, total, width int) string {
	if width < 8 {
		width = 8
	}
	if total <= 0 {
		return "[" + strings.Repeat("─", width) + "]"
	}
	if end < start {
		end = start
	}
	if end > total {
		end = total
	}
	beginRatio := float64(start) / float64(total)
	endRatio := float64(end) / float64(total)
	l := int(beginRatio * float64(width))
	r := int(endRatio * float64(width))
	if r <= l {
		r = l + 1
	}
	if r > width {
		r = width
	}
	var b strings.Builder
	b.WriteByte('[')
	for i := 0; i < width; i++ {
		if i >= l && i < r {
			b.WriteString("█")
		} else {
			b.WriteString("─")
		}
	}
	b.WriteByte(']')
	return b.String()
}

func isPreviewNoise(v string) bool {
	l := strings.ToLower(strings.TrimSpace(v))
	if l == "" {
		return true
	}
	noise := []string{
		"agents.md",
		"instructions for",
		"filesystem sandboxing",
		"approved command prefix",
		"global agent rules",
		"tooling priorities",
		"validation before finish",
		"use-modern-go",
	}
	for _, k := range noise {
		if strings.Contains(l, k) {
			return true
		}
	}
	return false
}
