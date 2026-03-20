package agents

import (
	"fmt"
	"strings"

	cliutil "github.com/MysticalDevil/codexsm/cli/util"
	"github.com/MysticalDevil/codexsm/usecase"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Inspect effective AGENTS.md instructions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newExplainCmd())
	cmd.AddCommand(newLintCmd())

	return cmd
}

func newExplainCmd() *cobra.Command {
	var (
		cwd           string
		format        string
		showShadowed  bool
		effectiveOnly bool
		sourceFilter  string
		ruleFilter    string
	)

	cmd := &cobra.Command{
		Use:   "explain",
		Short: "Explain AGENTS.md source chain and effective rules",
		Example: "  codexsm agents explain\n" +
			"  codexsm agents explain --cwd ~/Project/codexsm\n" +
			"  codexsm agents explain --format json\n" +
			"  codexsm agents explain --show-shadowed\n" +
			"  codexsm agents explain --effective-only --source sub --rule ast-grep",
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := strings.ToLower(strings.TrimSpace(format))
			if mode == "" {
				mode = "table"
			}

			if mode != "table" && mode != "json" {
				return fmt.Errorf("invalid --format %q (allowed: table, json)", format)
			}

			out, err := usecase.ExplainAgents(usecase.AgentsExplainInput{
				CWD:           cwd,
				EffectiveOnly: effectiveOnly,
				SourceFilter:  sourceFilter,
				RuleFilter:    ruleFilter,
			})
			if err != nil {
				return err
			}

			if mode == "json" {
				b, err := cliutil.MarshalPrettyJSON(out)
				if err != nil {
					return err
				}

				_, err = fmt.Fprintln(cmd.OutOrStdout(), string(b))

				return err
			}

			_, err = fmt.Fprint(cmd.OutOrStdout(), renderExplainTable(out, showShadowed))

			return err
		},
	}
	cmd.Flags().StringVar(&cwd, "cwd", "", "target directory to evaluate (default: current working directory)")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	cmd.Flags().BoolVar(&showShadowed, "show-shadowed", false, "include rules shadowed by higher-priority sources")
	cmd.Flags().BoolVar(&effectiveOnly, "effective-only", false, "show only effective rules")
	cmd.Flags().StringVar(&sourceFilter, "source", "", "filter rules by source path substring")
	cmd.Flags().StringVar(&ruleFilter, "rule", "", "filter rules by rule text/key substring")

	return cmd
}

func renderExplainTable(out usecase.AgentsExplainResult, showShadowed bool) string {
	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "cwd=%s sources=%d rules=%d effective=%d shadowed=%d\n",
		out.CWD, out.Summary.Sources, out.Summary.Rules, out.Summary.Effective, out.Summary.Shadowed)

	if out.Filters.EffectiveOnly || strings.TrimSpace(out.Filters.SourceFilter) != "" || strings.TrimSpace(out.Filters.RuleFilter) != "" {
		_, _ = fmt.Fprintf(&b, "filters: effective_only=%t source=%q rule=%q\n",
			out.Filters.EffectiveOnly,
			out.Filters.SourceFilter,
			out.Filters.RuleFilter,
		)
	}

	if len(out.Sources) == 0 {
		b.WriteString("no AGENTS.md sources discovered\n")
		return b.String()
	}

	b.WriteString("\nSOURCES\n")

	for _, src := range out.Sources {
		_, _ = fmt.Fprintf(&b, "  [%d] %s\n", src.Priority, src.Path)
	}

	b.WriteString("\nEFFECTIVE RULES\n")

	effective := 0

	for _, r := range out.Rules {
		if r.Status != "effective" {
			continue
		}

		effective++
		_, _ = fmt.Fprintf(&b, "  - (%d) %s  [%s:%d]\n", r.Priority, r.Text, r.SourcePath, r.Line)
	}

	if effective == 0 {
		b.WriteString("  - (none)\n")
	}

	if showShadowed {
		b.WriteString("\nSHADOWED RULES\n")

		shadowed := 0

		for _, r := range out.Rules {
			if r.Status != "shadowed" {
				continue
			}

			shadowed++
			_, _ = fmt.Fprintf(&b, "  - (%d) %s  [%s:%d] -> %s\n", r.Priority, r.Text, r.SourcePath, r.Line, r.ShadowedBy)
		}

		if shadowed == 0 {
			b.WriteString("  - (none)\n")
		}
	}

	return b.String()
}

func newLintCmd() *cobra.Command {
	var (
		cwd    string
		format string
		strict bool
	)

	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Lint AGENTS.md layering for shadowed/duplicate rules",
		Example: "  codexsm agents lint\n" +
			"  codexsm agents lint --cwd ~/Project/codexsm\n" +
			"  codexsm agents lint --strict\n" +
			"  codexsm agents lint --format json",
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := strings.ToLower(strings.TrimSpace(format))
			if mode == "" {
				mode = "table"
			}

			if mode != "table" && mode != "json" {
				return fmt.Errorf("invalid --format %q (allowed: table, json)", format)
			}

			out, err := usecase.LintAgents(usecase.AgentsLintInput{CWD: cwd})
			if err != nil {
				return err
			}

			if mode == "json" {
				b, err := cliutil.MarshalPrettyJSON(out)
				if err != nil {
					return err
				}

				if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(b)); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprint(cmd.OutOrStdout(), renderLintTable(out)); err != nil {
					return err
				}
			}

			if strict && out.Summary.Warnings > 0 {
				return fmt.Errorf("agents lint failed in strict mode: warnings=%d", out.Summary.Warnings)
			}

			if out.Summary.Errors > 0 {
				return fmt.Errorf("agents lint found errors: errors=%d", out.Summary.Errors)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&cwd, "cwd", "", "target directory to evaluate (default: current working directory)")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	cmd.Flags().BoolVar(&strict, "strict", false, "treat warnings as failures")

	return cmd
}

func renderLintTable(out usecase.AgentsLintResult) string {
	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "cwd=%s sources=%d rules=%d warnings=%d errors=%d\n",
		out.CWD,
		out.Summary.Sources,
		out.Summary.Rules,
		out.Summary.Warnings,
		out.Summary.Errors,
	)

	if len(out.Issues) == 0 {
		b.WriteString("no issues\n")
		return b.String()
	}

	b.WriteString("\nISSUES\n")

	for _, issue := range out.Issues {
		_, _ = fmt.Fprintf(&b, "  - [%s] %s (%s) [%s:%d] key=%q\n",
			issue.Level,
			issue.Code,
			issue.Message,
			issue.SourcePath,
			issue.Line,
			issue.Key,
		)
	}

	return b.String()
}
