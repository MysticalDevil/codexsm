package tui

import (
	"time"

	"github.com/MysticalDevil/codexsm/internal/core"
)

func shortID(id string) string {
	return core.ShortID(id)
}

func formatDisplayTime(t time.Time) string {
	return core.FormatDisplayTime(t)
}

func formatBytesIEC(size int64) string {
	return core.FormatBytesIEC(size)
}

func compactHomePath(path, home string) string {
	return core.CompactHomePath(path, home)
}
