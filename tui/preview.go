package tui

import (
	"os"

	"github.com/MysticalDevil/codexsm/tui/preview"
)

type angleTagTone = preview.AngleTagTone

const (
	angleTagToneDefault   = preview.AngleTagToneDefault
	angleTagToneSystem    = preview.AngleTagToneSystem
	angleTagToneLifecycle = preview.AngleTagToneLifecycle
	angleTagToneDanger    = preview.AngleTagToneDanger
	angleTagToneSuccess   = preview.AngleTagToneSuccess
)

// previewFor is a synchronous preview helper used by unit tests.
func (m *tuiModel) previewFor(path string, width, lines int) []string {
	sizeBytes := int64(0)
	updatedAtUnix := int64(0)
	if info, err := os.Stat(path); err == nil {
		sizeBytes = info.Size()
		updatedAtUnix = info.ModTime().UnixNano()
	}
	key := preview.CacheKeyForSession(path, width, sizeBytes, updatedAtUnix)
	if cached, ok := m.previewCacheGet(key); ok {
		return cached
	}
	out := buildPreviewLines(path, width, lines, m.theme)
	m.previewCachePut(key, out)
	return out
}

func buildPreviewLines(path string, width, lines int, theme tuiTheme) []string {
	return preview.BuildLines(path, width, lines, previewPalette(theme))
}

func previewPalette(theme tuiTheme) preview.ThemePalette {
	def := builtinThemes[defaultTUIThemeName()]
	return preview.ThemePalette{
		PrefixDefault:   theme.hex("prefix_default", def["prefix_default"]),
		PrefixUser:      theme.hex("prefix_user", def["prefix_user"]),
		PrefixAssistant: theme.hex("prefix_assistant", def["prefix_assistant"]),
		PrefixOther:     theme.hex("prefix_other", def["prefix_other"]),
		TagDanger:       theme.hex("tag_danger", def["tag_danger"]),
		TagDefault:      theme.hex("tag_default", def["tag_default"]),
		TagSystem:       theme.hex("tag_system", def["tag_system"]),
		TagLifecycle:    theme.hex("tag_lifecycle", def["tag_lifecycle"]),
		TagSuccess:      theme.hex("tag_success", def["tag_success"]),
	}
}

func classifyAngleTag(tag string) angleTagTone {
	return preview.ClassifyAngleTag(tag)
}
