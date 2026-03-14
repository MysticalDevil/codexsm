package tui

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/MysticalDevil/codexsm/internal/core"
	"github.com/MysticalDevil/codexsm/session"
	"github.com/charmbracelet/lipgloss"
)

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

func (m *tuiModel) rebuildTree() {
	hasPrev := false
	prevCursor := m.cursor
	prevKind := treeItemSession
	prevGroup := ""
	prevSessionID := ""

	if len(m.tree) > 0 && m.cursor >= 0 && m.cursor < len(m.tree) {
		hasPrev = true
		prev := m.tree[m.cursor]
		prevKind = prev.Kind
		prevGroup = prev.Month

		if prev.Kind == treeItemSession && prev.Index >= 0 && prev.Index < len(m.sessions) {
			prevSessionID = m.sessions[prev.Index].SessionID
		}
	}

	m.tree = make([]treeItem, 0, len(m.sessions)+16)

	mode := strings.ToLower(strings.TrimSpace(m.groupBy))
	if mode == "" {
		mode = "host"
	}

	groupOrder := make([]string, 0, len(m.sessions))

	grouped := make(map[string][]int, len(m.sessions))
	for i, s := range m.sessions {
		group := m.groupKeyForSession(s, mode)
		if _, exists := grouped[group]; !exists {
			groupOrder = append(groupOrder, group)
		}

		grouped[group] = append(grouped[group], i)
	}

	for _, group := range groupOrder {
		prefix := "▾ "
		if m.isGroupCollapsed(group) {
			prefix = "▸ "
		}

		m.tree = append(m.tree, treeItem{
			Kind:   treeItemMonth,
			Label:  prefix + group,
			Month:  group,
			Indent: 0,
		})

		if m.isGroupCollapsed(group) {
			continue
		}

		for _, i := range grouped[group] {
			m.tree = append(m.tree, treeItem{
				Kind:        treeItemSession,
				Label:       core.ShortID(m.sessions[i].SessionID),
				Month:       group,
				Index:       i,
				Indent:      1,
				HostMissing: m.sessionHostMissing(m.sessions[i]),
			})
		}
	}

	if len(m.tree) == 0 {
		m.cursor = 0
		m.syncPreviewSelection()

		return
	}

	if hasPrev && prevKind == treeItemMonth && strings.TrimSpace(prevGroup) != "" {
		if idx, ok := m.findGroupIndex(prevGroup); ok {
			m.cursor = idx
			m.syncPreviewSelection()

			return
		}
	}

	if hasPrev && strings.TrimSpace(prevSessionID) != "" {
		if idx, ok := m.findSessionIndexByID(prevSessionID); ok {
			m.cursor = idx
			m.syncPreviewSelection()

			return
		}
	}

	if hasPrev && prevCursor >= 0 && prevCursor < len(m.tree) {
		m.cursor = prevCursor
		m.syncPreviewSelection()

		return
	}

	m.cursor = 0
	m.skipToSelectable(1)
	m.syncPreviewSelection()
}

func (m *tuiModel) isGroupCollapsed(group string) bool {
	if len(m.collapsedGroups) == 0 {
		return false
	}

	return m.collapsedGroups[group]
}

func (m *tuiModel) toggleSelectedGroupCollapsed() bool {
	if len(m.tree) == 0 || m.cursor < 0 || m.cursor >= len(m.tree) {
		return false
	}

	item := m.tree[m.cursor]

	group := item.Month
	if item.Kind != treeItemMonth && item.Kind != treeItemSession {
		return false
	}

	if strings.TrimSpace(group) == "" {
		return false
	}

	if m.collapsedGroups == nil {
		m.collapsedGroups = make(map[string]bool)
	}

	if m.collapsedGroups[group] {
		delete(m.collapsedGroups, group)
	} else {
		m.collapsedGroups[group] = true
	}

	return true
}

func (m *tuiModel) currentCursorGroup() (string, bool) {
	if len(m.tree) == 0 || m.cursor < 0 || m.cursor >= len(m.tree) {
		return "", false
	}

	item := m.tree[m.cursor]
	if item.Kind != treeItemMonth && item.Kind != treeItemSession {
		return "", false
	}

	group := strings.TrimSpace(item.Month)
	if group == "" {
		return "", false
	}

	return group, true
}

func (m *tuiModel) findGroupIndex(group string) (int, bool) {
	for i, item := range m.tree {
		if item.Kind == treeItemMonth && item.Month == group {
			return i, true
		}
	}

	return 0, false
}

func (m *tuiModel) findSessionIndexByID(sessionID string) (int, bool) {
	for i, item := range m.tree {
		if item.Kind != treeItemSession || item.Index < 0 || item.Index >= len(m.sessions) {
			continue
		}

		if m.sessions[item.Index].SessionID == sessionID {
			return i, true
		}
	}

	return 0, false
}

func (m *tuiModel) expandAllGroups() bool {
	if len(m.collapsedGroups) == 0 {
		return false
	}

	clear(m.collapsedGroups)

	return true
}

func (m *tuiModel) groupKeyForSession(s session.Session, mode string) string {
	switch mode {
	case "day":
		if s.UpdatedAt.IsZero() {
			return "unknown-day"
		}

		return s.UpdatedAt.Local().Format("2006-01-02")
	case "host":
		host := core.CompactHomePath(s.HostDir, m.home)
		if strings.TrimSpace(host) == "" {
			return "unknown-host"
		}

		return host
	case "month":
		fallthrough
	default:
		if s.UpdatedAt.IsZero() {
			return "unknown-month"
		}

		return s.UpdatedAt.Format("2006-01")
	}
}

func normalizeTUIGroupBy(v string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(v))
	if mode == "" {
		mode = "host"
	}

	switch mode {
	case "month", "day", "host":
		return mode, nil
	default:
		return "", fmt.Errorf("invalid --group-by %q (allowed: month, day, host)", v)
	}
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
		if m.tree[m.cursor].Kind == treeItemSession {
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
	if item.Kind != treeItemSession || item.Index < 0 || item.Index >= len(m.sessions) {
		return session.Session{}, false
	}

	return m.sessions[item.Index], true
}

func (m *tuiModel) selectedSessionHostMissing() bool {
	if len(m.tree) == 0 || m.cursor < 0 || m.cursor >= len(m.tree) {
		return false
	}

	item := m.tree[m.cursor]
	if item.Kind != treeItemSession {
		return false
	}

	return item.HostMissing
}

func (m *tuiModel) sessionHostMissing(s session.Session) bool {
	host := strings.TrimSpace(s.HostDir)
	if host == "" {
		return false
	}

	_, err := os.Stat(host)

	return errors.Is(err, os.ErrNotExist)
}

func (m *tuiModel) detailRows(selected session.Session, rightW int) (header string, values string) {
	host := core.CompactHomePath(selected.HostDir, m.home)
	if strings.TrimSpace(host) == "" {
		host = "-"
	}

	if m.selectedSessionHostMissing() {
		host += " (missing)"
	}

	contentWidth := max(24, rightW)
	hostW := max(12, min(28, contentWidth/3))

	type col struct {
		name string
		val  string
		w    int
	}

	cols := []col{
		{name: "ID", val: core.ShortID(selected.SessionID), w: 12},
		{name: "UPDATED", val: core.FormatDisplayTime(selected.UpdatedAt), w: 19},
		{name: "SIZE", val: core.FormatBytesIEC(selected.SizeBytes), w: 8},
		{name: "HEALTH", val: strings.ToUpper(string(selected.Health)), w: 12},
		{name: "HOST", val: previewHostPath(host, hostW), w: hostW},
	}
	if contentWidth < 88 {
		hostW = max(16, contentWidth-12-12-4)
		cols = []col{
			{name: "ID", val: core.ShortID(selected.SessionID), w: 12},
			{name: "HEALTH", val: strings.ToUpper(string(selected.Health)), w: 12},
			{name: "HOST", val: previewHostPath(host, hostW), w: hostW},
		}
	}

	if contentWidth < 64 {
		hostW = max(14, contentWidth-12-3)
		cols = []col{
			{name: "ID", val: core.ShortID(selected.SessionID), w: 12},
			{name: "HOST", val: previewHostPath(host, hostW), w: hostW},
		}
	}

	var (
		headerParts []string
		valueParts  []string
	)

	for _, c := range cols {
		headerParts = append(headerParts, fitCell(c.name, c.w))
		if c.name == "HOST" {
			valueParts = append(valueParts, fitCellMiddle(c.val, c.w))
			continue
		}

		cell := fitCell(c.val, c.w)
		if c.name == "HEALTH" {
			cell = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(m.healthColorHex(selected.Health))).
				Render(cell)
		}

		valueParts = append(valueParts, cell)
	}

	risk := session.EvaluateRisk(selected, nil)
	if risk.Level == session.RiskNone {
		return strings.Join(headerParts, "  "), strings.Join(valueParts, "  ")
	}

	riskText := strings.ToUpper(string(risk.Level)) + ": " + strings.ReplaceAll(string(risk.Reason), "-", " ")
	if strings.TrimSpace(risk.Detail) != "" {
		riskText += " (" + strings.TrimSpace(risk.Detail) + ")"
	}

	return strings.Join(headerParts, "  "), strings.Join(valueParts, "  ") + "\n" +
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.colorHex("tag_error"))).Render("RISK "+riskText)
}

func (m *tuiModel) healthColorHex(h session.Health) string {
	switch h {
	case session.HealthOK:
		return m.colorHex("status_ok")
	case session.HealthCorrupted:
		return m.colorHex("status_risk")
	case session.HealthMissingMeta:
		return m.colorHex("status_warn")
	default:
		return m.colorHex("status_info")
	}
}

func (m *tuiModel) healthSymbolAndColor(h session.Health) (string, string) {
	switch h {
	case session.HealthOK:
		return "•", m.colorHex("status_ok")
	case session.HealthMissingMeta:
		return "!", m.colorHex("status_warn")
	case session.HealthCorrupted:
		return "✖", m.colorHex("status_risk")
	default:
		return "•", m.colorHex("status_info")
	}
}

func (m *tuiModel) treeHealthVisual(h session.Health, hostMissing bool) (string, string, bool) {
	risk := session.EvaluateRisk(session.Session{Health: h}, nil)
	if risk.Level == session.RiskHigh {
		return "⚠", m.colorHex("status_risk"), true
	}

	if risk.Level == session.RiskMedium {
		return "!", m.colorHex("status_warn"), true
	}

	if h == session.HealthCorrupted {
		return "✖", m.colorHex("status_risk"), true
	}

	if hostMissing || h == session.HealthMissingMeta {
		return "!", m.colorHex("status_warn"), true
	}

	sym, color := m.healthSymbolAndColor(h)

	return sym, color, false
}

func (m *tuiModel) riskCounts() (high, medium int) {
	for _, item := range m.tree {
		if item.Kind != treeItemSession || item.Index < 0 || item.Index >= len(m.sessions) {
			continue
		}

		s := m.sessions[item.Index]
		switch session.EvaluateRisk(s, nil).Level {
		case session.RiskHigh:
			high++
		case session.RiskMedium:
			medium++
		case session.RiskNone:
			if item.HostMissing {
				medium++
			}
		}
	}

	return high, medium
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
