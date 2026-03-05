package cli

import (
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/MysticalDevil/codexsm/session"
)

var ansiColorRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(v string) string {
	return ansiColorRe.ReplaceAllString(v, "")
}

func shouldUseColor(mode string, out io.Writer) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "always":
		return true
	case "never":
		return false
	case "", "auto":
		if strings.EqualFold(os.Getenv("NO_COLOR"), "1") || strings.TrimSpace(os.Getenv("NO_COLOR")) != "" {
			return false
		}
		return isTerminalWriter(out)
	default:
		return isTerminalWriter(out)
	}
}

func isTerminalWriter(out io.Writer) bool {
	f, ok := out.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

const (
	ansiReset    = "\x1b[0m"
	ansiGreen    = "\x1b[32m"
	ansiYellow   = "\x1b[33m"
	ansiRed      = "\x1b[31m"
	ansiDim      = "\x1b[2m"
	ansiCyanBold = "\x1b[1;36m"
)

func colorize(v, color string, enabled bool) string {
	if !enabled || color == "" {
		return v
	}
	return color + v + ansiReset
}

func colorizeRenderedTable(text string, sessions []session.Session, noHeader, hasHealth bool) string {
	if text == "" {
		return text
	}

	hasTrailingNewline := strings.HasSuffix(text, "\n")
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	dataStart := 0
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !noHeader && i == 0 {
			lines[i] = colorize(line, ansiCyanBold, true)
			dataStart = 1
			continue
		}
		if strings.HasPrefix(line, "showing ") {
			lines[i] = colorize(line, ansiDim, true)
			continue
		}
		if hasHealth {
			idx := i - dataStart
			if idx >= 0 && idx < len(sessions) {
				switch sessions[idx].Health {
				case session.HealthOK:
					lines[i] = colorize(line, ansiGreen, true)
				case session.HealthMissingMeta:
					lines[i] = colorize(line, ansiYellow, true)
				case session.HealthCorrupted:
					lines[i] = colorize(line, ansiRed, true)
				}
			}
		}
	}

	out := strings.Join(lines, "\n")
	if hasTrailingNewline {
		out += "\n"
	}
	return out
}

func shortID(id string) string {
	const max = 12
	if len(id) <= max {
		return id
	}
	return id[:max]
}
