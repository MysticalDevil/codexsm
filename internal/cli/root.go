// Package cli wires csm commands to the internal session and audit services.
package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const version = "0.1.0"

// NewRootCmd builds the top-level csm command and registers all subcommands.
func NewRootCmd() *cobra.Command {
	var (
		logLevel  string
		logFormat string
	)

	cmd := &cobra.Command{
		Use:   "csm",
		Short: "Codex session manager",
		Long: "csm manages local Codex sessions.\n\n" +
			"Use `csm help <command>` to view details for a subcommand.\n" +
			"Examples: `csm help delete`, `csm help list`, `csm help group`.",
		Example: "  csm list\n" +
			"  csm group --by day\n" +
			"  csm delete --id <session_id>\n" +
			"  csm help delete",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return configureLogger(logFormat, logLevel, cmd.ErrOrStderr())
		},
	}
	cmd.PersistentFlags().StringVar(&logLevel, "log-level", "warn", "log level: debug|info|warn|error")
	cmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log format: text|json")

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newGroupCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newRestoreCmd())
	cmd.AddCommand(newVersionCmd())
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
	b.WriteString(fmt.Sprintf("%sUse \"%s{{.CommandPath}} [command] --help%s\" for more information about a command.\n", dim, cyan, reset))
	return b.String()
}
