package tui

import (
	"container/list"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MysticalDevil/codexsm/session"
	"github.com/MysticalDevil/codexsm/tui/preview"
	"github.com/charmbracelet/lipgloss"
)

type previewIndexRecord = preview.IndexRecord

type previewLoadRequest struct {
	RequestID     uint64
	Key           string
	Path          string
	Width         int
	Lines         int
	Theme         tuiTheme
	IndexPath     string
	SizeBytes     int64
	UpdatedAtUnix int64
}

type previewLoadedMsg struct {
	RequestID uint64
	Key       string
	Lines     []string
	Err       string
	Record    previewIndexRecord
}

type previewIndexPersistedMsg struct {
	Err string
}

type previewLRUEntry struct {
	Key  string
	Size int64
}

func previewCacheKeyForSession(s session.Session, width int) string {
	return preview.CacheKeyForSession(s.Path, width, s.SizeBytes, s.UpdatedAt.UnixNano())
}

func (m *tuiModel) ensurePreviewRequest() tea.Cmd {
	selected, ok := m.selectedSession()
	if !ok {
		m.previewWait = ""
		return nil
	}
	width, lines := m.currentPreviewRequestDims()
	key := previewCacheKeyForSession(selected, width)
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
		Theme:         m.theme,
		IndexPath:     m.previewIndex,
		SizeBytes:     selected.SizeBytes,
		UpdatedAtUnix: selected.UpdatedAt.UnixNano(),
	}
	return loadPreviewCmd(req)
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
	newSize := previewLinesBytes(copied)
	oldSize := int64(0)
	if old, ok := m.previewCache[key]; ok {
		oldSize = previewLinesBytes(old)
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
			m.previewBytesUsed -= previewLinesBytes(old)
		}
		delete(m.previewCache, k)
	}
	if m.previewBytesUsed < 0 {
		m.previewBytesUsed = 0
	}
}

func loadPreviewCmd(req previewLoadRequest) tea.Cmd {
	return func() tea.Msg {
		if req.Path == "" {
			return previewLoadedMsg{RequestID: req.RequestID, Key: req.Key, Err: "empty path"}
		}

		if req.IndexPath != "" {
			if lines, ok, err := preview.LoadIndexEntry(req.IndexPath, req.Key); err == nil && ok {
				return previewLoadedMsg{
					RequestID: req.RequestID,
					Key:       req.Key,
					Lines:     lines,
					Record: previewIndexRecord{
						Key:           req.Key,
						Path:          req.Path,
						Width:         req.Width,
						SizeBytes:     req.SizeBytes,
						UpdatedAtUnix: req.UpdatedAtUnix,
						TouchedAtUnix: time.Now().UnixNano(),
						Lines:         lines,
					},
				}
			}
		}

		lines := buildPreviewLines(req.Path, req.Width, req.Lines, req.Theme)
		return previewLoadedMsg{
			RequestID: req.RequestID,
			Key:       req.Key,
			Lines:     lines,
			Record: previewIndexRecord{
				Key:           req.Key,
				Path:          req.Path,
				Width:         req.Width,
				SizeBytes:     req.SizeBytes,
				UpdatedAtUnix: req.UpdatedAtUnix,
				TouchedAtUnix: time.Now().UnixNano(),
				Lines:         lines,
			},
		}
	}
}

func persistPreviewIndexCmd(indexPath string, cap int, record previewIndexRecord) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(indexPath) == "" || strings.TrimSpace(record.Key) == "" {
			return previewIndexPersistedMsg{}
		}
		if err := preview.UpsertIndex(indexPath, cap, record); err != nil {
			return previewIndexPersistedMsg{Err: err.Error()}
		}
		return previewIndexPersistedMsg{}
	}
}

func previewLinesBytes(lines []string) int64 {
	return preview.LinesBytes(lines)
}
