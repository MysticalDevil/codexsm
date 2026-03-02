package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/MysticalDevil/codex-sm/cli"
)

func main() {
	root := cli.NewRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		var ex *cli.ExitError
		if errors.As(err, &ex) {
			os.Exit(ex.ExitCode())
		}
		os.Exit(1)
	}
}
