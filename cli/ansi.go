package cli

import (
	"io"
	"os"
	"strings"
)

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
