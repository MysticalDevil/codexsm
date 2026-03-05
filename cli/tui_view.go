package cli

import (
	"fmt"
	"strings"

	"github.com/MysticalDevil/codexsm/internal/tui/layout"
	"github.com/MysticalDevil/codexsm/session"
	"github.com/charmbracelet/lipgloss"
)

func (m tuiModel) View() string {
	metrics := layout.Compute(m.width, m.height)
	borderColor := m.colorHex("border")
	borderFocusColor := m.colorHex("border_focus")
	fgColor := m.colorHex("fg")
	bgColor := m.colorHex("bg")
	statusColor := m.colorHex("status")

	keysPanelStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Foreground(lipgloss.Color(fgColor)).
		Background(lipgloss.Color(bgColor)).
		Padding(0, 1)
	keysInnerW := max(20, metrics.TotalW-keysPanelStyle.GetHorizontalFrameSize())
	keybar := keysPanelStyle.
		Width(keysInnerW).
		Bold(true).
		Render(renderKeysLine(keysInnerW, m.theme))

	if layout.IsTooSmall(m.width, m.height) {
		msg := fmt.Sprintf(
			"Terminal too small.\nRequired at least: %dx%d\nCurrent: %dx%d\nResize terminal and try again. Press q to quit.",
			layout.MinWidth, layout.MinHeight, m.width, m.height,
		)
		warn := lipgloss.NewStyle().
			Width(max(32, metrics.TotalW-2)).
			Height(max(4, metrics.MainAreaH-2)).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(borderColor)).
			Foreground(lipgloss.Color(fgColor)).
			Background(lipgloss.Color(bgColor)).
			Padding(1, 2).
			Render(msg)
		return strings.Join([]string{warn, keybar}, "\n")
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
			Background(lipgloss.Color(bgColor)).
			Padding(0, 1).
			Render(empty + "\n" + m.status)
		return strings.Join([]string{keybar, emptyPane}, "\n")
	}

	leftBase := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Foreground(lipgloss.Color(fgColor)).
		Background(lipgloss.Color(bgColor)).
		Padding(0, 1)
	rightBase := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Foreground(lipgloss.Color(fgColor)).
		Background(lipgloss.Color(bgColor)).
		Padding(0, 1)
	infoBase := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Foreground(lipgloss.Color(fgColor)).
		Background(lipgloss.Color(bgColor)).
		Padding(0, 1)

	leftW := max(12, metrics.LeftOuterW-leftBase.GetHorizontalFrameSize())
	rightW := max(12, metrics.RightOuterW-rightBase.GetHorizontalFrameSize())

	_, previewLines, infoLines := m.buildPanelLines(rightW, statusColor)
	selected, ok := m.selectedSession()
	if ok {
		m.appendSelectedSessionPreview(&previewLines, &infoLines, selected, rightW)
	} else {
		previewLines = append(previewLines, " Select a session node")
		infoLines = append(infoLines, lipgloss.NewStyle().Foreground(lipgloss.Color(m.colorHex("tag_danger"))).Render("No session selected"))
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
	if infoInnerH > 2 {
		infoInnerH = 2
	}
	infoPane := infoBase.
		Width(rightW).
		Height(infoInnerH).
		BorderForeground(lipgloss.Color(infoBorder)).
		Render(strings.Join(infoLines[:minInt(len(infoLines), 2)], "\n"))

	rightBlock := lipgloss.JoinVertical(lipgloss.Left, infoPane, previewPane)
	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, strings.Repeat(" ", metrics.GapW), rightBlock)
	return strings.Join([]string{mainArea, keybar}, "\n")
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
	leftTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.colorHex("title_tree"))).Render(leftTitleText)
	rightTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.colorHex("title_preview"))).Render(rightTitleText)

	leftLines := make([]string, 0, m.visibleRows()+1)
	previewLines := make([]string, 0, m.visibleRows()+1)
	infoLines := make([]string, 0, 4)
	leftLines = append(leftLines, leftTitle)
	previewLines = append(previewLines, rightTitle)
	previewLines = append(previewLines, lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(" "+truncateDisplay(m.status, max(8, rightW-2))))
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
	leftTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.colorHex("title_tree"))).Render(leftTitleText)
	leftLines := make([]string, 0, m.visibleRows()+1)
	leftLines = append(leftLines, leftTitle)

	start, end := m.visibleRange()
	for i := start; i < end; i++ {
		item := m.tree[i]
		if item.kind == treeItemMonth {
			line := truncateDisplay(item.label, leftW-4)
			line = lipgloss.NewStyle().Foreground(lipgloss.Color(m.colorHex("group"))).Render(line)
			leftLines = append(leftLines, "  "+line)
			continue
		}
		connector := "├─"
		if i+1 >= len(m.tree) || m.tree[i+1].kind == treeItemMonth {
			connector = "└─"
		}
		connectorPart := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render("  " + connector + " ")
		idWidth := max(4, leftW-10)
		idText := truncateDisplay(item.label, idWidth)
		if i == m.cursor {
			if m.focus == focusTree {
				idText = lipgloss.NewStyle().
					Foreground(lipgloss.Color(m.colorHex("selected_fg"))).
					Background(lipgloss.Color(m.colorHex("selected_bg"))).
					Bold(true).Render(idText)
				leftLines = append(leftLines, lipgloss.NewStyle().Foreground(lipgloss.Color(m.colorHex("cursor_active"))).Render("▌")+" "+connectorPart+idText)
			} else {
				idText = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.colorHex("cursor_inactive"))).Render(idText)
				leftLines = append(leftLines, lipgloss.NewStyle().Foreground(lipgloss.Color(m.colorHex("cursor_inactive"))).Render("▏")+" "+connectorPart+idText)
			}
		} else {
			leftLines = append(leftLines, "  "+connectorPart+idText)
		}
	}
	return leftLines
}

func (m *tuiModel) appendSelectedSessionPreview(previewLines, infoLines *[]string, selected session.Session, rightW int) {
	previewOuterH := layout.Compute(m.width, m.height).PreviewOuterH
	rightBase := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
	previewInnerH := max(2, previewOuterH-rightBase.GetVerticalFrameSize())
	previewContentHeight := max(2, previewInnerH-4)
	previewTextWidth := max(8, rightW-8)

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
	scrollStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.colorHex("scroll")))
	barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.colorHex("bar")))
	if m.focus == focusPreview {
		scrollStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.colorHex("scroll_active")))
		barStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.colorHex("bar_active")))
	}
	*previewLines = append(*previewLines, scrollStyle.Render(truncateDisplay(scrollInfo, previewTextWidth)))
	*previewLines = append(*previewLines, barStyle.Render(" "+buildPreviewScrollBar(start, end, len(preview), max(10, previewTextWidth-2))))
	*previewLines = append(*previewLines, preview[start:end]...)

	h, v := m.detailRows(selected)
	*infoLines = append(*infoLines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.colorHex("info_header"))).Render(h))
	*infoLines = append(*infoLines, lipgloss.NewStyle().Foreground(lipgloss.Color(m.colorHex("info_value"))).Render(v))
}
