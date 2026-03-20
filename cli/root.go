// Package cli wires codexsm commands to the internal session and audit services.
package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/MysticalDevil/codexsm/cli/agents"
	cfg "github.com/MysticalDevil/codexsm/cli/config"
	del "github.com/MysticalDevil/codexsm/cli/delete"
	"github.com/MysticalDevil/codexsm/cli/doctor"
	"github.com/MysticalDevil/codexsm/cli/group"
	"github.com/MysticalDevil/codexsm/cli/list"
	"github.com/MysticalDevil/codexsm/cli/restore"
	"github.com/MysticalDevil/codexsm/tui"
	"github.com/spf13/cobra"
)

// Version is the application version and is usually injected at build time.
var Version = "dev"

// NewRootCmd builds the top-level codexsm command and registers all subcommands.
func NewRootCmd() *cobra.Command {
	var (
		logLevel  string
		logFormat string
	)

	cmd := &cobra.Command{
		Use:   "codexsm",
		Short: "Codex session manager",
		Long: "codexsm manages local Codex sessions.\n\n" +
			"Build/install requires GOEXPERIMENT=jsonv2.\n\n" +
			"Use `codexsm help <command>` to view details for a subcommand.\n" +
			"Examples: `codexsm help delete`, `codexsm help list`, `codexsm help group`, `codexsm help config`, `codexsm help doctor`, `codexsm help version`.",
		Example: "  codexsm list\n" +
			"  codexsm tui\n" +
			"  codexsm group --by day\n" +
			"  codexsm config show --resolved\n" +
			"  codexsm delete --id <session_id>\n" +
			"  codexsm session migrate --from /old/path --to /new/path\n" +
			"  codexsm doctor\n" +
			"  codexsm doctor risk --format json\n" +
			"  codexsm version --short\n" +
			"  codexsm help delete",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := configureLogger(logFormat, logLevel, cmd.ErrOrStderr()); err != nil {
				return err
			}

			return loadRuntimeConfig()
		},
	}
	cmd.PersistentFlags().StringVar(&logLevel, "log-level", "warn", "log level: debug|info|warn|error")
	cmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log format: text|json")

	cmd.AddCommand(list.NewCommand(runtimeSessionsRoot))
	cmd.AddCommand(group.NewCommand(runtimeSessionsRoot))
	cmd.AddCommand(del.NewCommand(runtimeSessionsRoot, runtimeTrashRoot, runtimeLogFile, runtimeNewBatchID, runtimeWriteActionLog, time.Now))
	cmd.AddCommand(restore.NewCommand(runtimeSessionsRoot, runtimeTrashRoot, runtimeLogFile, runtimeNewBatchID, runtimeWriteActionLog, time.Now))
	cmd.AddCommand(newSessionCmd())
	cmd.AddCommand(agents.NewCommand())
	cmd.AddCommand(cfg.NewCommand(runtimeSessionsRoot, runtimeTrashRoot, runtimeLogFile))
	cmd.AddCommand(tui.NewCommand(tui.CommandDeps{
		ResolveSessionsRoot: runtimeSessionsRoot,
		ResolveTrashRoot:    runtimeTrashRoot,
		ResolveLogFile:      runtimeLogFile,
		TUIConfig:           runtimeConfig.TUI,
	}))
	cmd.AddCommand(newCompletionCmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(doctor.NewCommand(runtimeSessionsRoot, runtimeTrashRoot, runtimeLogFile))
	applyHelpStyles(cmd)

	return cmd
}

func applyHelpStyles(root *cobra.Command) {
	helpTemplate := buildHelpTemplate()

	var walk func(*cobra.Command)

	walk = func(c *cobra.Command) {
		c.SetHelpTemplate(helpTemplate)

		for _, sc := range c.Commands() {
			walk(sc)
		}
	}
	walk(root)
}

func buildHelpTemplate() string {
	cyan := ansiCyanBold
	dim := ansiDim
	reset := ansiReset

	section := func(title string) string {
		return fmt.Sprintf("%s%s%s", cyan, title, reset)
	}

	var b strings.Builder
	b.WriteString("{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}\n\n{{end}}")
	b.WriteString("{{if or .Runnable .HasSubCommands}}")
	b.WriteString(section("Usage:"))
	b.WriteString("\n  {{.UseLine}}\n{{end}}")

	b.WriteString("{{if .HasAvailableSubCommands}}\n")
	b.WriteString(section("Available Commands:"))
	b.WriteString("\n{{range .Commands}}{{if (and .IsAvailableCommand (not .Hidden))}}  {{rpad .Name .NamePadding }} {{.Short}}\n{{end}}{{end}}{{end}}")

	b.WriteString("{{if .HasAvailableLocalFlags}}\n")
	b.WriteString(section("Flags:"))
	b.WriteString("\n{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}\n{{end}}")

	b.WriteString("{{if .HasAvailableInheritedFlags}}\n")
	b.WriteString(section("Global Flags:"))
	b.WriteString("\n{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}\n{{end}}")

	b.WriteString("{{if .HasExample}}\n")
	b.WriteString(section("Examples:"))
	b.WriteString("\n{{.Example}}\n{{end}}")

	b.WriteString("{{if .HasHelpSubCommands}}\n")
	b.WriteString(section("Additional Help Topics:"))
	b.WriteString("\n{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}\n{{end}}{{end}}{{end}}")

	b.WriteString("\n")
	fmt.Fprintf(&b, "%sUse \"%s{{.CommandPath}} [command] --help%s\" for more information about a command.\n", dim, cyan, reset)

	return b.String()
}
