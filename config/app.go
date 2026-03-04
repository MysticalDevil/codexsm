package config

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AppConfig is the user-level runtime configuration for codexsm.
type AppConfig struct {
	SessionsRoot string    `json:"sessions_root"`
	TrashRoot    string    `json:"trash_root"`
	LogFile      string    `json:"log_file"`
	TUI          TUIConfig `json:"tui"`
}

// TUIConfig contains defaults and visual options for TUI mode.
type TUIConfig struct {
	GroupBy string            `json:"group_by"`
	Theme   string            `json:"theme"`
	Colors  map[string]string `json:"colors"`
	Source  string            `json:"source"`
}

// AppConfigPath resolves the effective config path.
func AppConfigPath() (string, error) {
	if v := strings.TrimSpace(os.Getenv("CSM_CONFIG")); v != "" {
		return ResolvePath(v)
	}
	return ResolvePath("~/.config/codexsm/config.json")
}

// LoadAppConfig reads and parses user config.
// Missing config file is treated as zero-value config.
func LoadAppConfig() (AppConfig, error) {
	p, err := AppConfigPath()
	if err != nil {
		return AppConfig{}, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return AppConfig{}, nil
		}
		return AppConfig{}, fmt.Errorf("read config %s: %w", p, err)
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return AppConfig{}, fmt.Errorf("parse config %s: %w", p, err)
	}
	return cfg, nil
}

// ResolveConfigPath resolves path-like fields and preserves empty values.
func ResolveConfigPath(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", nil
	}
	return ResolvePath(v)
}

// EnsureConfigDir ensures config parent directory exists.
func EnsureConfigDir() error {
	p, err := AppConfigPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(p)
	return os.MkdirAll(dir, 0o755)
}
