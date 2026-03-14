package tui

const (
	// MinWidth is the minimal terminal width required by TUI.
	// This is driven by the main panes, not the keybar. The keybar can degrade
	// to shorter variants at narrower widths, so the minimum should reflect the
	// actual split-pane layout requirement plus the terminal-edge safety margin.
	MinWidth = 118
	// MinHeight is the minimal terminal height required by TUI.
	MinHeight = 24
)

// Metrics describes the top-level dimensions used by TUI panels.
type Metrics struct {
	TotalW        int
	TotalH        int
	KeysOuterH    int
	MainAreaH     int
	GapW          int
	LeftOuterW    int
	RightOuterW   int
	InfoOuterH    int
	PreviewOuterH int
}

// NormalizeSize applies fallback values when terminal size is unavailable.
func NormalizeSize(width, height int) (int, int) {
	if width <= 0 {
		width = 120
	}
	if height <= 0 {
		height = 32
	}
	return width, height
}

// RenderWidth returns a width safe for rendering without hitting the terminal's
// last column, which can trigger autowrap and break borders in some terminals.
func RenderWidth(width int) int {
	if width <= 1 {
		return width
	}
	return width - 1
}

// IsTooSmall reports whether current terminal size is below supported bounds.
func IsTooSmall(width, height int) bool {
	if width <= 0 || height <= 0 {
		return false
	}
	return RenderWidth(width) < MinWidth || height < MinHeight
}

// Compute calculates panel dimensions for a normalized terminal size.
func Compute(width, height int) Metrics {
	totalW, totalH := NormalizeSize(width, height)
	totalW = RenderWidth(totalW)

	keysOuterH := 3
	mainAreaH := max(8, totalH-keysOuterH)

	gapW := 1
	if totalW < 132 {
		gapW = 0
	}
	leftOuterW := int(float64(totalW) * 0.28)
	if leftOuterW < 28 {
		leftOuterW = 28
	}
	if leftOuterW > totalW-36-gapW {
		leftOuterW = max(28, totalW-36-gapW)
	}
	rightOuterW := totalW - leftOuterW - gapW
	if rightOuterW < 36 {
		rightOuterW = 36
		leftOuterW = max(28, totalW-rightOuterW-gapW)
	}
	if leftOuterW+gapW+rightOuterW > totalW {
		rightOuterW = max(36, totalW-leftOuterW-gapW)
	}

	infoOuterH := 4 // border + exactly 2 content rows
	if infoOuterH >= mainAreaH-4 {
		infoOuterH = max(3, mainAreaH/4)
	}
	previewOuterH := mainAreaH - infoOuterH
	if previewOuterH < 5 {
		previewOuterH = 5
		infoOuterH = max(3, mainAreaH-previewOuterH)
	}

	return Metrics{
		TotalW:        totalW,
		TotalH:        totalH,
		KeysOuterH:    keysOuterH,
		MainAreaH:     mainAreaH,
		GapW:          gapW,
		LeftOuterW:    leftOuterW,
		RightOuterW:   rightOuterW,
		InfoOuterH:    infoOuterH,
		PreviewOuterH: previewOuterH,
	}
}
