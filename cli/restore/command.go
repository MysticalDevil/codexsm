// Package restore provides the `codexsm restore` command.
package restore

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	cliutil "github.com/MysticalDevil/codexsm/cli/util"
	"github.com/MysticalDevil/codexsm/internal/core"
	"github.com/MysticalDevil/codexsm/internal/ops"
	"github.com/MysticalDevil/codexsm/session"
	"github.com/MysticalDevil/codexsm/usecase"
	"github.com/spf13/cobra"
)

// NewCommand builds the restore command.
func NewCommand(
	resolveSessionsRoot func() (string, error),
	resolveTrashRoot func() (string, error),
	resolveLogFile func() (string, error),
	runtimeAuditSink usecase.AuditSink,
	nowFn func() time.Time,
) *cobra.Command {
	var (
		sessionsRoot string
		trashRoot    string
		logFile      string
		id           string
		idPrefix     string
		hostContains string
		pathContains string
		headContains string
		batchID      string
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
			"  codexsm restore --batch-id <batch_id> --dry-run=false --confirm --yes\n" +
			"  codexsm restore --path-contains /trash/sessions/2026/03/02 --head-contains fixture --dry-run=false --confirm --yes\n" +
			"  codexsm restore --older-than 30d --dry-run=false --confirm --yes",
		RunE: func(cmd *cobra.Command, args []string) error {
			lg := slog.Default().With("command", "restore")

			var err error

			sessionsRoot, err = cliutil.ResolveOrDefault(sessionsRoot, resolveSessionsRoot)
			if err != nil {
				return err
			}

			trashRoot, err = cliutil.ResolveOrDefault(trashRoot, resolveTrashRoot)
			if err != nil {
				return err
			}

			logFile, err = cliutil.ResolveOrDefault(logFile, resolveLogFile)
			if err != nil {
				return err
			}

			sel, err := cliutil.BuildSelector(id, idPrefix, hostContains, pathContains, headContains, olderThan, health)
			if err != nil {
				return err
			}

			mode, err := ops.ParsePreviewMode(previewMode)
			if err != nil {
				return err
			}

			now := nowFn()
			trashSessionsRoot := filepath.Join(trashRoot, "sessions")
			batchID = strings.TrimSpace(batchID)

			selected, err := usecase.SelectRestoreSessions(usecase.RestoreSelectInput{
				TrashSessionsRoot: trashSessionsRoot,
				Selector:          sel,
				BatchID:           batchID,
				LogFile:           logFile,
				Now:               now,
			})
			if err != nil {
				return cliutil.WithExitCode(err, 1)
			}

			candidates := selected.Sessions
			lg.Info("matched restore candidates", "count", len(candidates), "dry_run", dryRun)

			if !dryRun {
				PrintRestorePreview(cmd, candidates, mode, previewLimit)
			}

			if !dryRun && interactive && !yes && len(candidates) > 0 {
				ok, err := InteractiveConfirmRestore(cmd, len(candidates))
				if err != nil {
					return cliutil.WithExitCode(err, 1)
				}

				if !ok {
					return cliutil.WithExitCode(errors.New("restore aborted by user"), 1)
				}

				yes = true
			}

			out, runErr := usecase.RunRestoreAction(usecase.RestoreActionInput{
				Sessions:           candidates,
				Selector:           sel,
				DryRun:             dryRun,
				Confirm:            confirm,
				Yes:                yes,
				AllowEmptySelector: batchID != "",
				MaxBatch:           maxBatch,
				MaxBatchChanged:    cmd.Flags().Changed("max-batch"),
				RealDefault:        usecase.DefaultMaxBatchReal,
				DryRunDefault:      usecase.DefaultMaxBatchDryRun,
				SessionsRoot:       sessionsRoot,
				TrashSessionsRoot:  trashSessionsRoot,
				LogFile:            logFile,
				AuditSink:          runtimeAuditSink,
				Now:                now,
			})
			if runErr == nil && out.LogError != nil {
				runErr = out.LogError
			}

			summary := out.Summary
			if out.LogError != nil {
				lg.Error("failed to write action log", "error", out.LogError, "log_file", logFile)
			}

			if batchID != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "rollback_from_batch_id=%s\n", batchID)
			}

			if out.BatchID != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "batch_id=%s\n", out.BatchID)
			}

			PrintRestoreSummary(cmd, summary)

			if out.LogError != nil {
				return cliutil.WithExitCode(fmt.Errorf("restore completed but failed to write log: %w", out.LogError), 3)
			}

			if runErr != nil {
				lg.Warn("restore validation or execution returned error", "error", runErr)
				return cliutil.WithExitCode(runErr, 1)
			}

			if summary.Failed > 0 {
				lg.Warn("restore completed with failures", "failed", summary.Failed, "succeeded", summary.Succeeded)

				if summary.Succeeded == 0 {
					return cliutil.WithExitCode(fmt.Errorf("all operations failed: %d failed", summary.Failed), 3)
				}

				return cliutil.WithExitCode(fmt.Errorf("partial failure: %d failed", summary.Failed), 2)
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
	cmd.Flags().StringVarP(&batchID, "batch-id", "B", "", "restore all sessions from a soft-delete batch id")
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

func PrintRestoreSummary(cmd *cobra.Command, s usecase.RestoreSummary) {
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

func PrintRestorePreview(cmd *cobra.Command, candidates []session.Session, mode ops.PreviewMode, previewLimit int) {
	if mode == ops.PreviewNone {
		return
	}

	var totalBytes int64
	for _, s := range candidates {
		totalBytes += s.SizeBytes
	}

	sampleLimit := previewLimit
	if mode == ops.PreviewFull {
		sampleLimit = len(candidates)
	}

	if sampleLimit < 0 {
		sampleLimit = 0
	}

	slog.Default().Debug("restore preview generated", "matched", len(candidates), "affected_bytes", totalBytes, "preview_mode", mode, "preview_limit", sampleLimit)

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "preview action=restore matched=%d affected=%s mode=%s\n", len(candidates), core.FormatBytesIEC(totalBytes), mode)
	for i, s := range candidates {
		if i >= sampleLimit {
			break
		}

		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  - %s %s\n", core.ShortID(s.SessionID), s.Path)
	}

	if mode == ops.PreviewSample && len(candidates) > sampleLimit {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  ... and %d more\n", len(candidates)-sampleLimit)
	}
}

func InteractiveConfirmRestore(cmd *cobra.Command, count int) (bool, error) {
	in := cmd.InOrStdin()
	if !ops.IsInteractiveReader(in) {
		slog.Default().Warn("interactive restore requested but stdin is not terminal", "count", count)
		return false, fmt.Errorf("interactive confirm requires a terminal stdin; use --yes to continue non-interactively")
	}

	ok, err := ops.ConfirmRestore(in, cmd.ErrOrStderr(), count)
	if err != nil {
		return false, err
	}

	slog.Default().Info("restore interactive confirmation received", "approved", ok, "count", count)

	return ok, nil
}
