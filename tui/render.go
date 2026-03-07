package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderKeysLine(width int, theme tuiTheme) string {
	plain := "[KEYS] Tab/h/l t/p/1/2 switch pane | j/k scroll | g/G top/bottom | Ctrl+d/u preview | d/r/m action | y/n confirm | q quit"
	if width <= 0 {
		return plain
	}
	if width < 72 {
		return truncateDisplay(plain, width)
	}
	parts := []string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.hex("keys_label", builtinThemes[defaultTUIThemeName()]["keys_label"]))).Render("[KEYS]"),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_key", builtinThemes[defaultTUIThemeName()]["keys_key"]))).Render(" Tab/h/l t/p/1/2 "),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_text", builtinThemes[defaultTUIThemeName()]["keys_text"]))).Render("switch pane"),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_sep", builtinThemes[defaultTUIThemeName()]["keys_sep"]))).Render(" | "),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_key", builtinThemes[defaultTUIThemeName()]["keys_key"]))).Render("j/k"),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_text", builtinThemes[defaultTUIThemeName()]["keys_text"]))).Render(" scroll"),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_sep", builtinThemes[defaultTUIThemeName()]["keys_sep"]))).Render(" | "),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_key", builtinThemes[defaultTUIThemeName()]["keys_key"]))).Render("g/G"),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_text", builtinThemes[defaultTUIThemeName()]["keys_text"]))).Render(" top/bottom"),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_sep", builtinThemes[defaultTUIThemeName()]["keys_sep"]))).Render(" | "),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_key", builtinThemes[defaultTUIThemeName()]["keys_key"]))).Render("Ctrl+d/u"),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_text", builtinThemes[defaultTUIThemeName()]["keys_text"]))).Render(" preview"),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_sep", builtinThemes[defaultTUIThemeName()]["keys_sep"]))).Render(" | "),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_key", builtinThemes[defaultTUIThemeName()]["keys_key"]))).Render("d/r"),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_text", builtinThemes[defaultTUIThemeName()]["keys_text"]))).Render(" action"),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_sep", builtinThemes[defaultTUIThemeName()]["keys_sep"]))).Render(" | "),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_key", builtinThemes[defaultTUIThemeName()]["keys_key"]))).Render("m"),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_text", builtinThemes[defaultTUIThemeName()]["keys_text"]))).Render(" migrate-host"),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_sep", builtinThemes[defaultTUIThemeName()]["keys_sep"]))).Render(" | "),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_key", builtinThemes[defaultTUIThemeName()]["keys_key"]))).Render("y/n"),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_text", builtinThemes[defaultTUIThemeName()]["keys_text"]))).Render(" confirm"),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_sep", builtinThemes[defaultTUIThemeName()]["keys_sep"]))).Render(" | "),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_key", builtinThemes[defaultTUIThemeName()]["keys_key"]))).Render("q"),
		lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_text", builtinThemes[defaultTUIThemeName()]["keys_text"]))).Render(" quit"),
	}
	return strings.Join(parts, "")
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

func (m tuiModel) colorHex(key string) string {
	theme := m.theme
	if strings.TrimSpace(theme.Name) == "" || len(theme.Colors) == 0 {
		theme = tuiTheme{
			Name:   defaultTUIThemeName(),
			Colors: cloneColorMap(builtinThemes[defaultTUIThemeName()]),
		}
	}
	fallback := builtinThemes[defaultTUIThemeName()][key]
	return theme.hex(key, fallback)
}
