package session

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MysticalDevil/codexsm/util"
)

// RestoreOptions controls restore behavior.
type RestoreOptions struct {
	DryRun             bool
	Confirm            bool
	Yes                bool
	AllowEmptySelector bool
	MaxBatch           int
	SessionsRoot       string
	TrashSessionsRoot  string
}

// RestoreSummary describes restore operation result.
type RestoreSummary struct {
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

// RestoreSessions runs restore over selected candidates.
func RestoreSessions(candidates []Session, sel Selector, opts RestoreOptions) (RestoreSummary, error) {
	summary := RestoreSummary{
		Action:       restoreActionName(opts.DryRun),
		Simulation:   opts.DryRun,
		MatchedCount: len(candidates),
		Results:      make([]DeleteResult, 0, len(candidates)),
	}
	if !sel.HasAnyFilter() && !opts.AllowEmptySelector {
		summary.ErrorSummary = "restore requires at least one selector (--id/--id-prefix/--host-contains/--path-contains/--head-contains/--older-than/--health or --batch-id)"
		return summary, errors.New(summary.ErrorSummary)
	}
	if opts.MaxBatch <= 0 {
		opts.MaxBatch = 50
	}
	if !opts.DryRun {
		if !opts.Confirm {
			summary.ErrorSummary = "real restore requires --confirm"
			return summary, errors.New(summary.ErrorSummary)
		}
		if len(candidates) > 1 && !opts.Yes {
			summary.ErrorSummary = "batch restore requires --yes"
			return summary, errors.New(summary.ErrorSummary)
		}
		if len(candidates) > opts.MaxBatch {
			summary.ErrorSummary = fmt.Sprintf("matched %d sessions; exceeds --max-batch=%d", len(candidates), opts.MaxBatch)
			return summary, errors.New(summary.ErrorSummary)
		}
	}

	for _, s := range candidates {
		summary.AffectedBytes += s.SizeBytes
		rel, err := filepath.Rel(opts.TrashSessionsRoot, s.Path)
		if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
			rel = filepath.Base(s.Path)
		}
		dst := filepath.Join(opts.SessionsRoot, rel)

		if opts.DryRun {
			summary.Skipped++
			summary.Results = append(summary.Results, DeleteResult{
				SessionID:   s.SessionID,
				Path:        s.Path,
				Destination: dst,
				Status:      "simulated",
			})
			continue
		}

		if _, err := os.Stat(dst); err == nil {
			summary.Failed++
			summary.Results = append(summary.Results, DeleteResult{
				SessionID: s.SessionID,
				Path:      s.Path,
				Status:    "failed",
				Error:     fmt.Sprintf("destination exists: %s", dst),
			})
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			summary.Failed++
			summary.Results = append(summary.Results, DeleteResult{
				SessionID: s.SessionID,
				Path:      s.Path,
				Status:    "failed",
				Error:     err.Error(),
			})
			continue
		}
		if err := util.MoveFile(s.Path, dst); err != nil {
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
		summary.Results = append(summary.Results, DeleteResult{
			SessionID:   s.SessionID,
			Path:        s.Path,
			Destination: dst,
			Status:      "restored",
		})
	}

	if summary.Failed > 0 {
		summary.ErrorSummary = fmt.Sprintf("%d failed", summary.Failed)
	}
	return summary, nil
}

func restoreActionName(dryRun bool) string {
	if dryRun {
		return "restore-dry-run"
	}
	return "restore"
}
