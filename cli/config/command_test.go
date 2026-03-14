package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cliutil "github.com/MysticalDevil/codexsm/cli/util"
	appconfig "github.com/MysticalDevil/codexsm/config"
	"github.com/MysticalDevil/codexsm/internal/testsupport"
)

func TestDefaultAppConfigTemplate(t *testing.T) {
	template := DefaultAppConfigTemplate()
	if template.SessionsRoot == "" || template.TrashRoot == "" || template.LogFile == "" {
		t.Fatalf("expected default paths, got %+v", template)
	}

	if template.TUI.GroupBy != "host" || template.TUI.Theme == "" || template.TUI.Source != "sessions" {
		t.Fatalf("unexpected default tui config: %+v", template.TUI)
	}
}

func TestValidateAppConfig(t *testing.T) {
	valid := DefaultAppConfigTemplate()
	if err := ValidateAppConfig(valid); err != nil {
		t.Fatalf("validateAppConfig valid: %v", err)
	}

	validDay := valid

	validDay.TUI.GroupBy = "day"
	if err := ValidateAppConfig(validDay); err != nil {
		t.Fatalf("validateAppConfig valid day group: %v", err)
	}

	badGroup := valid

	badGroup.TUI.GroupBy = "weekly"
	if err := ValidateAppConfig(badGroup); err == nil || !strings.Contains(err.Error(), "tui.group_by") {
		t.Fatalf("expected group_by validation error, got: %v", err)
	}

	badSource := valid

	badSource.TUI.Source = "archive"
	if err := ValidateAppConfig(badSource); err == nil || !strings.Contains(err.Error(), "tui.source") {
		t.Fatalf("expected source validation error, got: %v", err)
	}

	badTheme := valid

	badTheme.TUI.Theme = "not-a-theme"
	if err := ValidateAppConfig(badTheme); err == nil || !strings.Contains(err.Error(), "tui.theme") {
		t.Fatalf("expected theme validation error, got: %v", err)
	}

	badColor := valid

	badColor.TUI.Colors = map[string]string{"": "#ffffff"}
	if err := ValidateAppConfig(badColor); err == nil || !strings.Contains(err.Error(), "tui.colors") {
		t.Fatalf("expected colors validation error, got: %v", err)
	}
}

func TestWriteFileAtomic(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")

	p := filepath.Join(workspace, "config", "config.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := cliutil.WriteFileAtomic(p, []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatalf("writeFileAtomic initial: %v", err)
	}

	if err := cliutil.WriteFileAtomic(p, []byte(`{"a":2}`), 0o644); err != nil {
		t.Fatalf("writeFileAtomic replace: %v", err)
	}

	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	if strings.TrimSpace(string(data)) != `{"a":2}` {
		t.Fatalf("unexpected file data: %q", string(data))
	}
}

func TestConfigValidateCommandReadPathBranches(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")

	cfgPath := filepath.Join(workspace, "config", "config.json")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	t.Setenv("CSM_CONFIG", cfgPath)

	if err := os.WriteFile(cfgPath, []byte(`{"sessions_root":"~/.codex/sessions"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := appconfig.ResolveConfigPath("~/.codex/sessions"); err != nil {
		t.Fatalf("ResolveConfigPath sanity: %v", err)
	}
}
