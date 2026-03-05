package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
	cmd.Flags().IntVarP(&limit, "limit", "l", 100, "max sessions loaded into TUI (0 means unlimited)")
	cmd.Flags().StringVar(&groupBy, "group-by", "", "tree group key: month|day|health|host|none")
	cmd.Flags().StringVar(&source, "source", "", "session source: sessions|trash")
	cmd.Flags().StringVar(&themeName, "theme", "", "TUI theme: tokyonight|catppuccin|gruvbox|onedark|nord|dracula")
	cmd.Flags().StringArrayVar(&themeColors, "theme-color", nil, "custom theme override (key=value), repeatable")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", true, "simulate delete/restore from TUI")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "required for real delete/restore from TUI")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip TUI confirmation prompts")
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
