package cli

import (
	"github.com/MysticalDevil/codexsm/tui"
	"github.com/spf13/cobra"
)

func newTUICmd() *cobra.Command {
	return tui.NewCommand(tui.CommandDeps{
		ResolveSessionsRoot: runtimeSessionsRoot,
		ResolveTrashRoot:    runtimeTrashRoot,
		ResolveLogFile:      runtimeLogFile,
		TUIConfig:           runtimeConfig.TUI,
	})
}
