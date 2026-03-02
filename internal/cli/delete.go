package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/MysticalDevil/codex-sm/internal/audit"
	"github.com/MysticalDevil/codex-sm/internal/config"
	"github.com/MysticalDevil/codex-sm/internal/session"

	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	var (
		sessionsRoot string
		trashRoot    string
		logFile      string
		id           string
		idPrefix     string
		olderThan    string
		health       string
		dryRun       bool
		confirm      bool
		yes          bool
		hard         bool
		interactive  bool
		maxBatch     int
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete sessions safely (dry-run by default)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			sessionsRoot, err = resolveOrDefault(sessionsRoot, config.DefaultSessionsRoot)
			if err != nil {
				return err
			}
			trashRoot, err = resolveOrDefault(trashRoot, config.DefaultTrashRoot)
			if err != nil {
				return err
			}
			logFile, err = resolveOrDefault(logFile, config.DefaultLogFile)
			if err != nil {
				return err
			}

			sel, err := buildSelector(id, idPrefix, olderThan, health)
			if err != nil {
				return err
			}

			sessions, err := session.ScanSessions(sessionsRoot)
			if err != nil {
				return err
			}
			candidates := session.FilterSessions(sessions, sel, time.Now())

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

			printDeleteSummary(cmd, summary)

			if logErr != nil {
				return WithExitCode(fmt.Errorf("delete completed but failed to write log: %w", logErr), 3)
			}
			if deleteErr != nil {
				return WithExitCode(deleteErr, 1)
			}

			if summary.Failed > 0 {
				if summary.Succeeded == 0 {
					return WithExitCode(fmt.Errorf("all operations failed: %d failed", summary.Failed), 3)
				}
				return WithExitCode(fmt.Errorf("partial failure: %d failed", summary.Failed), 2)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&sessionsRoot, "sessions-root", "", "sessions root directory")
	cmd.Flags().StringVar(&trashRoot, "trash-root", "", "trash root directory")
	cmd.Flags().StringVar(&logFile, "log-file", "", "action log file (jsonl)")
	cmd.Flags().StringVar(&id, "id", "", "exact session id")
	cmd.Flags().StringVar(&idPrefix, "id-prefix", "", "session id prefix")
	cmd.Flags().StringVar(&olderThan, "older-than", "", "select sessions older than duration (e.g. 30d, 12h)")
	cmd.Flags().StringVar(&health, "health", "", "health filter: ok|corrupted|missing-meta")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "simulate delete without changing files")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "required for real delete")
	cmd.Flags().BoolVar(&yes, "yes", false, "skip interactive prompt and approve delete")
	cmd.Flags().BoolVar(&hard, "hard", false, "hard delete (permanent)")
	cmd.Flags().BoolVar(&interactive, "interactive-confirm", true, "prompt for interactive confirmation on real delete")
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

func interactiveConfirmDelete(cmd *cobra.Command, count int, hard bool) (bool, error) {
	in := cmd.InOrStdin()
	out := cmd.ErrOrStderr()
	if !isInteractiveReader(in) {
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
		return strings.TrimSpace(text) == "DELETE", nil
	}

	if _, err := fmt.Fprintf(out, "Delete %d session(s) to trash? [y/N]: ", count); err != nil {
		return false, err
	}
	text, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	v := strings.ToLower(strings.TrimSpace(text))
	return v == "y" || v == "yes", nil
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
