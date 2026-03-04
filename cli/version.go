package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	var short bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			if short {
				fmt.Fprintln(cmd.OutOrStdout(), Version)
				return
			}
			fmt.Fprintf(cmd.OutOrStdout(), "codexsm %s\n", Version)
		},
	}
	cmd.Flags().BoolVar(&short, "short", false, "print version only")
	return cmd
}
