package preview

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type HandleLoadedOutput struct {
	Accepted   bool
	NextWait   string
	CacheKey   string
	CacheLines []string
	PersistCmd tea.Cmd
}

// HandleLoaded validates stale async messages and returns cache/persist actions.
func HandleLoaded(currentReqID uint64, waitKey string, msg LoadedMsg, indexPath string, cap int) HandleLoadedOutput {
	if msg.RequestID != currentReqID || msg.Key != waitKey {
		return HandleLoadedOutput{Accepted: false, NextWait: waitKey}
	}

	out := HandleLoadedOutput{
		Accepted:   true,
		NextWait:   "",
		CacheKey:   msg.Key,
		CacheLines: msg.Lines,
	}
	if msg.Err != "" {
		out.CacheLines = []string{" preview load failed: " + msg.Err}
		return out
	}

	out.PersistCmd = PersistIndexCmd(indexPath, cap, msg.Record)

	return out
}

func LoadCmd(req Request) tea.Cmd {
	return func() tea.Msg {
		if req.Path == "" {
			return LoadedMsg{RequestID: req.RequestID, Key: req.Key, Err: "empty path"}
		}

		if req.IndexPath != "" {
			if lines, ok, err := LoadIndexEntry(req.IndexPath, req.Key); err == nil && ok {
				return LoadedMsg{
					RequestID: req.RequestID,
					Key:       req.Key,
					Lines:     lines,
					Record: IndexRecord{
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

		lines := BuildLines(req.Path, req.Width, req.Lines, req.Palette)

		return LoadedMsg{
			RequestID: req.RequestID,
			Key:       req.Key,
			Lines:     lines,
			Record: IndexRecord{
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

func PersistIndexCmd(indexPath string, cap int, record IndexRecord) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(indexPath) == "" || strings.TrimSpace(record.Key) == "" {
			return IndexPersistedMsg{}
		}

		if err := UpsertIndex(indexPath, cap, record); err != nil {
			return IndexPersistedMsg{Err: err.Error()}
		}

		return IndexPersistedMsg{}
	}
}
