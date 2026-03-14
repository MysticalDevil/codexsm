package tui

import (
	"container/list"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MysticalDevil/codexsm/tui/preview"
	"github.com/charmbracelet/lipgloss"
)

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
	case preview.LoadedMsg:
		out := preview.HandleLoaded(m.previewReqID, m.previewWait, msg, m.previewIndex, m.indexCap)
		if !out.Accepted {
			return m, nil
		}

		m.previewWait = out.NextWait
		m.previewCachePut(out.CacheKey, out.CacheLines)

		return m, out.PersistCmd
	case preview.IndexPersistedMsg:
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
			m.previewOffset = 1 << 30
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

type previewLoadRequest struct {
	RequestID     uint64
	Key           string
	Path          string
	Width         int
	Lines         int
	Palette       preview.ThemePalette
	IndexPath     string
	SizeBytes     int64
	UpdatedAtUnix int64
}

type previewLRUEntry struct {
	Key  string
	Size int64
}

type angleTagTone = preview.AngleTagTone

const (
	angleTagToneDefault   = preview.AngleTagToneDefault
	angleTagToneSystem    = preview.AngleTagToneSystem
	angleTagToneLifecycle = preview.AngleTagToneLifecycle
	angleTagToneDanger    = preview.AngleTagToneDanger
	angleTagToneSuccess   = preview.AngleTagToneSuccess
)

func (m *tuiModel) ensurePreviewRequest() tea.Cmd {
	selected, ok := m.selectedSession()
	if !ok {
		m.previewWait = ""
		return nil
	}

	width, lines := m.currentPreviewRequestDims()

	key := preview.CacheKeyForSession(selected.Path, width, selected.SizeBytes, selected.UpdatedAt.UnixNano())
	if _, ok := m.previewCacheGet(key); ok {
		m.previewWait = ""
		return nil
	}

	if m.previewWait == key {
		return nil
	}

	m.previewReqSeq++
	m.previewReqID = m.previewReqSeq
	m.previewWait = key
	req := previewLoadRequest{
		RequestID:     m.previewReqID,
		Key:           key,
		Path:          selected.Path,
		Width:         width,
		Lines:         lines,
		Palette:       previewPalette(m.theme),
		IndexPath:     m.previewIndex,
		SizeBytes:     selected.SizeBytes,
		UpdatedAtUnix: selected.UpdatedAt.UnixNano(),
	}

	return preview.LoadCmd(preview.Request{
		RequestID:     req.RequestID,
		Key:           req.Key,
		Path:          req.Path,
		Width:         req.Width,
		Lines:         req.Lines,
		Palette:       req.Palette,
		IndexPath:     req.IndexPath,
		SizeBytes:     req.SizeBytes,
		UpdatedAtUnix: req.UpdatedAtUnix,
	})
}

func (m *tuiModel) currentPreviewRequestDims() (int, int) {
	metrics := Compute(m.width, m.height)
	rightBase := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
	rightW := max(12, metrics.RightOuterW-rightBase.GetHorizontalFrameSize())
	previewInnerH := max(2, metrics.PreviewOuterH-rightBase.GetVerticalFrameSize())
	// Keep in sync with appendSelectedSessionPreview fixed-row budget.
	previewContentHeight := max(1, previewInnerH-5)
	previewTextWidth := max(8, rightW-8)

	return previewTextWidth, previewContentHeight
}

func (m *tuiModel) previewCacheGet(key string) ([]string, bool) {
	if m.previewCache == nil {
		m.previewCache = make(map[string][]string)
	}

	if m.previewNodes == nil {
		m.previewNodes = make(map[string]*list.Element)
	}

	if m.previewLRU == nil {
		m.previewLRU = list.New()
	}

	v, ok := m.previewCache[key]
	if !ok {
		return nil, false
	}

	if n, ok := m.previewNodes[key]; ok && n != nil {
		m.previewLRU.MoveToBack(n)
	}

	return append([]string(nil), v...), true
}

func (m *tuiModel) previewCachePeek(key string) ([]string, bool) {
	if m.previewCache == nil {
		return nil, false
	}

	v, ok := m.previewCache[key]
	if !ok {
		return nil, false
	}

	return append([]string(nil), v...), true
}

func (m *tuiModel) previewCachePut(key string, lines []string) {
	if m.previewBytesBudget <= 0 {
		m.previewBytesBudget = 8 << 20
	}

	if m.previewCache == nil {
		m.previewCache = make(map[string][]string)
	}

	if m.previewNodes == nil {
		m.previewNodes = make(map[string]*list.Element)
	}

	if m.previewLRU == nil {
		m.previewLRU = list.New()
	}

	copied := append([]string(nil), lines...)
	newSize := preview.LinesBytes(copied)

	oldSize := int64(0)
	if old, ok := m.previewCache[key]; ok {
		oldSize = preview.LinesBytes(old)
	}

	m.previewCache[key] = copied
	m.previewBytesUsed += newSize - oldSize

	if n, ok := m.previewNodes[key]; ok && n != nil {
		if ent, ok := n.Value.(previewLRUEntry); ok {
			ent.Size = newSize
			n.Value = ent
		}

		m.previewLRU.MoveToBack(n)
	} else {
		m.previewNodes[key] = m.previewLRU.PushBack(previewLRUEntry{Key: key, Size: newSize})
	}

	for m.previewBytesUsed > m.previewBytesBudget && m.previewLRU.Len() > 0 {
		front := m.previewLRU.Front()
		if front == nil {
			break
		}

		ent, ok := front.Value.(previewLRUEntry)
		if !ok {
			m.previewLRU.Remove(front)
			continue
		}

		k := ent.Key

		m.previewLRU.Remove(front)
		delete(m.previewNodes, k)

		if old, ok := m.previewCache[k]; ok {
			m.previewBytesUsed -= preview.LinesBytes(old)
		}

		delete(m.previewCache, k)
	}

	if m.previewBytesUsed < 0 {
		m.previewBytesUsed = 0
	}
}

// previewFor is a synchronous preview helper used by unit tests.
func (m *tuiModel) previewFor(path string, width, lines int) []string {
	sizeBytes := int64(0)
	updatedAtUnix := int64(0)

	if info, err := os.Stat(path); err == nil {
		sizeBytes = info.Size()
		updatedAtUnix = info.ModTime().UnixNano()
	}

	key := preview.CacheKeyForSession(path, width, sizeBytes, updatedAtUnix)
	if cached, ok := m.previewCacheGet(key); ok {
		return cached
	}

	out := preview.BuildLines(path, width, lines, previewPalette(m.theme))
	m.previewCachePut(key, out)

	return out
}

func previewPalette(theme tuiTheme) preview.ThemePalette {
	def := builtinThemes[DefaultThemeName()]

	return preview.ThemePalette{
		PrefixDefault:   theme.hex("prefix_default", def["prefix_default"]),
		PrefixUser:      theme.hex("prefix_user", def["prefix_user"]),
		PrefixAssistant: theme.hex("prefix_assistant", def["prefix_assistant"]),
		PrefixOther:     theme.hex("prefix_other", def["prefix_other"]),
		TagDanger:       theme.hex("tag_danger", def["tag_danger"]),
		TagDefault:      theme.hex("tag_default", def["tag_default"]),
		TagSystem:       theme.hex("tag_system", def["tag_system"]),
		TagLifecycle:    theme.hex("tag_lifecycle", def["tag_lifecycle"]),
		TagSuccess:      theme.hex("tag_success", def["tag_success"]),
	}
}
