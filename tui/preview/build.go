package preview

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

// AngleTagTone classifies angle-bracket tag semantics for highlight styling.
type AngleTagTone int

const (
	AngleTagToneDefault AngleTagTone = iota
	AngleTagToneSystem
	AngleTagToneLifecycle
	AngleTagToneDanger
	AngleTagToneSuccess
)

// CacheKeyForSession derives a stable preview cache key from session metadata.
func CacheKeyForSession(path string, width int, sizeBytes int64, updatedAtUnix int64) string {
	return fmt.Sprintf("%s|w:%d|sz:%d|mt:%d", path, width, sizeBytes, updatedAtUnix)
}

// LinesBytes returns raw string bytes occupied by preview lines.
func LinesBytes(lines []string) int64 {
	total := int64(0)
	for _, line := range lines {
		total += int64(len(line))
	}

	return total
}

// BuildLines renders preview entries from one session file.
func BuildLines(path string, width, lines int, palette ThemePalette) []string {
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
		contentWidth := max(4, width-3)
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

			prefixStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(or(palette.PrefixDefault, "#c0caf5")))

			switch p {
			case "U":
				prefixStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(or(palette.PrefixUser, "#7aa2f7")))
			case "A":
				prefixStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(or(palette.PrefixAssistant, "#9ece6a")))
			case "?":
				prefixStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(or(palette.PrefixOther, "#bb9af7")))
			}

			row := prefixStyle.Render(prefixCell) + highlightAngleTags(chunk, palette)
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
			Foreground(lipgloss.Color(or(palette.TagDanger, "#f7768e"))).
			Render(msg))
	}

	if len(out) == 0 {
		out = append(out, " no dialogue preview available")
	}

	return out
}

func highlightAngleTags(v string, palette ThemePalette) string {
	if strings.TrimSpace(v) == "" {
		return v
	}

	matches := angleTagRe.FindAllStringIndex(v, -1)
	if len(matches) == 0 {
		return v
	}

	styleDefault := lipgloss.NewStyle().Foreground(lipgloss.Color(or(palette.TagDefault, "#9aa5ce")))
	styleSystem := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(or(palette.TagSystem, "#7dcfff")))
	styleLifecycle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(or(palette.TagLifecycle, "#bb9af7")))
	styleDanger := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(or(palette.TagDanger, "#f7768e")))
	styleSuccess := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(or(palette.TagSuccess, "#9ece6a")))

	var b strings.Builder

	last := 0
	for _, m := range matches {
		if m[0] > last {
			b.WriteString(v[last:m[0]])
		}

		tag := v[m[0]:m[1]]
		switch ClassifyAngleTag(tag) {
		case AngleTagToneDanger:
			b.WriteString(styleDanger.Render(tag))
		case AngleTagToneSystem:
			b.WriteString(styleSystem.Render(tag))
		case AngleTagToneLifecycle:
			b.WriteString(styleLifecycle.Render(tag))
		case AngleTagToneSuccess:
			b.WriteString(styleSuccess.Render(tag))
		case AngleTagToneDefault:
			b.WriteString(styleDefault.Render(tag))
		}

		last = m[1]
	}

	if last < len(v) {
		b.WriteString(v[last:])
	}

	return b.String()
}

// ClassifyAngleTag applies semantic tag classification for preview coloring.
func ClassifyAngleTag(tag string) AngleTagTone {
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
		return AngleTagToneDefault
	}

	if strings.Contains(name, "error") || strings.Contains(name, "fail") || strings.Contains(name, "abort") || strings.Contains(name, "panic") {
		return AngleTagToneDanger
	}

	if strings.Contains(name, "ok") || strings.Contains(name, "success") || strings.Contains(name, "done") {
		return AngleTagToneSuccess
	}

	if strings.Contains(name, "mode") || strings.Contains(name, "context") || strings.Contains(name, "permission") || strings.Contains(name, "sandbox") || strings.Contains(name, "instruction") {
		return AngleTagToneSystem
	}

	if strings.Contains(name, "turn") || strings.Contains(name, "session") || strings.Contains(name, "meta") || strings.Contains(name, "event") {
		return AngleTagToneLifecycle
	}

	return AngleTagToneDefault
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

	var (
		out []string
		b   strings.Builder
	)

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

func or(v, fallback string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}

	return fallback
}
