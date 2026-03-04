package cli

import (
	"strings"

	"github.com/MysticalDevil/codexsm/config"
)

var runtimeConfig config.AppConfig

func loadRuntimeConfig() error {
	cfg, err := config.LoadAppConfig()
	if err != nil {
		return err
	}
	runtimeConfig = cfg
	return nil
}

func runtimeSessionsRoot() (string, error) {
	if v := strings.TrimSpace(runtimeConfig.SessionsRoot); v != "" {
		return config.ResolveConfigPath(v)
	}
	return config.DefaultSessionsRoot()
}

func runtimeTrashRoot() (string, error) {
	if v := strings.TrimSpace(runtimeConfig.TrashRoot); v != "" {
		return config.ResolveConfigPath(v)
	}
	return config.DefaultTrashRoot()
}

func runtimeLogFile() (string, error) {
	if v := strings.TrimSpace(runtimeConfig.LogFile); v != "" {
		return config.ResolveConfigPath(v)
	}
	return config.DefaultLogFile()
}
