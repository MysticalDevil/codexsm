package tui

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

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

func fitCellMiddle(v string, width int) string {
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(v) > width {
		v = truncateMiddleDisplay(v, width)
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

func truncateMiddleDisplay(v string, width int) string {
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(v) <= width {
		return v
	}
	if width <= 3 {
		return truncateDisplay(v, width)
	}
	leftTarget := (width - 3) / 2
	rightTarget := width - 3 - leftTarget
	left := takeDisplayPrefix(v, leftTarget)
	right := takeDisplaySuffix(v, rightTarget)
	return left + "..." + right
}

func takeDisplayPrefix(v string, width int) string {
	if width <= 0 {
		return ""
	}
	var b strings.Builder
	current := 0
	for _, r := range v {
		rw := runewidth.RuneWidth(r)
		if rw <= 0 {
			rw = 1
		}
		if current+rw > width {
			break
		}
		b.WriteRune(r)
		current += rw
	}
	return b.String()
}

func takeDisplaySuffix(v string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(v)
	current := 0
	start := len(runes)
	for i := len(runes) - 1; i >= 0; i-- {
		rw := runewidth.RuneWidth(runes[i])
		if rw <= 0 {
			rw = 1
		}
		if current+rw > width {
			break
		}
		current += rw
		start = i
	}
	return string(runes[start:])
}

func previewHostPath(host string, width int) string {
	if width <= 0 || host == "" {
		return host
	}
	if runewidth.StringWidth(host) <= width {
		return host
	}

	segs := strings.Split(strings.Trim(host, "/"), "/")
	if len(segs) >= 2 {
		tail := segs[len(segs)-2] + "/" + segs[len(segs)-1]
		if strings.HasPrefix(host, "~/") {
			candidate := "~/.../" + tail
			if runewidth.StringWidth(candidate) <= width {
				return candidate
			}
		} else {
			candidate := ".../" + tail
			if runewidth.StringWidth(candidate) <= width {
				return candidate
			}
		}
	}
	return truncateMiddleDisplay(host, width)
}
