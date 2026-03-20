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

	return cmd
}

func newExplainCmd() *cobra.Command {
	var (
		cwd          string
		format       string
		showShadowed bool
	)

	cmd := &cobra.Command{
		Use:   "explain",
		Short: "Explain AGENTS.md source chain and effective rules",
		Example: "  codexsm agents explain\n" +
			"  codexsm agents explain --cwd ~/Project/codexsm\n" +
			"  codexsm agents explain --format json\n" +
			"  codexsm agents explain --show-shadowed",
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := strings.ToLower(strings.TrimSpace(format))
			if mode == "" {
				mode = "table"
			}

			if mode != "table" && mode != "json" {
				return fmt.Errorf("invalid --format %q (allowed: table, json)", format)
			}

			out, err := usecase.ExplainAgents(usecase.AgentsExplainInput{CWD: cwd})
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

	return cmd
}

func renderExplainTable(out usecase.AgentsExplainResult, showShadowed bool) string {
	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "cwd=%s sources=%d rules=%d effective=%d shadowed=%d\n",
		out.CWD, out.Summary.Sources, out.Summary.Rules, out.Summary.Effective, out.Summary.Shadowed)

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
