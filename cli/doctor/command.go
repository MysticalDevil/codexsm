// Package doctor provides the `codexsm doctor` command tree.
package doctor

import (
	"fmt"

	cliutil "github.com/MysticalDevil/codexsm/cli/util"
	"github.com/MysticalDevil/codexsm/usecase"
	"github.com/spf13/cobra"
)

const (
	Pass = usecase.DoctorPass
	Warn = usecase.DoctorWarn
	Fail = usecase.DoctorFail
)

// NewCommand builds the doctor command tree.
func NewCommand(
	resolveSessionsRoot func() (string, error),
	resolveTrashRoot func() (string, error),
	resolveLogFile func() (string, error),
) *cobra.Command {
	var (
		strict          bool
		compactHomePath bool
	)

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run local environment and configuration checks",
		Long: "Run local checks for codexsm runtime prerequisites.\n\n" +
			"This command validates config and storage paths.",
		Example: "  codexsm doctor\n" +
			"  codexsm doctor --strict",
		RunE: func(cmd *cobra.Command, args []string) error {
			checks := runChecks(resolveSessionsRoot, resolveTrashRoot, resolveLogFile)

			out := renderChecks(checks, cliutil.ShouldUseColor("auto", cmd.OutOrStdout()), compactHomePath)
			if _, err := fmt.Fprint(cmd.OutOrStdout(), out); err != nil {
				return err
			}

			if strict {
				for _, c := range checks {
					if c.Level == Fail || c.Level == Warn {
						return cliutil.WithExitCode(fmt.Errorf("doctor check failed: %s (%s)", c.Name, c.Level), 1)
					}
				}
			}

			for _, c := range checks {
				if c.Level == Fail {
					return cliutil.WithExitCode(fmt.Errorf("doctor check failed: %s", c.Name), 1)
				}
			}

			return nil
		},
	}
	cmd.Flags().BoolVar(&strict, "strict", false, "treat warnings as failures")
	cmd.Flags().BoolVar(&compactHomePath, "compact-home", true, "compact $HOME-prefixed detail paths as ~")
	cmd.AddCommand(newRiskCmd(resolveSessionsRoot))

	return cmd
}
