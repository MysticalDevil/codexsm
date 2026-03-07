package cli

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/MysticalDevil/codexsm/session"
)

type listColumn struct {
	Key    string
	Header string
}

func parseListColumns(input string, detailed bool, format string) ([]listColumn, error) {
	defaults := []string{"id", "updated_at", "size", "health", "host", "head"}
	if detailed {
		defaults = []string{"session_id", "created_at", "updated_at", "size", "health", "host_dir", "head", "path"}
	}
	if format == "csv" || format == "tsv" {
		defaults = []string{"session_id", "created_at", "updated_at", "size_bytes", "health", "host_dir", "head", "path"}
	}

	raw := strings.TrimSpace(input)
	names := defaults
	if raw != "" {
		parts := strings.Split(raw, ",")
		names = make([]string, 0, len(parts))
		for _, p := range parts {
			name := strings.ToLower(strings.TrimSpace(p))
			if name == "" {
				continue
			}
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no columns selected")
	}

	defs := map[string]listColumn{
		"id":         {Key: "id", Header: "ID"},
		"session_id": {Key: "session_id", Header: "SESSION_ID"},
		"created_at": {Key: "created_at", Header: "CREATED_AT"},
		"updated_at": {Key: "updated_at", Header: "UPDATED_AT"},
		"size":       {Key: "size", Header: "SIZE"},
		"size_bytes": {Key: "size_bytes", Header: "SIZE_BYTES"},
		"health":     {Key: "health", Header: "HEALTH"},
		"host":       {Key: "host", Header: "HOST"},
		"host_dir":   {Key: "host_dir", Header: "HOST_DIR"},
		"path":       {Key: "path", Header: "PATH"},
		"name":       {Key: "name", Header: "NAME"},
		"head":       {Key: "head", Header: "HEAD"},
	}

	cols := make([]listColumn, 0, len(names))
	for _, n := range names {
		c, ok := defs[n]
		if !ok {
			return nil, fmt.Errorf("invalid --column value %q", n)
		}
		cols = append(cols, c)
	}
	return cols, nil
}

func listColumnValue(key string, s session.Session, home string, headWidth int, truncateHead bool) string {
	switch key {
	case "id":
		return shortID(s.SessionID)
	case "session_id":
		return s.SessionID
	case "created_at":
		return formatDisplayTime(s.CreatedAt)
	case "updated_at":
		return formatDisplayTime(s.UpdatedAt)
	case "size":
		return formatBytesIEC(s.SizeBytes)
	case "size_bytes":
		return fmt.Sprintf("%d", s.SizeBytes)
	case "health":
		return strings.ToUpper(string(s.Health))
	case "host", "host_dir":
		if strings.TrimSpace(s.HostDir) == "" {
			return "-"
		}
		return compactHomePath(s.HostDir, home)
	case "path":
		return compactHomePath(s.Path, home)
	case "name":
		return filepath.Base(s.Path)
	case "head":
		if strings.TrimSpace(s.Head) == "" {
			return "-"
		}
		if truncateHead {
			return truncateDisplayText(s.Head, headWidth)
		}
		return s.Head
	default:
		return ""
	}
}

func truncateDisplayText(v string, maxRunes int) string {
	if maxRunes <= 0 {
		return v
	}
	if utf8.RuneCountInString(v) <= maxRunes {
		return v
	}

	var b strings.Builder
	count := 0
	for _, r := range v {
		if count >= maxRunes {
			break
		}
		b.WriteRune(r)
		count++
	}
	b.WriteString("...")
	return b.String()
}

func hasHealthColumn(cols []listColumn) bool {
	for _, c := range cols {
		if c.Key == "health" {
			return true
		}
	}
	return false
}

func formatDisplayTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

func formatBytesIEC(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	value := float64(size)
	unit := -1
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	return fmt.Sprintf("%.1f%s", value, units[unit])
}

func compactHomePath(path, home string) string {
	if home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	prefix := home + string(os.PathSeparator)
	if strings.HasPrefix(path, prefix) {
		return "~" + string(os.PathSeparator) + strings.TrimPrefix(path, prefix)
	}
	return path
}

func writeListDelimited(out io.Writer, sessions []session.Session, sep rune, noHeader bool, columns []listColumn) error {
	home, _ := os.UserHomeDir()
	w := csv.NewWriter(out)
	w.Comma = sep
	if !noHeader {
		h := make([]string, 0, len(columns))
		for _, c := range columns {
			h = append(h, strings.ToLower(c.Header))
		}
		if err := w.Write(h); err != nil {
			return err
		}
	}
	for _, s := range sessions {
		record := make([]string, 0, len(columns))
		for _, c := range columns {
			switch c.Key {
			case "created_at":
				record = append(record, formatCSVTime(s.CreatedAt))
			case "updated_at":
				record = append(record, formatCSVTime(s.UpdatedAt))
			case "path":
				record = append(record, compactHomePath(s.Path, home))
			default:
				record = append(record, listColumnValue(c.Key, s, home, 0, false))
			}
		}
		if err := w.Write(record); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func formatCSVTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
