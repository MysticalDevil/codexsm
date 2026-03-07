package tui

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

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
	mode := strings.ToLower(strings.TrimSpace(m.groupBy))
	if mode == "" {
		mode = "month"
	}
	if mode == "none" {
		for i, s := range m.sessions {
			m.tree = append(m.tree, treeItem{
				kind:        treeItemSession,
				label:       shortID(s.SessionID),
				month:       m.groupKeyForSession(s, mode),
				index:       i,
				indent:      1,
				hostMissing: m.sessionHostMissing(s),
			})
		}
	} else {
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
			m.tree = append(m.tree, treeItem{
				kind:   treeItemMonth,
				label:  "▾ " + group,
				month:  group,
				indent: 0,
			})
			for _, i := range grouped[group] {
				m.tree = append(m.tree, treeItem{
					kind:        treeItemSession,
					label:       shortID(m.sessions[i].SessionID),
					month:       group,
					index:       i,
					indent:      1,
					hostMissing: m.sessionHostMissing(m.sessions[i]),
				})
			}
		}
	}
	m.cursor = 0
	m.skipToSelectable(1)
	m.syncPreviewSelection()
}

func (m *tuiModel) groupKeyForSession(s session.Session, mode string) string {
	switch mode {
	case "none":
		return ""
	case "day":
		if s.UpdatedAt.IsZero() {
			return "unknown-day"
		}
		return s.UpdatedAt.Local().Format("2006-01-02")
	case "health":
		if strings.TrimSpace(string(s.Health)) == "" {
			return "unknown-health"
		}
		return strings.ToUpper(string(s.Health))
	case "host":
		host := compactHomePath(s.HostDir, m.home)
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
		mode = "month"
	}
	switch mode {
	case "month", "day", "health", "host", "none":
		return mode, nil
	default:
		return "", fmt.Errorf("invalid --group-by %q (allowed: month, day, health, host, none)", v)
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

func (m *tuiModel) selectedSessionHostMissing() bool {
	if len(m.tree) == 0 || m.cursor < 0 || m.cursor >= len(m.tree) {
		return false
	}
	item := m.tree[m.cursor]
	if item.kind != treeItemSession {
		return false
	}
	return item.hostMissing
}

func (m *tuiModel) sessionHostMissing(s session.Session) bool {
	host := strings.TrimSpace(s.HostDir)
	if host == "" {
		return false
	}
	_, err := os.Stat(host)
	return errors.Is(err, os.ErrNotExist)
}

func (m *tuiModel) detailRows(selected session.Session) (header string, values string) {
	host := compactHomePath(selected.HostDir, m.home)
	if strings.TrimSpace(host) == "" {
		host = "-"
	}
	if m.selectedSessionHostMissing() {
		host += " (missing)"
	}
	contentWidth := max(40, m.width-4)
	hostW := max(18, minInt(36, contentWidth/3))
	cols := []struct {
		name string
		val  string
		w    int
	}{
		{name: "ID", val: shortID(selected.SessionID), w: 12},
		{name: "UPDATED", val: formatDisplayTime(selected.UpdatedAt), w: 19},
		{name: "SIZE", val: formatBytesIEC(selected.SizeBytes), w: 8},
		{name: "HEALTH", val: strings.ToUpper(string(selected.Health)), w: 12},
		{name: "HOST", val: previewHostPath(host, hostW), w: hostW},
	}

	var headerParts []string
	var valueParts []string
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
	return strings.Join(headerParts, "  "), strings.Join(valueParts, "  ")
}

func (m *tuiModel) healthColorHex(h session.Health) string {
	switch h {
	case session.HealthOK:
		return m.colorHex("tag_success")
	case session.HealthCorrupted:
		return m.colorHex("tag_danger")
	case session.HealthMissingMeta:
		return m.colorHex("tag_default")
	default:
		return m.colorHex("info_value")
	}
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
