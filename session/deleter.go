package session

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DeleteOptions controls delete mode and safety gates.
type DeleteOptions struct {
	DryRun       bool
	Confirm      bool
	Yes          bool
	Hard         bool
	MaxBatch     int
	TrashRoot    string
	SessionsRoot string
}

// DeleteResult is a per-session execution result entry.
type DeleteResult struct {
	SessionID   string `json:"session_id"`
	Path        string `json:"path"`
	Destination string `json:"destination,omitempty"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}

// DeleteSummary aggregates delete execution status and accounting.
type DeleteSummary struct {
	Action        string         `json:"action"`
	Simulation    bool           `json:"simulation"`
	MatchedCount  int            `json:"matched_count"`
	Succeeded     int            `json:"succeeded"`
	Failed        int            `json:"failed"`
	Skipped       int            `json:"skipped"`
	AffectedBytes int64          `json:"affected_bytes"`
	Results       []DeleteResult `json:"results"`
	ErrorSummary  string         `json:"error_summary,omitempty"`
}

// DeleteSessions executes dry-run, soft-delete, or hard-delete over matched sessions.
func DeleteSessions(candidates []Session, sel Selector, opts DeleteOptions) (DeleteSummary, error) {
	summary := DeleteSummary{
		Action:       actionName(opts),
		Simulation:   opts.DryRun,
		MatchedCount: len(candidates),
		Results:      make([]DeleteResult, 0, len(candidates)),
	}

	if !sel.HasAnyFilter() {
		summary.ErrorSummary = "delete requires at least one selector (--id/--id-prefix/--older-than/--health)"
		return summary, errors.New(summary.ErrorSummary)
	}
	if opts.MaxBatch <= 0 {
		opts.MaxBatch = 50
	}
	if !opts.DryRun {
		if !opts.Confirm {
			summary.ErrorSummary = "real delete requires --confirm"
			return summary, errors.New(summary.ErrorSummary)
		}
		if len(candidates) > 1 && !opts.Yes {
			summary.ErrorSummary = "batch delete requires --yes"
			return summary, errors.New(summary.ErrorSummary)
		}
		if len(candidates) > opts.MaxBatch {
			summary.ErrorSummary = fmt.Sprintf("matched %d sessions; exceeds --max-batch=%d", len(candidates), opts.MaxBatch)
			return summary, errors.New(summary.ErrorSummary)
		}
	}

	for _, s := range candidates {
		summary.AffectedBytes += s.SizeBytes
		if opts.DryRun {
			summary.Skipped++
			summary.Results = append(summary.Results, DeleteResult{
				SessionID: s.SessionID,
				Path:      s.Path,
				Status:    "simulated",
			})
			continue
		}

		if opts.Hard {
			if err := os.Remove(s.Path); err != nil {
				summary.Failed++
				summary.Results = append(summary.Results, DeleteResult{
					SessionID: s.SessionID,
					Path:      s.Path,
					Status:    "failed",
					Error:     err.Error(),
				})
				continue
			}
			summary.Succeeded++
			summary.Results = append(summary.Results, DeleteResult{SessionID: s.SessionID, Path: s.Path, Status: "deleted"})
			continue
		}

		dst, err := moveToTrash(s.Path, opts.SessionsRoot, opts.TrashRoot)
		if err != nil {
			summary.Failed++
			summary.Results = append(summary.Results, DeleteResult{
				SessionID: s.SessionID,
				Path:      s.Path,
				Status:    "failed",
				Error:     err.Error(),
			})
			continue
		}
		summary.Succeeded++
		summary.Results = append(summary.Results, DeleteResult{SessionID: s.SessionID, Path: s.Path, Destination: dst, Status: "deleted"})
	}

	if summary.Failed > 0 {
		summary.ErrorSummary = fmt.Sprintf("%d failed", summary.Failed)
	}
	return summary, nil
}

func actionName(opts DeleteOptions) string {
	if opts.DryRun {
		return "dry-run"
	}
	if opts.Hard {
		return "hard-delete"
	}
	return "soft-delete"
}

func moveToTrash(src, sessionsRoot, trashRoot string) (string, error) {
	rel, err := filepath.Rel(sessionsRoot, src)
	if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
		rel = filepath.Base(src)
	}
	dst := filepath.Join(trashRoot, "sessions", rel)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	if _, err := os.Stat(dst); err == nil {
		dst = filepath.Join(filepath.Dir(dst), fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(dst)))
	}

	if err := os.Rename(src, dst); err == nil {
		return dst, nil
	}

	if err := copyFile(src, dst); err != nil {
		return "", err
	}
	if err := os.Remove(src); err != nil {
		return "", err
	}
	return dst, nil
}

func copyFile(src, dst string) (retErr error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := in.Close(); closeErr != nil {
			if retErr == nil {
				retErr = closeErr
				return
			}
			retErr = errors.Join(retErr, closeErr)
		}
	}()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := out.Close(); closeErr != nil {
			if retErr == nil {
				retErr = closeErr
				return
			}
			retErr = errors.Join(retErr, closeErr)
		}
	}()

	if _, err := io.Copy(out, in); err != nil {
		retErr = err
		return
	}
	if err := out.Sync(); err != nil {
		retErr = err
		return
	}
	return
}
