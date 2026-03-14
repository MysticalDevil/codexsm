package tui

import (
	"fmt"
	"strings"

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

	_, previewLines, infoLines := m.buildPanelLines(rightW, statusColor)
	selected, ok := m.selectedSession()
	if ok {
		m.appendSelectedSessionPreview(&previewLines, &infoLines, selected, rightW)
	} else {
		previewLines = append(previewLines, " Select a session node")
		infoLines = append(
			infoLines,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.colorHex("tag_danger"))).
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

	leftPane := leftBase.
		Width(leftW).
		Height(max(2, metrics.MainAreaH-leftBase.GetVerticalFrameSize())).
		BorderForeground(lipgloss.Color(leftBorder)).
		Render(strings.Join(m.renderTreeLines(leftW, statusColor), "\n"))

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

func (m tuiModel) buildPanelLines(rightW int, statusColor string) ([]string, []string, []string) {
	leftTitleText := "SESSIONS"
	if gb := strings.ToLower(strings.TrimSpace(m.groupBy)); gb != "" && gb != "none" {
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
	highRisk, mediumRisk := riskCounts(m.sessions)
	riskLine := fmt.Sprintf(
		" risk=%d (high=%d medium=%d) ",
		highRisk+mediumRisk,
		highRisk,
		mediumRisk,
	)
	riskColor := m.colorHex("tag_success")
	if highRisk > 0 {
		riskColor = m.colorHex("tag_error")
	} else if mediumRisk > 0 {
		riskColor = m.colorHex("tag_danger")
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
	leftTitleText := "SESSIONS"
	if gb := strings.ToLower(strings.TrimSpace(m.groupBy)); gb != "" && gb != "none" {
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

	start, end := m.visibleRange()
	for i := start; i < end; i++ {
		item := m.tree[i]
		if item.Kind == treeItemMonth {
			line := truncateDisplay(item.Label, leftW-4)
			line = lipgloss.NewStyle().Foreground(lipgloss.Color(m.colorHex("group"))).Render(line)
			leftLines = append(leftLines, "  "+line)
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

	key := previewCacheKeyForSession(selected, previewTextWidth)
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

func (m tuiModel) renderBottomLine(width int) string {
	if m.pendingAction == "" {
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
