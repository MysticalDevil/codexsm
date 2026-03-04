// Package config resolves runtime paths and default locations used by the CLI.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolvePath expands "~" prefixes and returns the normalized path string.
func ResolvePath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if path == "~" {
		h, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return h, nil
	}
	if strings.HasPrefix(path, "~/") {
		h, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(h, path[2:]), nil
	}
	return path, nil
}

// DefaultSessionsRoot returns the default Codex sessions directory.
func DefaultSessionsRoot() (string, error) {
	if v := strings.TrimSpace(os.Getenv("SESSIONS_ROOT")); v != "" {
		return ResolvePath(v)
	}
	return ResolvePath("~/.codex/sessions")
}

// DefaultTrashRoot returns the default trash directory for soft-deleted sessions.
func DefaultTrashRoot() (string, error) {
	return ResolvePath("~/.codex/trash")
}

// DefaultLogFile returns the default action log path.
func DefaultLogFile() (string, error) {
	return ResolvePath("~/.codex/codexsm/logs/actions.log")
}

// EnsureDirForFile creates parent directories for the given file path.
func EnsureDirForFile(filePath string) error {
	dir := filepath.Dir(filePath)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}
	return nil
}
