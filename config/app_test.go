package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
)

func TestLoadAppConfigMissingReturnsZero(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	p := filepath.Join(workspace, "config", "missing.json")
	t.Setenv("CSM_CONFIG", p)

	cfg, err := LoadAppConfig()
	if err != nil {
		t.Fatalf("LoadAppConfig missing: %v", err)
	}
	if cfg.SessionsRoot != "" || cfg.TrashRoot != "" || cfg.LogFile != "" || cfg.TUI.Theme != "" || cfg.TUI.GroupBy != "" || cfg.TUI.Source != "" || len(cfg.TUI.Colors) != 0 {
		t.Fatalf("expected zero config for missing file, got %+v", cfg)
	}
}

func TestLoadAppConfigParsesFields(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	p := filepath.Join(workspace, "config", "config.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	data := []byte(`{
  "sessions_root":"~/.codex/sessions",
  "trash_root":"~/.codex/trash",
  "log_file":"~/.codex/codexsm/logs/actions.log",
  "tui":{
    "group_by":"host",
    "theme":"catppuccin",
    "source":"trash",
    "colors":{"keys_label":"#ffffff"}
  }
}`)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("CSM_CONFIG", p)

	cfg, err := LoadAppConfig()
	if err != nil {
		t.Fatalf("LoadAppConfig: %v", err)
	}
	if cfg.TUI.Theme != "catppuccin" || cfg.TUI.GroupBy != "host" || cfg.TUI.Source != "trash" {
		t.Fatalf("unexpected parsed config: %+v", cfg)
	}
	if cfg.TUI.Colors["keys_label"] != "#ffffff" {
		t.Fatalf("unexpected color override: %+v", cfg.TUI.Colors)
	}
}
