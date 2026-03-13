package tui

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/MysticalDevil/codexsm/usecase"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

var angleTagRe = regexp.MustCompile(`<[^>\n]{1,80}>`)

type angleTagTone int

const (
	angleTagToneDefault angleTagTone = iota
	angleTagToneSystem
	angleTagToneLifecycle
	angleTagToneDanger
	angleTagToneSuccess
)

func (m *tuiModel) previewFor(path string, width, lines int) []string {
	key := fmt.Sprintf("%s|w:%d", path, width)
	if cached, ok := m.previewCacheGet(key); ok {
		return cached
	}
	out := buildPreviewLines(path, width, lines, m.theme)
	m.previewCachePut(key, out)
	return out
}

func buildPreviewLines(path string, width, lines int, theme tuiTheme) []string {
	if width < 10 {
		width = 10
	}
	if lines < 5 {
		lines = 5
	}

	const maxPreviewLines = 600
	out := make([]string, 0, min(maxPreviewLines, lines*10))
	items, err := usecase.ExtractPreviewMessages(path, maxPreviewLines)
	if err != nil && len(items) == 0 && !errors.Is(err, usecase.ErrPreviewEntryTooLong) {
		out = append(out, " failed to open preview: "+err.Error())
		return out
	}
	for _, item := range items {
		role, text := item.Role, item.Text
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
			prefixCell := fmt.Sprintf(" %s ", p)
			remaining := max(0, width-runewidth.StringWidth(prefixCell))
			chunk = truncateDisplay(chunk, remaining)

			prefixStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(previewColorHex(theme, "prefix_default")))
			switch p {
			case "U":
				prefixStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(previewColorHex(theme, "prefix_user")))
			case "A":
				prefixStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(previewColorHex(theme, "prefix_assistant")))
			case "?":
				prefixStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(previewColorHex(theme, "prefix_other")))
			}

			row := prefixStyle.Render(prefixCell) + highlightAngleTags(chunk, theme)
			out = append(out, row)
		}
	}
	if err != nil {
		msg := " preview unavailable: failed to read session entries"
		if errors.Is(err, usecase.ErrPreviewEntryTooLong) {
			msg = " preview unavailable: a session entry exceeds the safe preview limit"
		}
		out = append(out, lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(previewColorHex(theme, "tag_danger"))).
			Render(msg))
	}
	if len(out) == 0 {
		out = append(out, " no dialogue preview available")
	}
	return out
}

func previewColorHex(theme tuiTheme, key string) string {
	fallback := builtinThemes[defaultTUIThemeName()][key]
	if strings.TrimSpace(theme.Name) == "" || len(theme.Colors) == 0 {
		return fallback
	}
	return theme.hex(key, fallback)
}

func highlightAngleTags(v string, theme tuiTheme) string {
	if strings.TrimSpace(v) == "" {
		return v
	}
	matches := angleTagRe.FindAllStringIndex(v, -1)
	if len(matches) == 0 {
		return v
	}

	styleDefault := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.hex("tag_default", builtinThemes[defaultTUIThemeName()]["tag_default"])))
	styleSystem := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.hex("tag_system", builtinThemes[defaultTUIThemeName()]["tag_system"])))
	styleLifecycle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.hex("tag_lifecycle", builtinThemes[defaultTUIThemeName()]["tag_lifecycle"])))
	styleDanger := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.hex("tag_danger", builtinThemes[defaultTUIThemeName()]["tag_danger"])))
	styleSuccess := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.hex("tag_success", builtinThemes[defaultTUIThemeName()]["tag_success"])))

	var b strings.Builder
	last := 0
	for _, m := range matches {
		if m[0] > last {
			b.WriteString(v[last:m[0]])
		}
		tag := v[m[0]:m[1]]
		switch classifyAngleTag(tag) {
		case angleTagToneDanger:
			b.WriteString(styleDanger.Render(tag))
		case angleTagToneSystem:
			b.WriteString(styleSystem.Render(tag))
		case angleTagToneLifecycle:
			b.WriteString(styleLifecycle.Render(tag))
		case angleTagToneSuccess:
			b.WriteString(styleSuccess.Render(tag))
		default:
			b.WriteString(styleDefault.Render(tag))
		}
		last = m[1]
	}
	if last < len(v) {
		b.WriteString(v[last:])
	}
	return b.String()
}

func classifyAngleTag(tag string) angleTagTone {
	name := strings.TrimSpace(tag)
	name = strings.TrimPrefix(name, "<")
	name = strings.TrimSuffix(name, ">")
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "/")
	if i := strings.IndexAny(name, " \t"); i >= 0 {
		name = name[:i]
	}
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return angleTagToneDefault
	}

	if strings.Contains(name, "error") || strings.Contains(name, "fail") || strings.Contains(name, "abort") || strings.Contains(name, "panic") {
		return angleTagToneDanger
	}
	if strings.Contains(name, "ok") || strings.Contains(name, "success") || strings.Contains(name, "done") {
		return angleTagToneSuccess
	}
	if strings.Contains(name, "mode") || strings.Contains(name, "context") || strings.Contains(name, "permission") || strings.Contains(name, "sandbox") || strings.Contains(name, "instruction") {
		return angleTagToneSystem
	}
	if strings.Contains(name, "turn") || strings.Contains(name, "session") || strings.Contains(name, "meta") || strings.Contains(name, "event") {
		return angleTagToneLifecycle
	}

	return angleTagToneDefault
}
