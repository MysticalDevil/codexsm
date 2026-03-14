package tui

import (
	"fmt"
	"strings"

	"github.com/MysticalDevil/codexsm/tui/preview"
	"github.com/charmbracelet/lipgloss"

	"github.com/MysticalDevil/codexsm/internal/core"
	"github.com/MysticalDevil/codexsm/session"
)

func (m tuiModel) View() string {
	metrics := Compute(m.width, m.height)
	borderColor := m.colorHex("border")
	borderFocusColor := m.colorHex("border_focus")
	fgColor := m.colorHex("fg")
	statusColor := m.colorHex("status")

	keysPanelStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Foreground(lipgloss.Color(fgColor)).
		Padding(0, 1)
	renderKeysBar := func(outerW int) string {
		innerW := max(1, outerW-keysPanelStyle.GetHorizontalFrameSize())
		keysLine := fitStyledWidth(m.renderBottomLine(innerW), innerW)
		bar := keysPanelStyle.Render(keysLine)

		return lipgloss.NewStyle().Width(outerW).MaxWidth(outerW).Render(bar)
	}

	if IsTooSmall(m.width, m.height) {
		msg := fmt.Sprintf(
			"Terminal too small.\nRequired at least: %dx%d\nCurrent: %dx%d\nResize terminal and try again.\nPress q to quit.",
			MinWidth+1,
			MinHeight,
			m.width,
			m.height,
		)
		warn := lipgloss.NewStyle().
			Width(max(32, metrics.TotalW-2)).
			Height(max(4, metrics.TotalH-2)).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(borderColor)).
			Foreground(lipgloss.Color(fgColor)).
			Padding(1, 2).
			Render(msg)
		warnW := lipgloss.Width(warn)
		warn = lipgloss.NewStyle().Width(warnW).MaxWidth(warnW).Render(warn)

		return lipgloss.NewStyle().
			Width(metrics.TotalW).
			MaxWidth(metrics.TotalW).
			AlignHorizontal(lipgloss.Center).
			Render(warn)
	}

	if len(m.sessions) == 0 || len(m.tree) == 0 {
		empty := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).
			Render("No sessions found.")
		emptyPane := lipgloss.NewStyle().
			Width(max(32, metrics.TotalW-2)).
			Height(max(4, metrics.MainAreaH-2)).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(borderColor)).
			Foreground(lipgloss.Color(fgColor)).
			Padding(0, 1).
			Render(empty + "\n" + m.status)
		emptyW := lipgloss.Width(emptyPane)
		emptyPane = lipgloss.NewStyle().Width(emptyW).MaxWidth(emptyW).Render(emptyPane)

		return lipgloss.NewStyle().
			Width(metrics.TotalW).
			MaxWidth(metrics.TotalW).
			AlignHorizontal(lipgloss.Center).
			Render(strings.Join([]string{emptyPane, renderKeysBar(emptyW)}, "\n"))
	}

	leftBase := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Foreground(lipgloss.Color(fgColor)).
		Padding(0, 1)
	rightBase := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Foreground(lipgloss.Color(fgColor)).
		Padding(0, 1)
	infoBase := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Foreground(lipgloss.Color(fgColor)).
		Padding(0, 1)

	leftW := max(12, metrics.LeftOuterW-leftBase.GetHorizontalFrameSize())
	rightW := max(12, metrics.RightOuterW-rightBase.GetHorizontalFrameSize())

	_, previewLines, infoLines := m.buildPanelLinesForMode(rightW, statusColor, metrics.Compact)

	selected, ok := m.selectedSession()
	if ok {
		m.appendSelectedSessionPreview(&previewLines, &infoLines, selected, rightW)
	} else {
		previewLines = append(previewLines, " Select a session node")
		infoLines = append(
			infoLines,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.colorHex("status_warn"))).
				Render("No session selected"),
		)
	}

	leftBorder := borderColor
	rightBorder := borderColor

	if m.focus == focusTree {
		leftBorder = borderFocusColor
	} else {
		rightBorder = borderFocusColor
	}

	treeOuterH := metrics.MainAreaH
	if metrics.Compact {
		treeOuterH = metrics.TreeOuterH
	}

	leftPane := leftBase.
		Width(leftW).
		Height(max(2, treeOuterH-leftBase.GetVerticalFrameSize())).
		BorderForeground(lipgloss.Color(leftBorder)).
		Render(m.renderTreePaneContentForMode(leftW, max(2, treeOuterH-leftBase.GetVerticalFrameSize()), statusColor, metrics.Compact))

	previewInnerH := max(2, metrics.PreviewOuterH-rightBase.GetVerticalFrameSize())
	previewPane := rightBase.
		Width(rightW).
		Height(previewInnerH).
		BorderForeground(lipgloss.Color(rightBorder)).
		Render(strings.Join(previewLines, "\n"))

	infoBorder := borderColor
	if m.focus == focusPreview {
		infoBorder = borderFocusColor
	}

	infoInnerH := max(1, metrics.InfoOuterH-infoBase.GetVerticalFrameSize())
	infoInnerH = min(infoInnerH, 3)
	infoPane := infoBase.
		Width(rightW).
		Height(infoInnerH).
		BorderForeground(lipgloss.Color(infoBorder)).
		Render(strings.Join(infoLines[:min(len(infoLines), 3)], "\n"))

	rightBlock := lipgloss.JoinVertical(lipgloss.Left, infoPane, previewPane)
	mainArea := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPane,
		strings.Repeat(" ", metrics.GapW),
		rightBlock,
	)

	mainOuterW := lipgloss.Width(mainArea)
	mainArea = lipgloss.NewStyle().Width(mainOuterW).MaxWidth(mainOuterW).Render(mainArea)
	keybar := renderKeysBar(mainOuterW)
	mainContainer := lipgloss.NewStyle().
		Width(metrics.TotalW).
		MaxWidth(metrics.TotalW).
		AlignHorizontal(lipgloss.Center).
		Foreground(lipgloss.Color(fgColor)).
		Render(lipgloss.JoinVertical(lipgloss.Left, mainArea, keybar))

	return mainContainer
}

func (m tuiModel) buildPanelLinesForMode(rightW int, statusColor string, compact bool) ([]string, []string, []string) {
	leftTitleText := "SESSIONS"
	if compact {
		leftTitleText = "SES [C]"
	} else if gb := strings.ToLower(strings.TrimSpace(m.groupBy)); gb != "" && gb != "none" {
		leftTitleText = fmt.Sprintf("SESSIONS (By %s)", strings.ToUpper(gb[:1])+gb[1:])
	}

	rightTitleText := "PREVIEW"

	if m.focus == focusTree {
		leftTitleText += " *"
	} else {
		rightTitleText += " *"
	}

	leftTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.colorHex("title_tree"))).
		Render(leftTitleText)
	rightTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.colorHex("title_preview"))).
		Render(rightTitleText)

	leftLines := make([]string, 0, m.visibleRows()+1)
	previewLines := make([]string, 0, m.visibleRows()+1)
	infoLines := make([]string, 0, 4)

	leftLines = append(leftLines, leftTitle)
	previewLines = append(previewLines, rightTitle)
	previewLines = append(
		previewLines,
		lipgloss.NewStyle().
			Foreground(lipgloss.Color(statusColor)).
			Render(" "+truncateDisplay(m.status, max(8, rightW-2))),
	)
	highRisk, mediumRisk := m.riskCounts()

	riskLine := fmt.Sprintf(
		" risk=%d (high=%d medium=%d) ",
		highRisk+mediumRisk,
		highRisk,
		mediumRisk,
	)
	if compact {
		riskLine = fmt.Sprintf(" r=%d h=%d m=%d ", highRisk+mediumRisk, highRisk, mediumRisk)
	}

	riskColor := m.colorHex("status_info")
	if highRisk > 0 {
		riskColor = m.colorHex("status_risk")
	} else if mediumRisk > 0 {
		riskColor = m.colorHex("status_warn")
	}

	previewLines = append(
		previewLines,
		lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(riskColor)).
			Render(" "+truncateDisplay(riskLine, max(8, rightW-2))),
	)

	return leftLines, previewLines, infoLines
}

func (m tuiModel) renderTreeLines(leftW int, statusColor string) []string {
	return m.renderTreeLinesForMode(leftW, statusColor, m.compactMode())
}

func (m tuiModel) renderTreeLinesForMode(leftW int, statusColor string, compact bool) []string {
	leftTitleText := "SESSIONS"
	if compact {
		leftTitleText = "SES [C]"
	} else if gb := strings.ToLower(strings.TrimSpace(m.groupBy)); gb != "" && gb != "none" {
		leftTitleText = fmt.Sprintf("SESSIONS (By %s)", strings.ToUpper(gb[:1])+gb[1:])
	}

	if m.focus == focusTree {
		leftTitleText += " *"
	}

	leftTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.colorHex("title_tree"))).
		Render(leftTitleText)
	leftLines := make([]string, 0, m.visibleRows()+1)
	leftLines = append(leftLines, leftTitle)

	start := m.offset
	if start < 0 {
		start = 0
	}

	end := start + m.visibleRows()
	if end > len(m.tree) {
		end = len(m.tree)
	}

	for i := start; i < end; i++ {
		item := m.tree[i]
		if item.Kind == treeItemMonth {
			groupLabel := item.Label
			if compact {
				groupLabel = compactTreeGroupLabel(item.Label)
			}

			line := truncateDisplay(groupLabel, leftW-4)

			if i == m.cursor {
				if m.focus == focusTree {
					line = lipgloss.NewStyle().
						Foreground(lipgloss.Color(m.colorHex("selected_fg"))).
						Background(lipgloss.Color(m.colorHex("selected_bg"))).
						Bold(true).
						Render(line)

					if compact {
						leftLines = append(leftLines, "  "+line)
					} else {
						leftLines = append(
							leftLines,
							lipgloss.NewStyle().
								Foreground(lipgloss.Color(m.colorHex("cursor_active"))).
								Render("▌")+" "+line,
						)
					}
				} else {
					line = lipgloss.NewStyle().
						Bold(true).
						Foreground(lipgloss.Color(m.colorHex("cursor_inactive"))).
						Render(line)

					if compact {
						leftLines = append(leftLines, "  "+line)
					} else {
						leftLines = append(
							leftLines,
							lipgloss.NewStyle().
								Foreground(lipgloss.Color(m.colorHex("cursor_inactive"))).
								Render("▏")+" "+line,
						)
					}
				}
			} else {
				groupColor := m.colorHex("accent_group")
				if compact && m.collapsedGroups[item.Month] {
					groupColor = m.colorHex("status_info")
				}

				line = lipgloss.NewStyle().Foreground(lipgloss.Color(groupColor)).Render(line)
				leftLines = append(leftLines, "  "+line)
			}

			continue
		}

		if compact {
			idWidth := max(4, leftW-4)
			idText := truncateDisplay(item.Label, idWidth)
			idColor := m.colorHex("status_info")

			if item.Index >= 0 && item.Index < len(m.sessions) {
				_, color, nonHealthy := m.treeHealthVisual(
					m.sessions[item.Index].Health,
					item.HostMissing,
				)

				idColor = color
				if nonHealthy {
					idText = lipgloss.NewStyle().
						Bold(true).
						Foreground(lipgloss.Color(color)).
						Render(idText)
				}
			}

			if i == m.cursor {
				if m.focus == focusTree {
					idText = lipgloss.NewStyle().
						Foreground(lipgloss.Color(m.colorHex("selected_fg"))).
						Background(lipgloss.Color(m.colorHex("selected_bg"))).
						Bold(true).
						Render(idText)
				} else {
					idText = lipgloss.NewStyle().
						Bold(true).
						Foreground(lipgloss.Color(m.colorHex("cursor_inactive"))).
						Render(idText)
				}

				leftLines = append(leftLines, "  "+idText)
			} else {
				idText = lipgloss.NewStyle().
					Foreground(lipgloss.Color(idColor)).
					Render(idText)
				leftLines = append(leftLines, "  "+idText)
			}

			continue
		}

		connector := "├─"
		if i+1 >= len(m.tree) || m.tree[i+1].Kind == treeItemMonth {
			connector = "└─"
		}

		connectorPart := lipgloss.NewStyle().
			Foreground(lipgloss.Color(statusColor)).
			Render("  " + connector + " ")

		idWidth := max(4, leftW-10)
		idText := truncateDisplay(item.Label, idWidth)
		healthSymbol := "●"

		healthColor := m.colorHex("status")
		if item.Index >= 0 && item.Index < len(m.sessions) {
			symbol, color, nonHealthy := m.treeHealthVisual(
				m.sessions[item.Index].Health,
				item.HostMissing,
			)
			healthSymbol = symbol

			healthColor = color
			if nonHealthy {
				idText = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color(color)).
					Render(idText)
			}
		}

		healthMark := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(healthColor)).
			Render(healthSymbol)

		if i == m.cursor {
			if m.focus == focusTree {
				idText = lipgloss.NewStyle().
					Foreground(lipgloss.Color(m.colorHex("selected_fg"))).
					Background(lipgloss.Color(m.colorHex("selected_bg"))).
					Bold(true).Render(idText)
				leftLines = append(
					leftLines,
					lipgloss.NewStyle().
						Foreground(lipgloss.Color(m.colorHex("cursor_active"))).
						Render("▌")+
						" "+connectorPart+healthMark+" "+idText,
				)
			} else {
				idText = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color(m.colorHex("cursor_inactive"))).
					Render(idText)
				leftLines = append(
					leftLines,
					lipgloss.NewStyle().
						Foreground(lipgloss.Color(m.colorHex("cursor_inactive"))).
						Render("▏")+
						" "+connectorPart+healthMark+" "+idText,
				)
			}
		} else {
			leftLines = append(leftLines, "  "+connectorPart+healthMark+" "+idText)
		}
	}

	return leftLines
}

func (m tuiModel) renderTreePaneContent(leftW, leftInnerH int, statusColor string) string {
	return m.renderTreePaneContentForMode(leftW, leftInnerH, statusColor, m.compactMode())
}

func (m tuiModel) renderTreePaneContentForMode(leftW, leftInnerH int, statusColor string, compact bool) string {
	lines := m.renderTreeLinesForMode(leftW, statusColor, compact)

	footer := m.treeFooterLine(leftW, statusColor, compact)
	if leftInnerH <= 1 {
		return footer
	}

	if len(lines) > leftInnerH-1 {
		lines = lines[:leftInnerH-1]
	}

	for len(lines) < leftInnerH-1 {
		lines = append(lines, "")
	}

	lines = append(lines, footer)

	return strings.Join(lines, "\n")
}

func (m tuiModel) treeFooterLine(leftW int, statusColor string, compact bool) string {
	pos, total := m.selectedSessionPosition()
	high, medium := m.riskCounts()
	warn := high + medium

	posPart := lipgloss.NewStyle().
		Foreground(lipgloss.Color(statusColor)).
		Render(fmt.Sprintf("%d/%d", pos, total))

	warnColor := m.colorHex("status_info")
	if high > 0 {
		warnColor = m.colorHex("status_risk")
	} else if warn > 0 {
		warnColor = m.colorHex("status_warn")
	}

	warnPart := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(warnColor)).
		Render(fmt.Sprintf("WARN: %d", warn))

	riskColor := m.colorHex("status_info")
	if high > 0 {
		riskColor = m.colorHex("status_risk")
	}

	riskPart := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(riskColor)).
		Render(fmt.Sprintf("RISK: %d", high))

	if compact {
		warnPart = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(warnColor)).
			Render(fmt.Sprintf("W:%d", warn))
		riskPart = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(riskColor)).
			Render(fmt.Sprintf("R:%d", high))
	}

	content := posPart + " | " + warnPart + " " + riskPart

	return lipgloss.NewStyle().
		Width(max(8, leftW)).
		AlignHorizontal(lipgloss.Center).
		Render(content)
}

func (m tuiModel) selectedSessionPosition() (int, int) {
	if len(m.tree) == 0 {
		return 0, 0
	}

	total := 0
	position := 0

	for i, item := range m.tree {
		if item.Kind != treeItemSession {
			continue
		}

		total++
		if i == m.cursor {
			position = total
		}
	}

	if total == 0 {
		return 0, 0
	}

	if m.cursor < 0 || m.cursor >= len(m.tree) {
		return 0, total
	}

	if position == 0 {
		return 0, total
	}

	return position, total
}

func (m *tuiModel) appendSelectedSessionPreview(
	previewLines, infoLines *[]string,
	selected session.Session,
	rightW int,
) {
	previewOuterH := Compute(m.width, m.height).PreviewOuterH
	rightBase := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
	previewInnerH := max(2, previewOuterH-rightBase.GetVerticalFrameSize())
	// PREVIEW pane reserves 5 fixed rows: title/status/risk/scroll/bar.
	previewContentHeight := max(1, previewInnerH-5)
	previewTextWidth := max(8, rightW-8)

	key := preview.CacheKeyForSession(selected.Path, previewTextWidth, selected.SizeBytes, selected.UpdatedAt.UnixNano())

	preview, ok := m.previewCachePeek(key)
	if !ok {
		if m.previewWait == key {
			preview = []string{" loading preview..."}
		} else {
			preview = []string{" preview not ready"}
		}
	}

	maybeMax := max(0, len(preview)-previewContentHeight)
	if m.previewOffset > maybeMax {
		m.previewOffset = maybeMax
	}

	start := m.previewOffset
	end := start + previewContentHeight
	end = min(end, len(preview))

	scrollInfo := fmt.Sprintf(" scroll %d-%d/%d ", start+1, end, len(preview))
	scrollStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.colorHex("scroll")))

	barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.colorHex("bar")))
	if m.focus == focusPreview {
		scrollStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(m.colorHex("scroll_active")))
		barStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(m.colorHex("bar_active")))
	}

	*previewLines = append(
		*previewLines,
		scrollStyle.Render(truncateDisplay(scrollInfo, previewTextWidth)),
	)
	*previewLines = append(
		*previewLines,
		barStyle.Render(
			" "+buildPreviewScrollBar(start, end, len(preview), max(10, previewTextWidth-2)),
		),
	)
	*previewLines = append(*previewLines, preview[start:end]...)

	h, v := m.detailRows(selected, rightW)
	*infoLines = append(
		*infoLines,
		lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(m.colorHex("info_header"))).
			Render(h),
	)
	*infoLines = append(*infoLines, v)
}

func fitStyledWidth(v string, width int) string {
	if width <= 0 {
		return ""
	}

	w := lipgloss.Width(v)
	if w >= width {
		return v
	}

	return v + strings.Repeat(" ", width-w)
}

func (m tuiModel) compactMode() bool {
	return IsCompactWidth(m.width)
}

func compactTreeGroupLabel(v string) string {
	if strings.HasPrefix(v, "▾ ") || strings.HasPrefix(v, "▸ ") {
		return strings.TrimSpace(v[3:])
	}

	return v
}

func (m tuiModel) renderBottomLine(width int) string {
	if m.pendingAction == "" {
		if m.compactMode() {
			return renderCompactKeysLine(width, m.theme)
		}

		return renderKeysLine(width, m.theme)
	}

	target := core.ShortID(strings.TrimSpace(m.pendingID))
	if target == "" {
		target = "-"
	}

	action := strings.ToUpper(strings.TrimSpace(m.pendingAction))
	plain := fmt.Sprintf("PENDING %s %s | Press Y to confirm, N to cancel", action, target)

	padded := truncateDisplay(plain, width)
	if w := lipgloss.Width(padded); w < width {
		padded += strings.Repeat(" ", width-w)
	}

	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.colorHex("bg"))).
		Background(lipgloss.Color(m.colorHex("tag_error"))).
		Render(padded)
}
