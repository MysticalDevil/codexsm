package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MysticalDevil/codexsm/audit"
	"github.com/MysticalDevil/codexsm/session"

	"github.com/spf13/cobra"
)

type restoreSummary struct {
	Action        string
	Simulation    bool
	MatchedCount  int
	Succeeded     int
	Failed        int
	Skipped       int
	AffectedBytes int64
	Results       []session.DeleteResult
	ErrorSummary  string
}

func newRestoreCmd() *cobra.Command {
	var (
		sessionsRoot string
		trashRoot    string
		logFile      string
		id           string
		idPrefix     string
		hostContains string
		pathContains string
		headContains string
		olderThan    string
		health       string
		dryRun       bool
		confirm      bool
		yes          bool
		interactive  bool
		previewMode  string
		previewLimit int
		maxBatch     int
	)

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore sessions from trash (dry-run by default)",
		Long: "Restore sessions that were previously soft-deleted to trash.\n\n" +
			"By default this command runs in dry-run mode and does not modify files.\n" +
			"Use `--dry-run=false --confirm` for real restore.",
		Example: "  codexsm restore --id <session_id>\n" +
			"  codexsm restore --id-prefix 019ca9 --dry-run=false --confirm\n" +
			"  codexsm restore --path-contains /trash/sessions/2026/03/02 --head-contains fixture --dry-run=false --confirm --yes\n" +
			"  codexsm restore --older-than 30d --dry-run=false --confirm --yes",
		RunE: func(cmd *cobra.Command, args []string) error {
			lg := logger().With("command", "restore")
			var err error
			sessionsRoot, err = resolveOrDefault(sessionsRoot, runtimeSessionsRoot)
			if err != nil {
				return err
			}
			trashRoot, err = resolveOrDefault(trashRoot, runtimeTrashRoot)
			if err != nil {
				return err
			}
			logFile, err = resolveOrDefault(logFile, runtimeLogFile)
			if err != nil {
				return err
			}

			sel, err := buildSelector(id, idPrefix, hostContains, pathContains, headContains, olderThan, health)
			if err != nil {
				return err
			}
			mode, err := parsePreviewMode(previewMode)
			if err != nil {
				return err
			}
			if !sel.HasAnyFilter() {
				return WithExitCode(errors.New("restore requires at least one selector (--id/--id-prefix/--host-contains/--path-contains/--head-contains/--older-than/--health)"), 1)
			}

			trashSessionsRoot := filepath.Join(trashRoot, "sessions")
			sessions, err := session.ScanSessions(trashSessionsRoot)
			if err != nil {
				return err
			}
			candidates := session.FilterSessions(sessions, sel, time.Now())
			lg.Info("matched restore candidates", "count", len(candidates), "dry_run", dryRun)
			if !dryRun {
				printRestorePreview(cmd, candidates, mode, previewLimit)
			}

			if !dryRun && interactive && !yes && len(candidates) > 0 {
				ok, err := interactiveConfirmRestore(cmd, len(candidates))
				if err != nil {
					return WithExitCode(err, 1)
				}
				if !ok {
					return WithExitCode(errors.New("restore aborted by user"), 1)
				}
				yes = true
			}

			summary, runErr := restoreSessions(candidates, sel, restoreOptions{
				DryRun:            dryRun,
				Confirm:           confirm,
				Yes:               yes,
				MaxBatch:          maxBatch,
				SessionsRoot:      sessionsRoot,
				TrashSessionsRoot: trashSessionsRoot,
			})

			rec := audit.ActionRecord{
				Timestamp:     time.Now().UTC(),
				Action:        summary.Action,
				Simulation:    summary.Simulation,
				Selector:      sel,
				MatchedCount:  summary.MatchedCount,
				AffectedBytes: summary.AffectedBytes,
				Results:       summary.Results,
				ErrorSummary:  summary.ErrorSummary,
			}
			rec.Sessions = make([]audit.SessionRef, 0, len(candidates))
			for _, s := range candidates {
				rec.Sessions = append(rec.Sessions, audit.SessionRef{SessionID: s.SessionID, Path: s.Path})
			}
			logErr := audit.WriteActionLog(logFile, rec)
			if logErr != nil {
				lg.Error("failed to write action log", "error", logErr, "log_file", logFile)
			}

			printRestoreSummary(cmd, summary)

			if logErr != nil {
				return WithExitCode(fmt.Errorf("restore completed but failed to write log: %w", logErr), 3)
			}
			if runErr != nil {
				lg.Warn("restore validation or execution returned error", "error", runErr)
				return WithExitCode(runErr, 1)
			}
			if summary.Failed > 0 {
				lg.Warn("restore completed with failures", "failed", summary.Failed, "succeeded", summary.Succeeded)
				if summary.Succeeded == 0 {
					return WithExitCode(fmt.Errorf("all operations failed: %d failed", summary.Failed), 3)
				}
				return WithExitCode(fmt.Errorf("partial failure: %d failed", summary.Failed), 2)
			}
			lg.Info("restore completed", "matched", summary.MatchedCount, "succeeded", summary.Succeeded, "simulation", summary.Simulation)
			return nil
		},
	}

	cmd.Flags().StringVar(&sessionsRoot, "sessions-root", "", "sessions root directory")
	cmd.Flags().StringVar(&trashRoot, "trash-root", "", "trash root directory")
	cmd.Flags().StringVar(&logFile, "log-file", "", "action log file (jsonl)")
	cmd.Flags().StringVarP(&id, "id", "i", "", "exact session id")
	cmd.Flags().StringVarP(&idPrefix, "id-prefix", "p", "", "session id prefix")
	cmd.Flags().StringVar(&hostContains, "host-contains", "", "case-insensitive substring match against host path")
	cmd.Flags().StringVar(&pathContains, "path-contains", "", "case-insensitive substring match against session file path")
	cmd.Flags().StringVar(&headContains, "head-contains", "", "case-insensitive substring match against preview head text")
	cmd.Flags().StringVarP(&olderThan, "older-than", "o", "", "select sessions older than duration (e.g. 30d, 12h)")
	cmd.Flags().StringVarP(&health, "health", "H", "", "health filter: ok|corrupted|missing-meta")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", true, "simulate restore without changing files")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "required for real restore")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip interactive prompt and approve restore")
	cmd.Flags().BoolVar(&interactive, "interactive-confirm", true, "prompt for interactive confirmation on real restore")
	cmd.Flags().StringVarP(&previewMode, "preview", "P", "sample", "preview mode before real restore: full|sample|none")
	cmd.Flags().IntVarP(&previewLimit, "preview-limit", "L", 20, "number of matched sessions shown when --preview=sample")
	cmd.Flags().IntVar(&maxBatch, "max-batch", 50, "max sessions allowed for one real restore command")

	return cmd
}

type restoreOptions struct {
	DryRun            bool
	Confirm           bool
	Yes               bool
	MaxBatch          int
	SessionsRoot      string
	TrashSessionsRoot string
}

func restoreSessions(candidates []session.Session, sel session.Selector, opts restoreOptions) (restoreSummary, error) {
	summary := restoreSummary{
		Action:       restoreActionName(opts.DryRun),
		Simulation:   opts.DryRun,
		MatchedCount: len(candidates),
		Results:      make([]session.DeleteResult, 0, len(candidates)),
	}
	if !sel.HasAnyFilter() {
		summary.ErrorSummary = "restore requires at least one selector (--id/--id-prefix/--host-contains/--path-contains/--head-contains/--older-than/--health)"
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
			summary.Results = append(summary.Results, session.DeleteResult{
				SessionID:   s.SessionID,
				Path:        s.Path,
				Destination: dst,
				Status:      "simulated",
			})
			continue
		}

		if _, err := os.Stat(dst); err == nil {
			summary.Failed++
			summary.Results = append(summary.Results, session.DeleteResult{
				SessionID: s.SessionID,
				Path:      s.Path,
				Status:    "failed",
				Error:     fmt.Sprintf("destination exists: %s", dst),
			})
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			summary.Failed++
			summary.Results = append(summary.Results, session.DeleteResult{SessionID: s.SessionID, Path: s.Path, Status: "failed", Error: err.Error()})
			continue
		}
		if err := moveFile(s.Path, dst); err != nil {
			summary.Failed++
			summary.Results = append(summary.Results, session.DeleteResult{SessionID: s.SessionID, Path: s.Path, Status: "failed", Error: err.Error()})
			continue
		}
		summary.Succeeded++
		summary.Results = append(summary.Results, session.DeleteResult{SessionID: s.SessionID, Path: s.Path, Destination: dst, Status: "restored"})
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

func printRestoreSummary(cmd *cobra.Command, s restoreSummary) {
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "action=%s simulation=%t matched=%d succeeded=%d failed=%d skipped=%d affected_bytes=%d\n",
		s.Action, s.Simulation, s.MatchedCount, s.Succeeded, s.Failed, s.Skipped, s.AffectedBytes)
	for _, r := range s.Results {
		if r.Error == "" {
			_, _ = fmt.Fprintf(out, "%s %s %s\n", r.Status, r.SessionID, r.Path)
			continue
		}
		_, _ = fmt.Fprintf(out, "%s %s %s err=%s\n", r.Status, r.SessionID, r.Path, r.Error)
	}
}

func printRestorePreview(cmd *cobra.Command, candidates []session.Session, mode previewMode, previewLimit int) {
	if mode == previewNone {
		return
	}
	var totalBytes int64
	for _, s := range candidates {
		totalBytes += s.SizeBytes
	}
	sampleLimit := previewLimit
	if mode == previewFull {
		sampleLimit = len(candidates)
	}
	if sampleLimit < 0 {
		sampleLimit = 0
	}
	logger().Debug("restore preview generated", "matched", len(candidates), "affected_bytes", totalBytes, "preview_mode", mode, "preview_limit", sampleLimit)
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "preview action=restore matched=%d affected=%s mode=%s\n", len(candidates), formatBytesIEC(totalBytes), mode)
	for i, s := range candidates {
		if i >= sampleLimit {
			break
		}
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  - %s %s\n", shortID(s.SessionID), s.Path)
	}
	if mode == previewSample && len(candidates) > sampleLimit {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  ... and %d more\n", len(candidates)-sampleLimit)
	}
}

func interactiveConfirmRestore(cmd *cobra.Command, count int) (bool, error) {
	in := cmd.InOrStdin()
	out := cmd.ErrOrStderr()
	if !isInteractiveReader(in) {
		logger().Warn("interactive restore requested but stdin is not terminal", "count", count)
		return false, fmt.Errorf("interactive confirm requires a terminal stdin; use --yes to continue non-interactively")
	}
	if _, err := fmt.Fprintf(out, "Restore %d session(s) from trash? [y/N]: ", count); err != nil {
		return false, err
	}
	reader := bufio.NewReader(in)
	text, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	v := strings.ToLower(strings.TrimSpace(text))
	ok := v == "y" || v == "yes"
	logger().Info("restore interactive confirmation received", "approved", ok, "count", count)
	return ok, nil
}

func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := copyFileForRestore(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}

func copyFileForRestore(src, dst string) (retErr error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := in.Close(); closeErr != nil {
			if retErr == nil {
				retErr = closeErr
			} else {
				retErr = errors.Join(retErr, closeErr)
			}
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
			} else {
				retErr = errors.Join(retErr, closeErr)
			}
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
