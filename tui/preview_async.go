package tui

import (
	"container/list"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MysticalDevil/codexsm/internal/tui/layout"
	"github.com/MysticalDevil/codexsm/session"
	"github.com/charmbracelet/lipgloss"
)

type previewIndexRecord struct {
	Key           string   `json:"key"`
	Path          string   `json:"path"`
	Width         int      `json:"width"`
	SizeBytes     int64    `json:"size_bytes"`
	UpdatedAtUnix int64    `json:"updated_at_unix"`
	TouchedAtUnix int64    `json:"touched_at_unix"`
	Lines         []string `json:"lines"`
}

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

func previewCacheKeyForSession(s session.Session, width int) string {
	return fmt.Sprintf("%s|w:%d|sz:%d|mt:%d", s.Path, width, s.SizeBytes, s.UpdatedAt.UnixNano())
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
	metrics := layout.Compute(m.width, m.height)
	rightBase := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
	rightW := max(12, metrics.RightOuterW-rightBase.GetHorizontalFrameSize())
	previewInnerH := max(2, metrics.PreviewOuterH-rightBase.GetVerticalFrameSize())
	previewContentHeight := max(2, previewInnerH-4)
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
	if m.previewCap <= 0 {
		m.previewCap = 256
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

	m.previewCache[key] = append([]string(nil), lines...)
	if n, ok := m.previewNodes[key]; ok && n != nil {
		m.previewLRU.MoveToBack(n)
	} else {
		m.previewNodes[key] = m.previewLRU.PushBack(key)
	}
	for len(m.previewCache) > m.previewCap {
		front := m.previewLRU.Front()
		if front == nil {
			break
		}
		k, _ := front.Value.(string)
		m.previewLRU.Remove(front)
		delete(m.previewNodes, k)
		delete(m.previewCache, k)
	}
}

func loadPreviewCmd(req previewLoadRequest) tea.Cmd {
	return func() tea.Msg {
		if req.Path == "" {
			return previewLoadedMsg{RequestID: req.RequestID, Key: req.Key, Err: "empty path"}
		}

		if req.IndexPath != "" {
			if lines, ok, err := loadPreviewIndexEntry(req.IndexPath, req.Key); err == nil && ok {
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
		if err := upsertPreviewIndex(indexPath, cap, record); err != nil {
			return previewIndexPersistedMsg{Err: err.Error()}
		}
		return previewIndexPersistedMsg{}
	}
}
