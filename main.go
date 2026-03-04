package main

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/MysticalDevil/codex-sm/cli"
)

// version is injected at build time via -ldflags.
var version = "dev"

func main() {
	cli.Version = resolveVersion(version)
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

func resolveVersion(ldflagsVersion string) string {
	if strings.TrimSpace(ldflagsVersion) != "" && ldflagsVersion != "dev" {
		return ldflagsVersion
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		if v := normalizeBuildInfoVersion(bi.Main.Version); v != "" {
			return v
		}
	}
	return "dev"
}

func normalizeBuildInfoVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || v == "(devel)" {
		return ""
	}
	return v
}
