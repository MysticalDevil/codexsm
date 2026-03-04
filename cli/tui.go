package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/MysticalDevil/codexsm/config"
	"github.com/MysticalDevil/codexsm/session"
)

type treeItemKind int

const (
	treeItemMonth treeItemKind = iota
	treeItemSession
)

type treeItem struct {
	kind   treeItemKind
	label  string
	month  string
	index  int
	indent int
}

type tuiFocus int

const (
	focusTree tuiFocus = iota
	focusPreview
)

type tuiModel struct {
	sessions      []session.Session
	tree          []treeItem
	cursor        int
	offset        int
	previewOffset int
	width         int
	height        int
	home          string
	sessionsRoot  string
	status        string
	previewCache  map[string][]string
	lastPath      string
	focus         tuiFocus
	groupBy       string
	source        string
	theme         tuiTheme
	trashRoot     string
	logFile       string
	dryRun        bool
	confirm       bool
	yes           bool
	hardDelete    bool
	maxBatch      int
	pendingAction string
	pendingID     string
}

const (
	tuiMinWidth  = 100
	tuiMinHeight = 24
)

func newTUICmd() *cobra.Command {
	var (
		sessionsRoot string
		trashRoot    string
		logFile      string
		limit        int
		groupBy      string
		source       string
		themeName    string
		themeColors  []string
		dryRun       bool
		confirm      bool
		yes          bool
		hardDelete   bool
		maxBatch     int
	)

	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive TUI session browser",
		Long: "Interactive session browser (optional).\n\n" +
			"Keys:\n" +
			"  j/k or Down/Up: move cursor\n" +
			"  g/G: first/last\n" +
			"  Ctrl+d / Ctrl+u: scroll preview\n" +
			"  d: delete current session (respects --dry-run/--confirm)\n" +
			"  r: restore current session (only when --source=trash)\n" +
			"  y/n: confirm/cancel pending action\n" +
			"  q: quit",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(sessionsRoot) == "" {
				v, err := runtimeSessionsRoot()
				if err != nil {
					return err
				}
				sessionsRoot = v
			} else {
				v, err := config.ResolvePath(sessionsRoot)
				if err != nil {
					return err
				}
				sessionsRoot = v
			}
			if strings.TrimSpace(trashRoot) == "" {
				v, err := runtimeTrashRoot()
				if err != nil {
					return err
				}
				trashRoot = v
			} else {
				v, err := config.ResolvePath(trashRoot)
				if err != nil {
					return err
				}
				trashRoot = v
			}
			if strings.TrimSpace(logFile) == "" {
				v, err := runtimeLogFile()
				if err != nil {
					return err
				}
				logFile = v
			} else {
				v, err := config.ResolvePath(logFile)
				if err != nil {
					return err
				}
				logFile = v
			}

			if strings.TrimSpace(source) == "" {
				source = strings.TrimSpace(runtimeConfig.TUI.Source)
			}
			source = strings.ToLower(strings.TrimSpace(source))
			if source == "" {
				source = "sessions"
			}
			if source != "sessions" && source != "trash" {
				return fmt.Errorf("invalid --source %q (allowed: sessions, trash)", source)
			}
			scanRoot := sessionsRoot
			if source == "trash" {
				scanRoot = filepath.Join(trashRoot, "sessions")
			}

			items, err := session.ScanSessions(scanRoot)
			if err != nil {
				return err
			}
			sortTUISessions(items)
			if limit > 0 && len(items) > limit {
				items = items[:limit]
			}

			home, _ := config.ResolvePath("~")
			if strings.TrimSpace(groupBy) == "" {
				groupBy = strings.TrimSpace(runtimeConfig.TUI.GroupBy)
			}
			mode, err := normalizeTUIGroupBy(groupBy)
			if err != nil {
				return err
			}
			theme, err := resolveTUITheme(runtimeConfig.TUI.Theme, runtimeConfig.TUI.Colors, themeName, themeColors)
			if err != nil {
				return err
			}
			m := tuiModel{
				sessions:     items,
				home:         home,
				sessionsRoot: sessionsRoot,
				status:       "Ready. Press q to quit.",
				previewCache: make(map[string][]string),
				focus:        focusTree,
				groupBy:      mode,
				source:       source,
				theme:        theme,
				trashRoot:    trashRoot,
				logFile:      logFile,
				dryRun:       dryRun,
				confirm:      confirm,
				yes:          yes,
				hardDelete:   hardDelete,
				maxBatch:     maxBatch,
			}
			m.rebuildTree()
			_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
			return err
		},
	}

	cmd.Flags().StringVar(&sessionsRoot, "sessions-root", "", "sessions root directory")
	cmd.Flags().StringVar(&trashRoot, "trash-root", "", "trash root directory")
	cmd.Flags().StringVar(&logFile, "log-file", "", "action log file")
	cmd.Flags().IntVar(&limit, "limit", 100, "max sessions loaded into TUI (0 means unlimited)")
	cmd.Flags().StringVar(&groupBy, "group-by", "", "tree group key: month|day|health|host|none")
	cmd.Flags().StringVar(&source, "source", "", "session source: sessions|trash")
	cmd.Flags().StringVar(&themeName, "theme", "", "TUI theme: tokyonight|catppuccin|gruvbox|onedark|nord|dracula")
	cmd.Flags().StringArrayVar(&themeColors, "theme-color", nil, "custom theme override (key=value), repeatable")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "simulate delete/restore from TUI")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "required for real delete/restore from TUI")
	cmd.Flags().BoolVar(&yes, "yes", false, "skip TUI confirmation prompts")
	cmd.Flags().BoolVar(&hardDelete, "hard", false, "hard delete on session source")
	cmd.Flags().IntVar(&maxBatch, "max-batch", 50, "max sessions allowed for one real TUI action")
	return cmd
}

func (m tuiModel) Init() tea.Cmd {
	return nil
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampOffset()
		return m, nil
	case tea.KeyMsg:
		if m.handleKey(msg.String()) {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *tuiModel) handleKey(key string) bool {
	switch key {
	case "q", "ctrl+c":
		return true
	case "tab", "ctrl+i":
		if m.focus == focusTree {
			m.focus = focusPreview
		} else {
			m.focus = focusTree
		}
	case "shift+tab", "backtab":
		if m.focus == focusPreview {
			m.focus = focusTree
		} else {
			m.focus = focusPreview
		}
	case "right", "l":
		m.focus = focusPreview
	case "left", "h":
		m.focus = focusTree
	case "t", "1":
		m.focus = focusTree
	case "p", "2":
		m.focus = focusPreview
	case "j", "down":
		if m.focus == focusTree {
			if m.cursor < len(m.tree)-1 {
				m.cursor++
			}
			m.skipToSelectable(1)
			m.syncPreviewSelection()
			m.clampOffset()
		} else {
			m.previewOffset++
		}
	case "k", "up":
		if m.focus == focusTree {
			if m.cursor > 0 {
				m.cursor--
			}
			m.skipToSelectable(-1)
			m.syncPreviewSelection()
			m.clampOffset()
		} else {
			m.previewOffset--
			if m.previewOffset < 0 {
				m.previewOffset = 0
			}
		}
	case "g":
		if m.focus == focusTree {
			m.cursor = 0
			m.skipToSelectable(1)
			m.syncPreviewSelection()
			m.clampOffset()
		} else {
			m.previewOffset = 0
		}
	case "G":
		if m.focus == focusTree {
			if len(m.tree) > 0 {
				m.cursor = len(m.tree) - 1
			}
			m.skipToSelectable(-1)
			m.syncPreviewSelection()
			m.clampOffset()
		} else {
			m.previewOffset = 1 << 30 // clamped in View by preview length
		}
	case "ctrl+d":
		if m.focus == focusPreview {
			m.previewOffset += m.previewPageStep()
		}
	case "ctrl+u":
		if m.focus == focusPreview {
			m.previewOffset -= m.previewPageStep()
			if m.previewOffset < 0 {
				m.previewOffset = 0
			}
		}
	case "d":
		m.requestDelete()
	case "r":
		m.requestRestore()
	case "y":
		m.commitPendingAction()
	case "n", "esc":
		m.cancelPendingAction()
	}
	return false
}

func (m tuiModel) View() string {
	totalW := m.width
	totalH := m.height
	if totalW <= 0 {
		totalW = 120
	}
	if totalH <= 0 {
		totalH = 32
	}

	borderColor := m.colorHex("border")
	borderFocusColor := m.colorHex("border_focus")
	fgColor := m.colorHex("fg")
	bgColor := m.colorHex("bg")
	statusColor := m.colorHex("status")

	keysPanelStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Foreground(lipgloss.Color(fgColor)).
		Background(lipgloss.Color(bgColor)).
		Padding(0, 1)
	keysOuterH := 3
	keysInnerW := max(20, totalW-keysPanelStyle.GetHorizontalFrameSize())
	keybar := keysPanelStyle.
		Width(keysInnerW).
		Bold(true).
		Render(renderKeysLine(keysInnerW, m.theme))

	if m.width > 0 && m.height > 0 && (m.width < tuiMinWidth || m.height < tuiMinHeight) {
		msg := fmt.Sprintf(
			"Terminal too small.\nRequired at least: %dx%d\nCurrent: %dx%d\nResize terminal and try again. Press q to quit.",
			tuiMinWidth, tuiMinHeight, m.width, m.height,
		)
		mainAreaH := max(6, totalH-keysOuterH)
		warn := lipgloss.NewStyle().
			Width(max(32, totalW-2)).
			Height(max(4, mainAreaH-2)).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(borderColor)).
			Foreground(lipgloss.Color(fgColor)).
			Background(lipgloss.Color(bgColor)).
			Padding(1, 2).
			Render(msg)
		return strings.Join([]string{warn, keybar}, "\n")
	}

	mainAreaH := max(8, totalH-keysOuterH)

	if len(m.sessions) == 0 || len(m.tree) == 0 {
		empty := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).
			Render("No sessions found.")
		emptyPane := lipgloss.NewStyle().
			Width(max(32, totalW-2)).
			Height(max(4, mainAreaH-2)).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(borderColor)).
			Foreground(lipgloss.Color(fgColor)).
			Background(lipgloss.Color(bgColor)).
			Padding(0, 1).
			Render(empty + "\n" + m.status)
		return strings.Join([]string{keybar, emptyPane}, "\n")
	}

	gapW := 1
	leftOuterW := int(float64(totalW) * 0.33)
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

	infoOuterH := 4 // border + exactly 2 text rows, no blank
	if infoOuterH >= mainAreaH-4 {
		infoOuterH = max(3, mainAreaH/4)
	}
	previewOuterH := mainAreaH - infoOuterH
	if previewOuterH < 5 {
		previewOuterH = 5
		infoOuterH = max(3, mainAreaH-previewOuterH)
	}

	leftBase := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Foreground(lipgloss.Color(fgColor)).
		Background(lipgloss.Color(bgColor)).
		Padding(0, 1)
	rightBase := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Foreground(lipgloss.Color(fgColor)).
		Background(lipgloss.Color(bgColor)).
		Padding(0, 1)
	infoBase := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Foreground(lipgloss.Color(fgColor)).
		Background(lipgloss.Color(bgColor)).
		Padding(0, 1)

	leftW := max(12, leftOuterW-leftBase.GetHorizontalFrameSize())
	rightW := max(12, rightOuterW-rightBase.GetHorizontalFrameSize())

	leftTitleText := "SESSIONS"
	if gb := strings.ToLower(strings.TrimSpace(m.groupBy)); gb != "" && gb != "none" {
		leftTitleText = fmt.Sprintf("SESSIONS (By %s)", strings.ToUpper(gb[:1])+gb[1:])
	}
	rightTitleText := "PREVIEW"
	if m.focus == focusTree {
		leftTitleText += " *"
	} else {
		rightTitleText += " *"
	}
	leftTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.colorHex("title_tree"))).Render(leftTitleText)
	rightTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.colorHex("title_preview"))).Render(rightTitleText)

	leftLines := make([]string, 0, m.visibleRows()+1)
	previewLines := make([]string, 0, m.visibleRows()+1)
	infoLines := make([]string, 0, 4)
	leftLines = append(leftLines, leftTitle)
	previewLines = append(previewLines, rightTitle)
	previewLines = append(previewLines, lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(" "+truncateDisplay(m.status, max(8, rightW-2))))

	start, end := m.visibleRange()
	for i := start; i < end; i++ {
		item := m.tree[i]
		if item.kind == treeItemMonth {
			line := truncateDisplay(item.label, leftW-4)
			line = lipgloss.NewStyle().Foreground(lipgloss.Color(m.colorHex("group"))).Render(line)
			leftLines = append(leftLines, "  "+line)
			continue
		}

		connector := "├─"
		if i+1 >= len(m.tree) || m.tree[i+1].kind == treeItemMonth {
			connector = "└─"
		}
		connectorPart := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render("  " + connector + " ")
		idWidth := max(4, leftW-10)
		idText := truncateDisplay(item.label, idWidth)

		if i == m.cursor {
			if m.focus == focusTree {
				idText = lipgloss.NewStyle().
					Foreground(lipgloss.Color(m.colorHex("selected_fg"))).
					Background(lipgloss.Color(m.colorHex("selected_bg"))).
					Bold(true).Render(idText)
				leftLines = append(leftLines, lipgloss.NewStyle().Foreground(lipgloss.Color(m.colorHex("cursor_active"))).Render("▌")+" "+connectorPart+idText)
			} else {
				idText = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.colorHex("cursor_inactive"))).Render(idText)
				leftLines = append(leftLines, lipgloss.NewStyle().Foreground(lipgloss.Color(m.colorHex("cursor_inactive"))).Render("▏")+" "+connectorPart+idText)
			}
		} else {
			leftLines = append(leftLines, "  "+connectorPart+idText)
		}
	}

	selected, ok := m.selectedSession()
	infoInnerH := max(1, infoOuterH-infoBase.GetVerticalFrameSize())
	if infoInnerH > 2 {
		infoInnerH = 2
	}
	previewInnerH := max(2, previewOuterH-rightBase.GetVerticalFrameSize())
	previewContentHeight := max(2, previewInnerH-4) // title + status + scroll + bar
	previewTextWidth := max(8, rightW-8)
	if ok {
		preview := m.previewFor(selected.Path, previewTextWidth, previewContentHeight)
		maybeMax := max(0, len(preview)-previewContentHeight)
		if m.previewOffset > maybeMax {
			m.previewOffset = maybeMax
		}
		start := m.previewOffset
		end := start + previewContentHeight
		if end > len(preview) {
			end = len(preview)
		}
		scrollInfo := fmt.Sprintf(" scroll %d-%d/%d ", start+1, end, len(preview))
		scrollStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.colorHex("scroll")))
		barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.colorHex("bar")))
		if m.focus == focusPreview {
			scrollStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.colorHex("scroll_active")))
			barStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.colorHex("bar_active")))
		}
		previewLines = append(previewLines, scrollStyle.Render(truncateDisplay(scrollInfo, previewTextWidth)))
		previewLines = append(previewLines, barStyle.Render(" "+buildPreviewScrollBar(start, end, len(preview), max(10, previewTextWidth-2))))
		previewLines = append(previewLines, preview[start:end]...)
		h, v := m.detailRows(selected)
		infoLines = append(infoLines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.colorHex("info_header"))).Render(h))
		infoLines = append(infoLines, lipgloss.NewStyle().Foreground(lipgloss.Color(m.colorHex("info_value"))).Render(v))
	} else {
		previewLines = append(previewLines, " Select a session node")
		infoLines = append(infoLines, lipgloss.NewStyle().Foreground(lipgloss.Color(m.colorHex("tag_danger"))).Render("No session selected"))
	}

	leftBorder := borderColor
	rightBorder := borderColor
	if m.focus == focusTree {
		leftBorder = borderFocusColor
	} else {
		rightBorder = borderFocusColor
	}

	leftPane := leftBase.
		Width(leftW).
		Height(max(2, mainAreaH-leftBase.GetVerticalFrameSize())).
		BorderForeground(lipgloss.Color(leftBorder)).
		Render(strings.Join(leftLines, "\n"))

	previewPane := rightBase.
		Width(rightW).
		Height(previewInnerH).
		BorderForeground(lipgloss.Color(rightBorder)).
		Render(strings.Join(previewLines, "\n"))

	infoBorder := borderColor
	if m.focus == focusPreview {
		infoBorder = borderFocusColor
	}
	infoPane := infoBase.
		Width(rightW).
		Height(infoInnerH).
		BorderForeground(lipgloss.Color(infoBorder)).
		Render(strings.Join(infoLines[:minInt(len(infoLines), 2)], "\n"))

	rightBlock := lipgloss.JoinVertical(lipgloss.Left, infoPane, previewPane)
	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, strings.Repeat(" ", gapW), rightBlock)

	return strings.Join([]string{
		mainArea,
		keybar,
	}, "\n")
}
