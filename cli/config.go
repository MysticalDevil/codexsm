package cli

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/MysticalDevil/codexsm/config"
	"github.com/MysticalDevil/codexsm/tui"
	"github.com/spf13/cobra"
)

type configShowOutput struct {
	Path      string            `json:"path"`
	Exists    bool              `json:"exists"`
	Config    config.AppConfig  `json:"config"`
	Effective *effectiveRuntime `json:"effective,omitempty"`
}

type effectiveRuntime struct {
	SessionsRoot string `json:"sessions_root"`
	TrashRoot    string `json:"trash_root"`
	LogFile      string `json:"log_file"`
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage codexsm config file",
		Long: "Inspect and manage the user config file.\n\n" +
			"Config path:\n" +
			"  - $CSM_CONFIG when set\n" +
			"  - ~/.config/codexsm/config.json by default",
		Example: "  codexsm config show\n" +
			"  codexsm config show --resolved\n" +
			"  codexsm config init\n" +
			"  codexsm config init --force\n" +
			"  codexsm config validate",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigInitCmd())
	cmd.AddCommand(newConfigValidateCmd())
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	var resolved bool
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Print config content",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.AppConfigPath()
			if err != nil {
				return err
			}
			cfg, err := config.LoadAppConfig()
			if err != nil {
				return err
			}
			_, statErr := os.Stat(p)
			exists := statErr == nil
			if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
				return fmt.Errorf("stat config %s: %w", p, statErr)
			}

			out := configShowOutput{
				Path:   p,
				Exists: exists,
				Config: cfg,
			}
			if resolved {
				sessionsRoot, err := runtimeSessionsRoot()
				if err != nil {
					return err
				}
				trashRoot, err := runtimeTrashRoot()
				if err != nil {
					return err
				}
				logFile, err := runtimeLogFile()
				if err != nil {
					return err
				}
				out.Effective = &effectiveRuntime{
					SessionsRoot: sessionsRoot,
					TrashRoot:    trashRoot,
					LogFile:      logFile,
				}
			}

			data, err := marshalPrettyJSON(out)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return err
		},
	}
	cmd.Flags().BoolVar(&resolved, "resolved", false, "include effective runtime paths after applying defaults")
	return cmd
}

func newConfigInitCmd() *cobra.Command {
	var (
		force  bool
		dryRun bool
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Write a starter config template",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.AppConfigPath()
			if err != nil {
				return err
			}
			if !force {
				if _, err := os.Stat(p); err == nil {
					return fmt.Errorf("config file already exists: %s (use --force to overwrite)", p)
				} else if !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("stat config %s: %w", p, err)
				}
			}

			data, err := marshalPrettyJSON(defaultAppConfigTemplate())
			if err != nil {
				return err
			}
			data = append(data, '\n')

			if dryRun {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "dry-run: would write %s\n", p); err != nil {
					return err
				}
				_, err := cmd.OutOrStdout().Write(data)
				return err
			}

			if err := config.EnsureConfigDir(); err != nil {
				return err
			}
			if err := writeFileAtomic(p, data, 0o644); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "initialized config: %s\n", p)
			return err
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing config file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print template without writing file")
	return cmd
}

func newConfigValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate config schema and key values",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.AppConfigPath()
			if err != nil {
				return err
			}
			raw, err := os.ReadFile(p)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("config file does not exist: %s", p)
				}
				return fmt.Errorf("read config %s: %w", p, err)
			}
			var cfg config.AppConfig
			if err := json.Unmarshal(raw, &cfg); err != nil {
				return fmt.Errorf("parse config %s: %w", p, err)
			}
			if err := validateAppConfig(cfg); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "valid: %s\n", p)
			return err
		},
	}
	return cmd
}

func defaultAppConfigTemplate() config.AppConfig {
	return config.AppConfig{
		SessionsRoot: "~/.codex/sessions",
		TrashRoot:    "~/.codex/trash",
		LogFile:      "~/.codex/codexsm/logs/actions.log",
		TUI: config.TUIConfig{
			GroupBy: "month",
			Theme:   tui.DefaultThemeName(),
			Source:  "sessions",
			Colors:  map[string]string{},
		},
	}
}

func validateAppConfig(cfg config.AppConfig) error {
	var errs []error

	checkPath := func(name, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		if _, err := config.ResolveConfigPath(value); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
	}
	checkPath("sessions_root", cfg.SessionsRoot)
	checkPath("trash_root", cfg.TrashRoot)
	checkPath("log_file", cfg.LogFile)

	if v := strings.ToLower(strings.TrimSpace(cfg.TUI.GroupBy)); v != "" {
		allowed := []string{"month", "day", "health", "host", "none"}
		if !slices.Contains(allowed, v) {
			errs = append(errs, fmt.Errorf("tui.group_by: invalid value %q (allowed: %s)", cfg.TUI.GroupBy, strings.Join(allowed, ", ")))
		}
	}
	if v := strings.ToLower(strings.TrimSpace(cfg.TUI.Source)); v != "" && v != "sessions" && v != "trash" {
		errs = append(errs, fmt.Errorf("tui.source: invalid value %q (allowed: sessions, trash)", cfg.TUI.Source))
	}
	if err := tui.ValidateTheme(cfg.TUI.Theme, cfg.TUI.Colors, "", nil); err != nil {
		errs = append(errs, fmt.Errorf("tui.theme/tui.colors: %w", err))
	}
	for k, v := range cfg.TUI.Colors {
		if strings.TrimSpace(k) == "" {
			errs = append(errs, errors.New("tui.colors: key must not be empty"))
			continue
		}
		if strings.TrimSpace(v) == "" {
			errs = append(errs, fmt.Errorf("tui.colors.%s: value must not be empty", k))
		}
	}

	return errors.Join(errs...)
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".codexsm-config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = os.Remove(tmpPath)
	}

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("replace config %s: %w", path, err)
	}
	return nil
}

func marshalPrettyJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.MarshalWrite(&buf, v, jsontext.Multiline(true), jsontext.WithIndent("  ")); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
