package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderKeysLine(width int, theme tuiTheme) string {
	variants := [][]keysSegment{
		{
			{label: "[KEYS]", kind: keysLabel},
			{label: " Tab/h/l t/p/1/2 ", kind: keysKey},
			{label: "switch pane", kind: keysText},
			{label: " | ", kind: keysSep},
			{label: "j/k", kind: keysKey},
			{label: " scroll", kind: keysText},
			{label: " | ", kind: keysSep},
			{label: "g/G", kind: keysKey},
			{label: " top/bottom", kind: keysText},
			{label: " | ", kind: keysSep},
			{label: "Ctrl+d/u", kind: keysKey},
			{label: " preview", kind: keysText},
			{label: " | ", kind: keysSep},
			{label: "d/r/m", kind: keysKey},
			{label: " action", kind: keysText},
			{label: " | ", kind: keysSep},
			{label: "y/n", kind: keysKey},
			{label: " confirm", kind: keysText},
			{label: " | ", kind: keysSep},
			{label: "q", kind: keysKey},
			{label: " quit", kind: keysText},
		},
		{
			{label: "[KEYS]", kind: keysLabel},
			{label: " Tab/h/l ", kind: keysKey},
			{label: "switch", kind: keysText},
			{label: " | ", kind: keysSep},
			{label: "j/k", kind: keysKey},
			{label: " scroll", kind: keysText},
			{label: " | ", kind: keysSep},
			{label: "g/G", kind: keysKey},
			{label: " top", kind: keysText},
			{label: " | ", kind: keysSep},
			{label: "d/r/m", kind: keysKey},
			{label: " action", kind: keysText},
			{label: " | ", kind: keysSep},
			{label: "q", kind: keysKey},
			{label: " quit", kind: keysText},
		},
		{
			{label: "[KEYS]", kind: keysLabel},
			{label: " d/r/m ", kind: keysKey},
			{label: "action", kind: keysText},
			{label: " | ", kind: keysSep},
			{label: "q", kind: keysKey},
			{label: " quit", kind: keysText},
		},
		{
			{label: "[KEYS]", kind: keysLabel},
			{label: " q ", kind: keysKey},
			{label: "quit", kind: keysText},
		},
	}

	if width <= 0 {
		return renderKeysSegments(variants[len(variants)-1], theme)
	}
	for _, variant := range variants {
		line := renderKeysSegments(variant, theme)
		if lipgloss.Width(line) <= width {
			return line
		}
	}
	return truncateDisplay(renderKeysSegments(variants[len(variants)-1], theme), width)
}

type keysSegmentKind int

const (
	keysLabel keysSegmentKind = iota
	keysKey
	keysText
	keysSep
)

type keysSegment struct {
	label string
	kind  keysSegmentKind
}

func renderKeysSegments(segments []keysSegment, theme tuiTheme) string {
	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		switch segment.kind {
		case keysLabel:
			parts = append(parts, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.hex("keys_label", builtinThemes[defaultTUIThemeName()]["keys_label"]))).Render(segment.label))
		case keysKey:
			parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_key", builtinThemes[defaultTUIThemeName()]["keys_key"]))).Render(segment.label))
		case keysText:
			parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_text", builtinThemes[defaultTUIThemeName()]["keys_text"]))).Render(segment.label))
		case keysSep:
			parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("keys_sep", builtinThemes[defaultTUIThemeName()]["keys_sep"]))).Render(segment.label))
		}
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
