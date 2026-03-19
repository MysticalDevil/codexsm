package doctor

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/MysticalDevil/codexsm/usecase"
	"github.com/charmbracelet/x/term"
	"github.com/mattn/go-runewidth"
)

const (
	ansiReset    = "\x1b[0m"
	ansiGreen    = "\x1b[32m"
	ansiYellow   = "\x1b[33m"
	ansiRed      = "\x1b[31m"
	ansiCyanBold = "\x1b[1;36m"

	detailContinuationIndent = "   "
)

func colorize(v, color string, enabled bool) string {
	if !enabled || color == "" {
		return v
	}

	return color + v + ansiReset
}

func renderChecks(checks []usecase.DoctorCheck, color bool) string {
	var buf bytes.Buffer

	checkW := len("CHECK")

	statusW := len("STATUS")
	for _, c := range checks {
		if len(c.Name) > checkW {
			checkW = len(c.Name)
		}

		if len(c.Level) > statusW {
			statusW = len(c.Level)
		}
	}

	headCheck := fmt.Sprintf("%-*s", checkW, "CHECK")
	headStatus := fmt.Sprintf("%-*s", statusW, "STATUS")
	headDetail := "DETAIL"

	if color {
		headCheck = colorize(headCheck, ansiCyanBold, true)
		headStatus = colorize(headStatus, ansiCyanBold, true)
		headDetail = colorize(headDetail, ansiCyanBold, true)
	}

	_, _ = fmt.Fprintf(&buf, "%s  %s  %s\n", headCheck, headStatus, headDetail)
	detailWrapW := detailWrapWidth(checkW, statusW)

	for _, c := range checks {
		status := fmt.Sprintf("%-*s", statusW, string(c.Level))
		if color {
			switch c.Level {
			case Pass:
				status = colorize(status, ansiGreen, true)
			case Warn:
				status = colorize(status, ansiYellow, true)
			case Fail:
				status = colorize(status, ansiRed, true)
			}
		}

		lines := detailLines(c.Detail)
		if len(lines) == 0 {
			lines = []string{""}
		}
		lines = wrapDetailLines(lines, detailWrapW)

		_, _ = fmt.Fprintf(&buf, "%-*s  %s  %s\n", checkW, c.Name, status, lines[0])
		for _, line := range lines[1:] {
			_, _ = fmt.Fprintf(&buf, "%s  %s  %s\n", strings.Repeat(" ", checkW), strings.Repeat(" ", statusW), line)
		}
	}

	return buf.String()
}

func detailLines(detail string) []string {
	d := strings.TrimSpace(detail)
	if d == "" {
		return nil
	}

	raw := strings.Split(d, "\n")

	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		out = append(out, line)
	}

	return out
}

func detailWrapWidth(checkW, statusW int) int {
	cols, ok := terminalColumns()
	if !ok || cols <= 0 {
		return 0
	}

	prefixW := checkW + 2 + statusW + 2
	available := cols - prefixW
	if available <= 0 {
		return 1
	}

	return available
}

func terminalColumns() (int, bool) {
	if colsRaw, ok := os.LookupEnv("COLUMNS"); ok {
		cols, err := strconv.Atoi(strings.TrimSpace(colsRaw))
		if err == nil && cols > 0 {
			return cols, true
		}
	}

	if term.IsTerminal(os.Stdout.Fd()) {
		if cols, _, err := term.GetSize(os.Stdout.Fd()); err == nil && cols > 0 {
			return cols, true
		}
	}

	if term.IsTerminal(os.Stderr.Fd()) {
		if cols, _, err := term.GetSize(os.Stderr.Fd()); err == nil && cols > 0 {
			return cols, true
		}
	}

	return 0, false
}

func wrapDetailLines(lines []string, width int) []string {
	if len(lines) == 0 {
		return lines
	}

	if width <= 0 {
		return lines
	}

	out := make([]string, 0, len(lines))
	for _, line := range lines {
		wrapped := wrapLineByWidth(line, width, width-len(detailContinuationIndent))
		if len(wrapped) == 0 {
			out = append(out, "")
			continue
		}

		out = append(out, wrapped[0])
		for _, cont := range wrapped[1:] {
			out = append(out, detailContinuationIndent+cont)
		}
	}

	return out
}

func wrapLineByWidth(v string, firstWidth, continuationWidth int) []string {
	if firstWidth <= 0 {
		return []string{v}
	}
	if continuationWidth <= 0 {
		continuationWidth = 1
	}

	words := strings.Fields(v)
	if len(words) == 0 {
		return []string{""}
	}

	out := make([]string, 0, 4)
	current := words[0]
	currentLimit := firstWidth

	if runewidth.StringWidth(current) > currentLimit {
		split := wrapRunesByWidth(current, currentLimit, continuationWidth)
		out = append(out, split[:len(split)-1]...)
		current = split[len(split)-1]
		currentLimit = continuationWidth
	}

	for _, w := range words[1:] {
		candidate := current + " " + w
		if runewidth.StringWidth(candidate) <= currentLimit {
			current = candidate
			continue
		}

		out = append(out, current)
		currentLimit = continuationWidth
		if runewidth.StringWidth(w) > currentLimit {
			split := wrapRunesByWidth(w, currentLimit, continuationWidth)
			out = append(out, split[:len(split)-1]...)
			current = split[len(split)-1]
			continue
		}

		current = w
	}

	out = append(out, current)
	return out
}

func wrapRunesByWidth(v string, firstWidth, continuationWidth int) []string {
	if firstWidth <= 0 {
		return []string{v}
	}
	if continuationWidth <= 0 {
		continuationWidth = 1
	}

	var (
		out     []string
		b       strings.Builder
		current int
	)
	limit := firstWidth

	for _, r := range v {
		rw := runewidth.RuneWidth(r)
		if rw <= 0 {
			rw = 1
		}

		if current+rw > limit && b.Len() > 0 {
			out = append(out, b.String())
			b.Reset()
			current = 0
			limit = continuationWidth
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
