package cli

import (
	"fmt"
	"strings"
	"time"

	cliutil "github.com/MysticalDevil/codexsm/cli/util"
	"github.com/MysticalDevil/codexsm/config"
	"github.com/MysticalDevil/codexsm/session/migrate"
	"github.com/spf13/cobra"
)

func newMigrateCmd() *cobra.Command {
	var (
		fromPath     string
		toPath       string
		filePath     string
		branch       string
		sessionsRoot string
		stateDB      string
		limit        int
		sinceRaw     string
		dryRun       bool
		confirm      bool
		printCreated bool
	)

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Copy Codex sessions to a new cwd and keep Resume compatibility",
		Long: "Copy sessions matched by source cwd to a new destination cwd.\n\n" +
			"The command copies rollout files and clones matching Codex thread index rows.\n" +
			"By default it runs in dry-run mode and prints the migration plan without writing.",
		Example: "  codexsm session migrate --from /path/to/source/project --to /path/to/target/worktree\n" +
			"  codexsm session migrate --from /old/path --to /new/path --dry-run=false --confirm\n" +
			"  codexsm session migrate --file ./migrate.toml\n" +
			"  codexsm session migrate --from /old/path --to /new/path --branch main --since 2026-03-10",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			sessionsRoot, err = cliutil.ResolveOrDefault(sessionsRoot, runtimeSessionsRoot)
			if err != nil {
				return err
			}

			stateDB, err = cliutil.ResolveOrDefault(stateDB, config.DefaultCodexStateDB)
			if err != nil {
				return err
			}

			since, hasSince, err := parseSinceTime(sinceRaw)
			if err != nil {
				return err
			}

			filePath = strings.TrimSpace(filePath)
			fromPath = strings.TrimSpace(fromPath)

			toPath = strings.TrimSpace(toPath)
			switch {
			case filePath != "" && (fromPath != "" || toPath != ""):
				return cliutil.WithExitCode(fmt.Errorf("--file cannot be combined with --from or --to"), 1)
			case filePath == "" && (fromPath == "" || toPath == ""):
				return cliutil.WithExitCode(fmt.Errorf("either --file or both --from and --to are required"), 1)
			}

			if filePath != "" {
				result, err := migrate.MigrateSessionsBatch(migrate.MigrateBatchOptions{
					FilePath:     filePath,
					SessionsRoot: sessionsRoot,
					StateDBPath:  stateDB,
					Limit:        limit,
					Since:        since,
					HasSince:     hasSince,
					DryRun:       dryRun,
					Confirm:      confirm,
					PrintCreated: printCreated,
				})
				printMigrateBatchResult(cmd, result)

				if err != nil {
					return cliutil.WithExitCode(err, 1)
				}

				return nil
			}

			result, err := migrate.MigrateSessions(migrate.MigrateOptions{
				FromCWD:      fromPath,
				ToCWD:        toPath,
				Branch:       branch,
				SessionsRoot: sessionsRoot,
				StateDBPath:  stateDB,
				Limit:        limit,
				Since:        since,
				HasSince:     hasSince,
				DryRun:       dryRun,
				Confirm:      confirm,
				PrintCreated: printCreated,
			})
			if err != nil {
				return cliutil.WithExitCode(err, 1)
			}

			printMigrateResult(cmd, result)

			return nil
		},
	}

	cmd.Flags().StringVar(&fromPath, "from", "", "source cwd to match in existing sessions")
	cmd.Flags().StringVar(&toPath, "to", "", "destination cwd to write into copied sessions")
	cmd.Flags().StringVar(&filePath, "file", "", "TOML file containing [[mapping]] entries for batch migration")
	cmd.Flags().StringVar(&branch, "branch", "", "override git branch recorded in cloned thread rows")
	cmd.Flags().StringVar(&sessionsRoot, "sessions-root", "", "sessions root directory")
	cmd.Flags().StringVar(&stateDB, "codex-state-db", "", "Codex local sqlite state database")
	cmd.Flags().IntVar(&limit, "limit", 0, "max number of sessions to migrate after filters")
	cmd.Flags().StringVar(&sinceRaw, "since", "", "include sessions updated at or after RFC3339 timestamp or YYYY-MM-DD date")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "simulate migration without writing files or sqlite rows")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "required for real migration")
	cmd.Flags().BoolVar(&printCreated, "print-created", false, "print source id to destination id mappings")

	return cmd
}

func parseSinceTime(raw string) (time.Time, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false, nil
	}

	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts, true, nil
		}
	}

	return time.Time{}, false, fmt.Errorf("invalid --since value %q: expected RFC3339 timestamp or YYYY-MM-DD", raw)
}

func printMigrateResult(cmd *cobra.Command, result migrate.MigrateResult) {
	out := cmd.OutOrStdout()

	action := "dry-run"
	if !result.DryRun {
		action = "migrate"
	}

	_, _ = fmt.Fprintf(out, "session-migrate: action=%s matched=%d planned=%d created=%d skipped=%d warnings=%d\n",
		action, result.Matched, len(result.Planned), result.Created, result.Skipped, len(result.Warnings))
	_, _ = fmt.Fprintf(out, "from=%s\n", result.FromCWD)

	_, _ = fmt.Fprintf(out, "to=%s\n", result.ToCWD)
	if result.DestBranch != "" {
		_, _ = fmt.Fprintf(out, "branch=%s\n", result.DestBranch)
	}

	if result.PrintCreated && len(result.Planned) > 0 {
		_, _ = fmt.Fprintln(out, "mappings:")
		for _, p := range result.Planned {
			_, _ = fmt.Fprintf(out, "- %s -> %s %s\n", p.SourceID, p.DestID, p.DestRollout)
		}
	}

	if len(result.Warnings) > 0 {
		_, _ = fmt.Fprintln(out, "warnings:")
		for _, warning := range result.Warnings {
			_, _ = fmt.Fprintf(out, "- %s\n", warning)
		}
	}
}

func printMigrateBatchResult(cmd *cobra.Command, result migrate.MigrateBatchResult) {
	out := cmd.OutOrStdout()

	action := "dry-run"
	if !result.DryRun {
		action = "migrate"
	}

	_, _ = fmt.Fprintf(out, "session-migrate-batch: action=%s mappings=%d succeeded=%d failed=%d matched=%d created=%d skipped=%d\n",
		action, result.TotalMappings, result.Succeeded, result.Failed, result.Matched, result.Created, result.Skipped)
	for i, item := range result.Items {
		status := "ok"
		if item.Err != nil {
			status = "error"
		}

		_, _ = fmt.Fprintf(out, "mapping[%d]: status=%s from=%s to=%s matched=%d created=%d skipped=%d\n",
			i+1, status, item.Mapping.FromCWD, item.Mapping.ToCWD, item.Result.Matched, item.Result.Created, item.Result.Skipped)
		if item.Mapping.Branch != "" {
			_, _ = fmt.Fprintf(out, "branch=%s\n", item.Mapping.Branch)
		}

		if item.Err != nil {
			_, _ = fmt.Fprintf(out, "error=%s\n", item.Err)
			continue
		}

		if result.PrintCreated && len(item.Result.Planned) > 0 {
			_, _ = fmt.Fprintln(out, "mappings:")
			for _, p := range item.Result.Planned {
				_, _ = fmt.Fprintf(out, "- %s -> %s %s\n", p.SourceID, p.DestID, p.DestRollout)
			}
		}

		if len(item.Result.Warnings) > 0 {
			_, _ = fmt.Fprintln(out, "warnings:")
			for _, warning := range item.Result.Warnings {
				_, _ = fmt.Fprintf(out, "- %s\n", warning)
			}
		}
	}
}
