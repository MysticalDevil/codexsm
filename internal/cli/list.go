package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/MysticalDevil/codex-sm/internal/config"
	"github.com/MysticalDevil/codex-sm/internal/session"
	"github.com/MysticalDevil/codex-sm/internal/util"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var (
		sessionsRoot string
		id           string
		idPrefix     string
		olderThan    string
		health       string
		format       string
		limit        int
		detailed     bool
		pager        bool
		pageSize     int
		colorMode    string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Codex sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(sessionsRoot) == "" {
				v, err := config.DefaultSessionsRoot()
				if err != nil {
					return err
				}
				sessionsRoot = v
			} else {
				v, err := config.ResolvePath(sessionsRoot)
				if err != nil {
					return err
				}
				sessionsRoot = v
			}

			sel, err := buildSelector(id, idPrefix, olderThan, health)
			if err != nil {
				return err
			}

			sessions, err := session.ScanSessions(sessionsRoot)
			if err != nil {
				return err
			}
			filteredAll := session.FilterSessions(sessions, sel, time.Now())
			total := len(filteredAll)

			if pager && !cmd.Flags().Changed("limit") {
				limit = 0
			}
			filtered := filteredAll
			if limit > 0 && len(filtered) > limit {
				filtered = filtered[:limit]
			}

			switch strings.ToLower(strings.TrimSpace(format)) {
			case "", "table":
				table, err := renderTable(filtered, total, listRenderOptions{
					Detailed:     detailed,
					SessionsRoot: sessionsRoot,
					ColorMode:    colorMode,
					Out:          cmd.OutOrStdout(),
				})
				if err != nil {
					return err
				}
				return writeWithPager(cmd.OutOrStdout(), table, pager, pageSize)
			case "json":
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(filtered)
			default:
				return fmt.Errorf("unsupported format %q", format)
			}
		},
	}

	cmd.Flags().StringVar(&sessionsRoot, "sessions-root", "", "sessions root directory")
	cmd.Flags().StringVar(&id, "id", "", "exact session id")
	cmd.Flags().StringVar(&idPrefix, "id-prefix", "", "session id prefix")
	cmd.Flags().StringVar(&olderThan, "older-than", "", "select sessions older than duration (e.g. 30d, 12h)")
	cmd.Flags().StringVar(&health, "health", "", "health filter: ok|corrupted|missing-meta")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")
	cmd.Flags().IntVar(&limit, "limit", 10, "max rows to print (0 means unlimited)")
	cmd.Flags().BoolVar(&detailed, "detailed", false, "show detailed columns")
	cmd.Flags().BoolVar(&pager, "pager", false, "enable interactive pager")
	cmd.Flags().IntVar(&pageSize, "page-size", 10, "rows per page when --pager is enabled")
	cmd.Flags().StringVar(&colorMode, "color", "always", "color mode: auto|always|never")

	return cmd
}

type listRenderOptions struct {
	Detailed     bool
	SessionsRoot string
	ColorMode    string
	Out          io.Writer
}

func renderTable(sessions []session.Session, total int, opts listRenderOptions) (string, error) {
	useColor := shouldUseColor(opts.ColorMode, opts.Out)
	home, _ := os.UserHomeDir()

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 2, 4, 2, ' ', 0)
	if opts.Detailed {
		_, _ = fmt.Fprintln(w, "ID\tCREATED\tUPDATED\tSIZE\tHEALTH\tPATH")
	} else {
		_, _ = fmt.Fprintln(w, "ID\tUPDATED\tSIZE\tHEALTH\tNAME")
	}

	for _, s := range sessions {
		if opts.Detailed {
			_, _ = fmt.Fprintf(
				w,
				"%s\t%s\t%s\t%s\t%s\t%s\n",
				s.SessionID,
				formatDisplayTime(s.CreatedAt),
				formatDisplayTime(s.UpdatedAt),
				formatBytesIEC(s.SizeBytes),
				s.Health,
				compactHomePath(s.Path, home),
			)
			continue
		}

		_, _ = fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%s\n",
			shortID(s.SessionID),
			formatDisplayTime(s.UpdatedAt),
			formatBytesIEC(s.SizeBytes),
			s.Health,
			filepath.Base(s.Path),
		)
	}

	if err := w.Flush(); err != nil {
		return "", err
	}

	shown := len(sessions)
	footer := fmt.Sprintf("showing %d of %d", shown, total)
	if shown < total {
		footer += " (use --limit 0 for all)"
	}
	_, _ = fmt.Fprintf(&buf, "%s\n", footer)
	rendered := buf.String()
	if useColor {
		rendered = colorizeRenderedTable(rendered, opts.Detailed)
	}
	return rendered, nil
}

func writeWithPager(out io.Writer, text string, pager bool, pageSize int) error {
	if !pager || pageSize <= 0 || !isTerminalWriter(out) {
		_, err := io.WriteString(out, text)
		return err
	}

	trimmed := strings.TrimRight(text, "\n")
	if trimmed == "" {
		return nil
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) <= pageSize {
		_, err := io.WriteString(out, text)
		return err
	}

	in := bufio.NewReader(os.Stdin)
	for i := 0; i < len(lines); {
		end := i + pageSize
		if end > len(lines) {
			end = len(lines)
		}
		for _, line := range lines[i:end] {
			if _, err := fmt.Fprintln(out, line); err != nil {
				return err
			}
		}
		i = end
		if i >= len(lines) {
			break
		}
		if _, err := fmt.Fprint(out, "-- More -- (Enter/q): "); err != nil {
			return err
		}
		choice, err := in.ReadString('\n')
		if err != nil {
			return err
		}
		c := strings.ToLower(strings.TrimSpace(choice))
		if c == "q" || c == "quit" {
			break
		}
	}
	return nil
}

func shouldUseColor(mode string, out io.Writer) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "always":
		return true
	case "never":
		return false
	case "", "auto":
		if strings.EqualFold(os.Getenv("NO_COLOR"), "1") || strings.TrimSpace(os.Getenv("NO_COLOR")) != "" {
			return false
		}
		return isTerminalWriter(out)
	default:
		return isTerminalWriter(out)
	}
}

func isTerminalWriter(out io.Writer) bool {
	f, ok := out.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

const (
	ansiReset    = "\x1b[0m"
	ansiGreen    = "\x1b[32m"
	ansiYellow   = "\x1b[33m"
	ansiRed      = "\x1b[31m"
	ansiDim      = "\x1b[2m"
	ansiCyanBold = "\x1b[1;36m"
)

func colorize(v, color string, enabled bool) string {
	if !enabled || color == "" {
		return v
	}
	return color + v + ansiReset
}

func colorizeRenderedTable(text string, detailed bool) string {
	if text == "" {
		return text
	}

	hasTrailingNewline := strings.HasSuffix(text, "\n")
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if i == 0 {
			lines[i] = colorize(line, ansiCyanBold, true)
			continue
		}
		if strings.HasPrefix(line, "showing ") {
			lines[i] = colorize(line, ansiDim, true)
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		healthToken := fields[len(fields)-2]
		switch healthToken {
		case string(session.HealthOK):
			lines[i] = colorize(line, ansiGreen, true)
		case string(session.HealthMissingMeta):
			lines[i] = colorize(line, ansiYellow, true)
		case string(session.HealthCorrupted):
			lines[i] = colorize(line, ansiRed, true)
		}
	}

	out := strings.Join(lines, "\n")
	if hasTrailingNewline {
		out += "\n"
	}
	return out
}

func shortID(id string) string {
	const max = 12
	if len(id) <= max {
		return id
	}
	return id[:max]
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

func buildSelector(id, idPrefix, olderThan, health string) (session.Selector, error) {
	sel := session.Selector{
		ID:       strings.TrimSpace(id),
		IDPrefix: strings.TrimSpace(idPrefix),
	}

	if strings.TrimSpace(olderThan) != "" {
		d, err := util.ParseOlderThan(olderThan)
		if err != nil {
			return sel, err
		}
		sel.OlderThan = d
		sel.HasOlderThan = true
	}

	if strings.TrimSpace(health) != "" {
		h, err := parseHealth(health)
		if err != nil {
			return sel, err
		}
		sel.Health = h
		sel.HasHealth = true
	}

	return sel, nil
}

func parseHealth(v string) (session.Health, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case string(session.HealthOK):
		return session.HealthOK, nil
	case string(session.HealthCorrupted):
		return session.HealthCorrupted, nil
	case string(session.HealthMissingMeta):
		return session.HealthMissingMeta, nil
	default:
		return "", fmt.Errorf("invalid health %q", v)
	}
}
