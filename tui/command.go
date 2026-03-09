package tui

import (
	"container/list"
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
	// hostMissing marks sessions whose host path does not exist on local filesystem.
	hostMissing bool
}

type tuiFocus int

const (
	focusTree tuiFocus = iota
	focusPreview
)

type tuiModel struct {
	sessions           []session.Session
	tree               []treeItem
	cursor             int
	offset             int
	previewOffset      int
	width              int
	height             int
	home               string
	sessionsRoot       string
	status             string
	previewCache       map[string][]string
	previewLRU         *list.List
	previewNodes       map[string]*list.Element
	previewBytesBudget int64
	previewBytesUsed   int64
	previewReqSeq      uint64
	previewReqID       uint64
	previewWait        string
	previewIndex       string
	indexCap           int
	lastPath           string
	focus              tuiFocus
	groupBy            string
	source             string
	theme              tuiTheme
	trashRoot          string
	logFile            string
	dryRun             bool
	confirm            bool
	yes                bool
	hardDelete         bool
	maxBatch           int
	pendingAction      string
	pendingID          string
	pendingHost        string
}

type CommandDeps struct {
	ResolveSessionsRoot func() (string, error)
	ResolveTrashRoot    func() (string, error)
	ResolveLogFile      func() (string, error)
	TUIConfig           config.TUIConfig
}

func NewCommand(deps CommandDeps) *cobra.Command {
	if deps.ResolveSessionsRoot == nil {
		deps.ResolveSessionsRoot = config.DefaultSessionsRoot
	}
	if deps.ResolveTrashRoot == nil {
		deps.ResolveTrashRoot = config.DefaultTrashRoot
	}
	if deps.ResolveLogFile == nil {
		deps.ResolveLogFile = config.DefaultLogFile
	}

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
			"  m: migrate sessions with missing selected host to trash\n" +
			"  r: restore current session (only when --source=trash)\n" +
			"  y/n: confirm/cancel pending action\n" +
			"  q: quit",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(sessionsRoot) == "" {
				v, err := deps.ResolveSessionsRoot()
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
				v, err := deps.ResolveTrashRoot()
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
				v, err := deps.ResolveLogFile()
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
				source = strings.TrimSpace(deps.TUIConfig.Source)
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

			items, err := session.ScanSessionsLimited(scanRoot, limit, func(a, b session.Session) bool {
				ra := session.EvaluateRisk(a, nil)
				rb := session.EvaluateRisk(b, nil)
				if c := riskLevelRank(rb.Level) - riskLevelRank(ra.Level); c != 0 {
					return c < 0
				}
				if c := b.UpdatedAt.Compare(a.UpdatedAt); c != 0 {
					return c < 0
				}
				return strings.Compare(a.SessionID, b.SessionID) < 0
			})
			if err != nil {
				return err
			}
			sortTUISessions(items)

			home, _ := config.ResolvePath("~")
			if strings.TrimSpace(groupBy) == "" {
				groupBy = strings.TrimSpace(deps.TUIConfig.GroupBy)
			}
			mode, err := normalizeTUIGroupBy(groupBy)
			if err != nil {
				return err
			}
			theme, err := resolveTUITheme(deps.TUIConfig.Theme, deps.TUIConfig.Colors, themeName, themeColors)
			if err != nil {
				return err
			}
			previewIndex, err := config.ResolvePath("~/.codex/codexsm/index/preview.v1.jsonl")
			if err != nil {
				previewIndex = ""
			}
			m := tuiModel{
				sessions:           items,
				home:               home,
				sessionsRoot:       sessionsRoot,
				status:             "Ready. Press q to quit.",
				previewCache:       make(map[string][]string),
				previewNodes:       make(map[string]*list.Element),
				previewBytesBudget: 8 << 20,
				focus:              focusTree,
				groupBy:            mode,
				source:             source,
				theme:              theme,
				previewIndex:       previewIndex,
				indexCap:           5000,
				trashRoot:          trashRoot,
				logFile:            logFile,
				dryRun:             dryRun,
				confirm:            confirm,
				yes:                yes,
				hardDelete:         hardDelete,
				maxBatch:           maxBatch,
			}
			m.rebuildTree()
			_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
			return err
		},
	}

	cmd.Flags().StringVar(&sessionsRoot, "sessions-root", "", "sessions root directory")
	cmd.Flags().StringVar(&trashRoot, "trash-root", "", "trash root directory")
	cmd.Flags().StringVar(&logFile, "log-file", "", "action log file")
	cmd.Flags().IntVarP(&limit, "limit", "l", 100, "max sessions retained for TUI ordering and rendering (0 means unlimited)")
	cmd.Flags().StringVar(&groupBy, "group-by", "", "tree group key: host|day|month")
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
		return m, m.ensurePreviewRequest()
	case previewLoadedMsg:
		if msg.RequestID != m.previewReqID || msg.Key != m.previewWait {
			return m, nil
		}
		m.previewWait = ""
		if msg.Err == "" {
			m.previewCachePut(msg.Key, msg.Lines)
			return m, persistPreviewIndexCmd(m.previewIndex, m.indexCap, msg.Record)
		}
		m.previewCachePut(msg.Key, []string{" preview load failed: " + msg.Err})
		return m, nil
	case previewIndexPersistedMsg:
		return m, nil
	case tea.KeyMsg:
		if m.handleKey(msg.String()) {
			return m, tea.Quit
		}
		return m, m.ensurePreviewRequest()
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
	case "m":
		m.requestHostMigrate()
	case "r":
		m.requestRestore()
	case "y":
		m.commitPendingAction()
	case "n", "esc":
		m.cancelPendingAction()
	}
	return false
}
