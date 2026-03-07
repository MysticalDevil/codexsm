package tui

import (
	"fmt"
	"slices"
	"strings"
)

type tuiTheme struct {
	Name   string
	Colors map[string]string
}

func (t tuiTheme) hex(key, fallback string) string {
	if t.Colors != nil {
		if v := strings.TrimSpace(t.Colors[key]); v != "" {
			return v
		}
	}
	return fallback
}

func (t tuiTheme) merge(overrides map[string]string) tuiTheme {
	if len(overrides) == 0 {
		return t
	}
	if t.Colors == nil {
		t.Colors = make(map[string]string, len(overrides))
	}
	for k, v := range overrides {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		t.Colors[strings.TrimSpace(strings.ToLower(k))] = strings.TrimSpace(v)
	}
	return t
}

func defaultTUIThemeName() string {
	return "tokyonight"
}

func availableTUIThemes() []string {
	names := make([]string, 0, len(builtinThemes))
	for name := range builtinThemes {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func resolveTUITheme(cfgName string, cfgColors map[string]string, flagName string, flagColors []string) (tuiTheme, error) {
	name := strings.ToLower(strings.TrimSpace(flagName))
	if name == "" {
		name = strings.ToLower(strings.TrimSpace(cfgName))
	}
	if name == "" || name == "custom" {
		name = defaultTUIThemeName()
	}

	base, ok := builtinThemes[name]
	if !ok {
		return tuiTheme{}, fmt.Errorf("unknown theme %q (available: %s)", name, strings.Join(availableTUIThemes(), ", "))
	}
	theme := tuiTheme{
		Name:   name,
		Colors: cloneColorMap(base),
	}
	theme = theme.merge(cfgColors)

	flagOverride, err := parseThemeOverrides(flagColors)
	if err != nil {
		return tuiTheme{}, err
	}
	theme = theme.merge(flagOverride)
	return theme, nil
}

func parseThemeOverrides(items []string) (map[string]string, error) {
	if len(items) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(items))
	for _, item := range items {
		v := strings.TrimSpace(item)
		if v == "" {
			continue
		}
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --theme-color %q (expected key=value)", item)
		}
		key := strings.TrimSpace(strings.ToLower(parts[0]))
		val := strings.TrimSpace(parts[1])
		if key == "" || val == "" {
			return nil, fmt.Errorf("invalid --theme-color %q (expected non-empty key=value)", item)
		}
		out[key] = val
	}
	return out, nil
}

func cloneColorMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func DefaultThemeName() string {
	return defaultTUIThemeName()
}

func ValidateTheme(cfgName string, cfgColors map[string]string, flagName string, flagColors []string) error {
	_, err := resolveTUITheme(cfgName, cfgColors, flagName, flagColors)
	return err
}

var builtinThemes = map[string]map[string]string{
	"tokyonight": {
		"bg":               "#1a1b26",
		"fg":               "#c0caf5",
		"border":           "#7aa2f7",
		"border_focus":     "#7dcfff",
		"title_tree":       "#73daca",
		"title_preview":    "#7dcfff",
		"group":            "#e0af68",
		"cursor_active":    "#7dcfff",
		"cursor_inactive":  "#7aa2f7",
		"selected_fg":      "#1a1b26",
		"selected_bg":      "#7dcfff",
		"keys_label":       "#bb9af7",
		"keys_key":         "#7dcfff",
		"keys_sep":         "#e0af68",
		"keys_text":        "#565f89",
		"status":           "#565f89",
		"info_header":      "#ff9e64",
		"info_value":       "#c0caf5",
		"scroll":           "#565f89",
		"scroll_active":    "#7dcfff",
		"bar":              "#73daca",
		"bar_active":       "#7dcfff",
		"prefix_user":      "#7dcfff",
		"prefix_assistant": "#7aa2f7",
		"prefix_other":     "#bb9af7",
		"prefix_default":   "#565f89",
		"tag_default":      "#e0af68",
		"tag_system":       "#7dcfff",
		"tag_lifecycle":    "#73daca",
		"tag_danger":       "#ff9e64",
		"tag_success":      "#9ece6a",
	},
	"catppuccin": {
		"bg":               "#1e1e2e",
		"fg":               "#cdd6f4",
		"border":           "#89b4fa",
		"border_focus":     "#89dceb",
		"title_tree":       "#94e2d5",
		"title_preview":    "#89dceb",
		"group":            "#f9e2af",
		"cursor_active":    "#89dceb",
		"cursor_inactive":  "#89b4fa",
		"selected_fg":      "#1e1e2e",
		"selected_bg":      "#89dceb",
		"keys_label":       "#cba6f7",
		"keys_key":         "#89dceb",
		"keys_sep":         "#f9e2af",
		"keys_text":        "#6c7086",
		"status":           "#6c7086",
		"info_header":      "#fab387",
		"info_value":       "#cdd6f4",
		"scroll":           "#6c7086",
		"scroll_active":    "#89dceb",
		"bar":              "#94e2d5",
		"bar_active":       "#89dceb",
		"prefix_user":      "#89dceb",
		"prefix_assistant": "#89b4fa",
		"prefix_other":     "#cba6f7",
		"prefix_default":   "#6c7086",
		"tag_default":      "#f9e2af",
		"tag_system":       "#89dceb",
		"tag_lifecycle":    "#94e2d5",
		"tag_danger":       "#fab387",
		"tag_success":      "#a6e3a1",
	},
	"gruvbox": {
		"bg":               "#1d2021",
		"fg":               "#ebdbb2",
		"border":           "#458588",
		"border_focus":     "#83a598",
		"title_tree":       "#8ec07c",
		"title_preview":    "#83a598",
		"group":            "#fabd2f",
		"cursor_active":    "#83a598",
		"cursor_inactive":  "#458588",
		"selected_fg":      "#1d2021",
		"selected_bg":      "#83a598",
		"keys_label":       "#d3869b",
		"keys_key":         "#83a598",
		"keys_sep":         "#fabd2f",
		"keys_text":        "#7c6f64",
		"status":           "#7c6f64",
		"info_header":      "#fe8019",
		"info_value":       "#ebdbb2",
		"scroll":           "#7c6f64",
		"scroll_active":    "#83a598",
		"bar":              "#8ec07c",
		"bar_active":       "#83a598",
		"prefix_user":      "#83a598",
		"prefix_assistant": "#458588",
		"prefix_other":     "#d3869b",
		"prefix_default":   "#7c6f64",
		"tag_default":      "#fabd2f",
		"tag_system":       "#83a598",
		"tag_lifecycle":    "#8ec07c",
		"tag_danger":       "#fe8019",
		"tag_success":      "#b8bb26",
	},
	"onedark": {
		"bg":               "#1e2127",
		"fg":               "#abb2bf",
		"border":           "#61afef",
		"border_focus":     "#56b6c2",
		"title_tree":       "#98c379",
		"title_preview":    "#56b6c2",
		"group":            "#e5c07b",
		"cursor_active":    "#56b6c2",
		"cursor_inactive":  "#61afef",
		"selected_fg":      "#1e2127",
		"selected_bg":      "#56b6c2",
		"keys_label":       "#c678dd",
		"keys_key":         "#56b6c2",
		"keys_sep":         "#e5c07b",
		"keys_text":        "#5c6370",
		"status":           "#5c6370",
		"info_header":      "#d19a66",
		"info_value":       "#abb2bf",
		"scroll":           "#5c6370",
		"scroll_active":    "#56b6c2",
		"bar":              "#98c379",
		"bar_active":       "#56b6c2",
		"prefix_user":      "#56b6c2",
		"prefix_assistant": "#61afef",
		"prefix_other":     "#c678dd",
		"prefix_default":   "#5c6370",
		"tag_default":      "#e5c07b",
		"tag_system":       "#56b6c2",
		"tag_lifecycle":    "#98c379",
		"tag_danger":       "#d19a66",
		"tag_success":      "#98c379",
	},
	"nord": {
		"bg":               "#2e3440",
		"fg":               "#d8dee9",
		"border":           "#5e81ac",
		"border_focus":     "#88c0d0",
		"title_tree":       "#8fbcbb",
		"title_preview":    "#88c0d0",
		"group":            "#ebcb8b",
		"cursor_active":    "#88c0d0",
		"cursor_inactive":  "#5e81ac",
		"selected_fg":      "#2e3440",
		"selected_bg":      "#88c0d0",
		"keys_label":       "#b48ead",
		"keys_key":         "#88c0d0",
		"keys_sep":         "#ebcb8b",
		"keys_text":        "#4c566a",
		"status":           "#4c566a",
		"info_header":      "#d08770",
		"info_value":       "#d8dee9",
		"scroll":           "#4c566a",
		"scroll_active":    "#88c0d0",
		"bar":              "#8fbcbb",
		"bar_active":       "#88c0d0",
		"prefix_user":      "#88c0d0",
		"prefix_assistant": "#5e81ac",
		"prefix_other":     "#b48ead",
		"prefix_default":   "#4c566a",
		"tag_default":      "#ebcb8b",
		"tag_system":       "#88c0d0",
		"tag_lifecycle":    "#8fbcbb",
		"tag_danger":       "#d08770",
		"tag_success":      "#a3be8c",
	},
	"dracula": {
		"bg":               "#282a36",
		"fg":               "#f8f8f2",
		"border":           "#6272a4",
		"border_focus":     "#8be9fd",
		"title_tree":       "#50fa7b",
		"title_preview":    "#8be9fd",
		"group":            "#f1fa8c",
		"cursor_active":    "#8be9fd",
		"cursor_inactive":  "#6272a4",
		"selected_fg":      "#282a36",
		"selected_bg":      "#8be9fd",
		"keys_label":       "#bd93f9",
		"keys_key":         "#8be9fd",
		"keys_sep":         "#f1fa8c",
		"keys_text":        "#6272a4",
		"status":           "#6272a4",
		"info_header":      "#ffb86c",
		"info_value":       "#f8f8f2",
		"scroll":           "#6272a4",
		"scroll_active":    "#8be9fd",
		"bar":              "#50fa7b",
		"bar_active":       "#8be9fd",
		"prefix_user":      "#8be9fd",
		"prefix_assistant": "#6272a4",
		"prefix_other":     "#bd93f9",
		"prefix_default":   "#6272a4",
		"tag_default":      "#f1fa8c",
		"tag_system":       "#8be9fd",
		"tag_lifecycle":    "#50fa7b",
		"tag_danger":       "#ffb86c",
		"tag_success":      "#50fa7b",
	},
}
