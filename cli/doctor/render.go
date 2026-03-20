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
	ansiBlue     = "\x1b[34m"
	ansiMagenta  = "\x1b[35m"
	ansiDim      = "\x1b[2m"
	ansiCyanBold = "\x1b[1;36m"

	detailContinuationIndent = "   "
)

func colorize(v, color string, enabled bool) string {
	if !enabled || color == "" {
		return v
	}

	return color + v + ansiReset
}

func renderChecks(checks []usecase.DoctorCheck, color bool, compactHomePath bool) string {
	var buf bytes.Buffer

	home := ""
	if compactHomePath {
		home, _ = os.UserHomeDir()
	}

	checkW := len("CHECK")

	statusW := len("STATUS")

	for _, c := range checks {
		checkName := strings.TrimSpace(c.Name)
		if len(checkName) > checkW {
			checkW = len(checkName)
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

		lines = normalizeDetailLines(lines, compactHomePath, home)
		lines = wrapDetailLines(lines, detailWrapW)

		checkName := strings.TrimSpace(c.Name)

		first := lines[0]
		if color {
			first = colorizeDetailLine(first)
		}

		_, _ = fmt.Fprintf(&buf, "%-*s  %s  %s\n", checkW, checkName, status, first)

		for _, line := range lines[1:] {
			if color {
				line = colorizeDetailLine(line)
			}

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

func normalizeDetailLines(lines []string, compactHomePath bool, home string) []string {
	if !compactHomePath || home == "" || len(lines) == 0 {
		return lines
	}

	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, normalizeDetailLine(line, home))
	}

	return out
}

func normalizeDetailLine(line, home string) string {
	tokens := strings.Fields(line)
	if len(tokens) == 0 {
		return line
	}

	out := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		out = append(out, compactHomePathToken(tok, home))
	}

	return strings.Join(out, " ")
}

func colorizeDetailLine(line string) string {
	prefixLen := len(line) - len(strings.TrimLeft(line, " "))
	prefix := line[:prefixLen]

	content := strings.TrimLeft(line, " ")
	if content == "" {
		return line
	}

	tokens := strings.Fields(content)

	styled := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		styled = append(styled, colorizeDetailToken(tok))
	}

	return prefix + strings.Join(styled, " ")
}

func colorizeDetailToken(tok string) string {
	leading, core, trailing := splitTokenAffixes(tok)
	if core == "" {
		return tok
	}

	styledCore := core

	lower := strings.ToLower(core)
	switch {
	case lower == "codexsm":
		styledCore = colorize(core, ansiCyanBold, true)
	case lower == "list":
		styledCore = colorize(core, ansiGreen, true)
	case lower == "delete":
		styledCore = colorize(core, ansiRed, true)
	case lower == "doctor":
		styledCore = colorize(core, ansiBlue, true)
	case strings.HasPrefix(core, "--"):
		styledCore = colorize(core, ansiYellow, true)
	case strings.HasPrefix(core, "/"), strings.HasPrefix(core, "~/"):
		styledCore = colorize(core, ansiDim, true)
	case strings.HasSuffix(core, "."):
		styledCore = colorize(core, ansiDim, true)
	}

	return leading + styledCore + trailing
}

func compactHomePathToken(tok, home string) string {
	if home == "" {
		return tok
	}

	leading, core, trailing := splitTokenAffixes(tok)
	if core == "" {
		return tok
	}

	replaced := compactHomePathCore(core, home)

	return leading + replaced + trailing
}

func compactHomePathCore(pathToken, home string) string {
	if pathToken == home {
		return "~"
	}

	if strings.HasPrefix(pathToken, home+"/") {
		return "~" + pathToken[len(home):]
	}

	return pathToken
}

func splitTokenAffixes(tok string) (leading, core, trailing string) {
	start := 0
	end := len(tok)

	for start < end {
		switch tok[start] {
		case '(', '[', '{', '"', '\'':
			start++
		default:
			goto right
		}
	}

right:
	for end > start {
		switch tok[end-1] {
		case ',', ';', ')', ']', '}', '"', '\'':
			end--
		default:
			break right
		}
	}

	return tok[:start], tok[start:end], tok[end:]
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
