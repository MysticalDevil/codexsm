package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/MysticalDevil/codexsm/audit"
	"github.com/MysticalDevil/codexsm/config"
	"github.com/MysticalDevil/codexsm/session"

	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
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
		hard         bool
		interactive  bool
		previewMode  string
		previewLimit int
		maxBatch     int
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete sessions safely (dry-run by default)",
		Long: "Delete sessions matched by selectors.\n\n" +
			"By default this command runs in dry-run mode and does not modify files.\n" +
			"Use `--dry-run=false --confirm` for real deletion.",
		Example: "  codexsm delete --id <session_id>\n" +
			"  codexsm delete --id-prefix 019ca9 --dry-run=false --confirm\n" +
			"  codexsm delete --older-than 90d --dry-run=false --confirm --yes\n" +
			"  codexsm delete --host-contains /workspace/delete --head-contains fixture --dry-run=false --confirm --yes\n" +
			"  codexsm delete --id <session_id> --dry-run=false --confirm --hard",
		RunE: func(cmd *cobra.Command, args []string) error {
			lg := logger().With("command", "delete")
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

			sessions, err := session.ScanSessions(sessionsRoot)
			if err != nil {
				return err
			}
			candidates := session.FilterSessions(sessions, sel, time.Now())
			lg.Info("matched delete candidates", "count", len(candidates), "dry_run", dryRun, "hard", hard)
			if !dryRun {
				printDeletePreview(cmd, candidates, hard, mode, previewLimit)
			}
			batchID := ""
			if len(candidates) > 0 {
				batchID, err = audit.NewBatchID()
				if err != nil {
					return err
				}
			}

			if !dryRun && interactive && !yes && len(candidates) > 0 {
				ok, err := interactiveConfirmDelete(cmd, len(candidates), hard)
				if err != nil {
					return WithExitCode(err, 1)
				}
				if !ok {
					return WithExitCode(errors.New("delete aborted by user"), 1)
				}
				yes = true
			}

			opts := session.DeleteOptions{
				DryRun:       dryRun,
				Confirm:      confirm,
				Yes:          yes,
				Hard:         hard,
				MaxBatch:     maxBatch,
				TrashRoot:    trashRoot,
				SessionsRoot: sessionsRoot,
			}
			summary, deleteErr := session.DeleteSessions(candidates, sel, opts)

			rec := audit.ActionRecord{
				BatchID:       batchID,
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

			if batchID != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "batch_id=%s\n", batchID)
			}
			printDeleteSummary(cmd, summary)

			if logErr != nil {
				return WithExitCode(fmt.Errorf("delete completed but failed to write log: %w", logErr), 3)
			}
			if deleteErr != nil {
				lg.Warn("delete validation or execution returned error", "error", deleteErr)
				return WithExitCode(deleteErr, 1)
			}

			if summary.Failed > 0 {
				lg.Warn("delete completed with failures", "failed", summary.Failed, "succeeded", summary.Succeeded)
				if summary.Succeeded == 0 {
					return WithExitCode(fmt.Errorf("all operations failed: %d failed", summary.Failed), 3)
				}
				return WithExitCode(fmt.Errorf("partial failure: %d failed", summary.Failed), 2)
			}
			lg.Info("delete completed", "matched", summary.MatchedCount, "succeeded", summary.Succeeded, "simulation", summary.Simulation)
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
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", true, "simulate delete without changing files")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "required for real delete")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip interactive prompt and approve delete")
	cmd.Flags().BoolVar(&hard, "hard", false, "hard delete (permanent)")
	cmd.Flags().BoolVar(&interactive, "interactive-confirm", true, "prompt for interactive confirmation on real delete")
	cmd.Flags().StringVarP(&previewMode, "preview", "P", "sample", "preview mode before real delete: full|sample|none")
	cmd.Flags().IntVarP(&previewLimit, "preview-limit", "L", 20, "number of matched sessions shown when --preview=sample")
	cmd.Flags().IntVar(&maxBatch, "max-batch", 50, "max sessions allowed for one real delete command")

	return cmd
}

func resolveOrDefault(v string, fallback func() (string, error)) (string, error) {
	if strings.TrimSpace(v) == "" {
		return fallback()
	}
	return config.ResolvePath(v)
}

func printDeleteSummary(cmd *cobra.Command, s session.DeleteSummary) {
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

type previewMode string

const (
	previewFull   previewMode = "full"
	previewSample previewMode = "sample"
	previewNone   previewMode = "none"
)

func parsePreviewMode(v string) (previewMode, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", string(previewSample):
		return previewSample, nil
	case string(previewFull):
		return previewFull, nil
	case string(previewNone):
		return previewNone, nil
	default:
		return "", fmt.Errorf("invalid --preview %q (allowed: full, sample, none)", v)
	}
}

func printDeletePreview(cmd *cobra.Command, candidates []session.Session, hard bool, mode previewMode, previewLimit int) {
	if mode == previewNone {
		return
	}
	action := "soft-delete"
	if hard {
		action = "hard-delete"
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
	logger().Debug("delete preview generated", "matched", len(candidates), "affected_bytes", totalBytes, "preview_mode", mode, "preview_limit", sampleLimit, "hard", hard)
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "preview action=%s matched=%d affected=%s mode=%s\n", action, len(candidates), formatBytesIEC(totalBytes), mode)
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

func interactiveConfirmDelete(cmd *cobra.Command, count int, hard bool) (bool, error) {
	in := cmd.InOrStdin()
	out := cmd.ErrOrStderr()
	if !isInteractiveReader(in) {
		logger().Warn("interactive confirm requested but stdin is not terminal", "count", count)
		return false, fmt.Errorf("interactive confirm requires a terminal stdin; use --yes to continue non-interactively")
	}

	reader := bufio.NewReader(in)
	if hard {
		if _, err := fmt.Fprintf(out, "Hard delete %d session(s). Type DELETE to continue: ", count); err != nil {
			return false, err
		}
		text, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		ok := strings.TrimSpace(text) == "DELETE"
		logger().Info("hard delete interactive confirmation received", "approved", ok, "count", count)
		return ok, nil
	}

	if _, err := fmt.Fprintf(out, "Delete %d session(s) to trash? [y/N]: ", count); err != nil {
		return false, err
	}
	text, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	v := strings.ToLower(strings.TrimSpace(text))
	ok := v == "y" || v == "yes"
	logger().Info("delete interactive confirmation received", "approved", ok, "count", count, "mode", "soft")
	return ok, nil
}

func isInteractiveReader(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
