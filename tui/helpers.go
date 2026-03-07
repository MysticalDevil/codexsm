package tui

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func formatDisplayTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

func formatBytesIEC(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	value := float64(size)
	unit := -1
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	return fmt.Sprintf("%.1f%s", value, units[unit])
}

func compactHomePath(path, home string) string {
	if home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	prefix := home + string(os.PathSeparator)
	if strings.HasPrefix(path, prefix) {
		return "~" + string(os.PathSeparator) + strings.TrimPrefix(path, prefix)
	}
	return path
}
