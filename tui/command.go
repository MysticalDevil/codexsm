// Package tui implements the interactive terminal UI for browsing and managing sessions.
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
	"github.com/MysticalDevil/codexsm/usecase"
)

type treeItemKind int

const (
	treeItemMonth treeItemKind = iota
	treeItemSession
)

type treeItem struct {
	Kind   treeItemKind
	Label  string
	Month  string
	Index  int
	Indent int
	// HostMissing marks sessions whose host path does not exist on local filesystem.
	HostMissing bool
}

type tuiFocus int

const (
	focusTree tuiFocus = iota
	focusPreview
)

type ultraPane int

const (
	ultraPaneTree ultraPane = iota
	ultraPanePreview
)

type tuiModel struct {
	sessions           []session.Session
	tree               []treeItem
	collapsedGroups    map[string]bool
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
	ultraPane          ultraPane
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
	maxBatchChanged    bool
	pendingAction      string
	pendingStep        int
	pendingID          string
	pendingHost        string
	pendingGroup       string
	pendingCount       int
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
		scanLimit    int
		viewLimit    int
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
			"  z: collapse/expand selected session group\n" +
			"  Z: expand all groups\n" +
			"  Ctrl+d / Ctrl+u: scroll preview\n" +
			"  d: delete current session, or selected group on a group header\n" +
			"  m: migrate sessions with missing selected host to trash\n" +
			"  r: restore current session (only when --source=trash)\n" +
			"  y/n: confirm/cancel pending action (group real delete requires 3 confirms)\n" +
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

			if scanLimit < 0 {
				return fmt.Errorf("invalid --scan-limit value %d", scanLimit)
			}

			if viewLimit < 0 {
				return fmt.Errorf("invalid --view-limit value %d", viewLimit)
			}

			scanRoot := sessionsRoot
			if source == "trash" {
				scanRoot = filepath.Join(trashRoot, "sessions")
			}

			result, err := usecase.LoadTUISessions(usecase.LoadTUISessionsInput{
				SessionsRoot: scanRoot,
				ScanLimit:    scanLimit,
				ViewLimit:    viewLimit,
			})
			if err != nil {
				return err
			}

			items := result.Items

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
				collapsedGroups:    make(map[string]bool),
				home:               home,
				sessionsRoot:       sessionsRoot,
				status:             "Ready. Press q to quit.",
				previewCache:       make(map[string][]string),
				previewNodes:       make(map[string]*list.Element),
				previewBytesBudget: 8 << 20,
				focus:              focusTree,
				ultraPane:          ultraPaneTree,
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
				maxBatchChanged:    cmd.Flags().Changed("max-batch"),
			}
			m.rebuildTree()
			_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()

			return err
		},
	}

	cmd.Flags().StringVar(&sessionsRoot, "sessions-root", "", "sessions root directory")
	cmd.Flags().StringVar(&trashRoot, "trash-root", "", "trash root directory")
	cmd.Flags().StringVar(&logFile, "log-file", "", "action log file")
	cmd.Flags().IntVar(&scanLimit, "scan-limit", 2000, "max sessions scanned then sorted for TUI (0 means unlimited)")
	cmd.Flags().IntVarP(&viewLimit, "view-limit", "l", 100, "max sessions rendered in TUI after sorting (0 means unlimited)")
	cmd.Flags().StringVar(&groupBy, "group-by", "", "tree group key: host|day|month")
	cmd.Flags().StringVar(&source, "source", "", "session source: sessions|trash")
	cmd.Flags().StringVar(&themeName, "theme", "", "TUI theme: tokyonight|catppuccin|gruvbox|onedark|nord|dracula")
	cmd.Flags().StringArrayVar(&themeColors, "theme-color", nil, "custom theme override (key=value), repeatable")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", true, "simulate delete/restore from TUI")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "required for real delete/restore from TUI")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip TUI confirmation prompts")
	cmd.Flags().BoolVar(&hardDelete, "hard", false, "hard delete on session source")
	cmd.Flags().IntVar(&maxBatch, "max-batch", 100, "max sessions allowed for one real TUI action")

	return cmd
}
