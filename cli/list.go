package cli

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json/v2"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/tabwriter"
	"time"
	"unicode/utf8"

	"github.com/MysticalDevil/codex-sm/config"
	"github.com/MysticalDevil/codex-sm/session"
	"github.com/MysticalDevil/codex-sm/util"

	"github.com/spf13/cobra"
)

type listColumn struct {
	Key    string
	Header string
}

type listRenderOptions struct {
	Detailed  bool
	NoHeader  bool
	ColorMode string
	Out       io.Writer
	Columns   []listColumn
	HeadWidth int
}

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
		noHeader     bool
		column       string
		headWidth    int
		sortBy       string
		order        string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Codex sessions",
		Example: "  csm list\n" +
			"  csm list --detailed\n" +
			"  csm list --head-width 48\n" +
			"  csm list --limit 0 --pager\n" +
			"  csm list --sort size --order asc --limit 20\n" +
			"  csm list --id-prefix 019ca9 --format json\n" +
			"  csm list --format csv --column session_id,updated_at,size",
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
			if err := sortSessions(filteredAll, sortBy, order); err != nil {
				return err
			}
			total := len(filteredAll)

			if pager && !cmd.Flags().Changed("limit") {
				limit = 0
			}
			filtered := filteredAll
			if limit > 0 && len(filtered) > limit {
				filtered = filtered[:limit]
			}

			formatMode := strings.ToLower(strings.TrimSpace(format))
			if formatMode == "" {
				formatMode = "table"
			}
			if formatMode == "json" && (noHeader || strings.TrimSpace(column) != "") {
				return fmt.Errorf("--no-header and --column are only supported with table/csv/tsv")
			}

			columns, err := parseListColumns(column, detailed, formatMode)
			if err != nil {
				return err
			}

			switch formatMode {
			case "table":
				table, err := renderTable(filtered, total, listRenderOptions{
					Detailed:  detailed,
					NoHeader:  noHeader,
					ColorMode: colorMode,
					Out:       cmd.OutOrStdout(),
					Columns:   columns,
					HeadWidth: headWidth,
				})
				if err != nil {
					return err
				}
				return writeWithPager(cmd.OutOrStdout(), table, pager, pageSize, !noHeader)
			case "json":
				b, err := json.Marshal(filtered)
				if err != nil {
					return err
				}
				if _, err := cmd.OutOrStdout().Write(append(b, '\n')); err != nil {
					return err
				}
				return nil
			case "csv":
				return writeListDelimited(cmd.OutOrStdout(), filtered, ',', noHeader, columns)
			case "tsv":
				return writeListDelimited(cmd.OutOrStdout(), filtered, '\t', noHeader, columns)
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
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json|csv|tsv")
	cmd.Flags().IntVar(&limit, "limit", 10, "max rows to print (0 means unlimited)")
	cmd.Flags().BoolVar(&detailed, "detailed", false, "show detailed columns")
	cmd.Flags().BoolVar(&pager, "pager", false, "enable interactive pager")
	cmd.Flags().IntVar(&pageSize, "page-size", 10, "rows per page when --pager is enabled")
	cmd.Flags().StringVar(&colorMode, "color", "always", "color mode: auto|always|never")
	cmd.Flags().BoolVar(&noHeader, "no-header", false, "hide header row for table/csv/tsv")
	cmd.Flags().StringVar(&column, "column", "", "comma-separated columns (e.g. session_id,updated_at,size)")
	cmd.Flags().IntVar(&headWidth, "head-width", 36, "max HEAD width in table format (0 means no truncation)")
	cmd.Flags().StringVar(&sortBy, "sort", "updated_at", "sort field: updated_at|created_at|size|health|id|session_id")
	cmd.Flags().StringVar(&order, "order", "desc", "sort order: asc|desc")

	return cmd
}

func sortSessions(items []session.Session, sortBy, order string) error {
	if len(items) <= 1 {
		return nil
	}

	by := strings.ToLower(strings.TrimSpace(sortBy))
	if by == "" {
		by = "updated_at"
	}
	switch by {
	case "updated_at", "created_at", "size", "health", "id", "session_id":
	default:
		return fmt.Errorf("invalid --sort value %q", sortBy)
	}

	desc := true
	switch strings.ToLower(strings.TrimSpace(order)) {
	case "", "desc":
		desc = true
	case "asc":
		desc = false
	default:
		return fmt.Errorf("invalid --order value %q", order)
	}

	healthRank := func(h session.Health) int {
		switch h {
		case session.HealthOK:
			return 0
		case session.HealthMissingMeta:
			return 1
		case session.HealthCorrupted:
			return 2
		default:
			return 3
		}
	}

	compare := func(a, b session.Session) int {
		switch by {
		case "updated_at":
			return a.UpdatedAt.Compare(b.UpdatedAt)
		case "created_at":
			return a.CreatedAt.Compare(b.CreatedAt)
		case "size":
			if a.SizeBytes < b.SizeBytes {
				return -1
			}
			if a.SizeBytes > b.SizeBytes {
				return 1
			}
			return 0
		case "health":
			ra := healthRank(a.Health)
			rb := healthRank(b.Health)
			if ra < rb {
				return -1
			}
			if ra > rb {
				return 1
			}
			return 0
		case "id", "session_id":
			return strings.Compare(a.SessionID, b.SessionID)
		default:
			return 0
		}
	}

	slices.SortStableFunc(items, func(a, b session.Session) int {
		c := compare(a, b)
		if c == 0 {
			c = strings.Compare(a.SessionID, b.SessionID)
		}
		if c == 0 {
			c = strings.Compare(a.Path, b.Path)
		}
		if desc {
			c = -c
		}
		return c
	})
	return nil
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

func renderTable(sessions []session.Session, total int, opts listRenderOptions) (string, error) {
	useColor := shouldUseColor(opts.ColorMode, opts.Out)
	home, _ := os.UserHomeDir()

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 2, 4, 2, ' ', 0)
	if !opts.NoHeader {
		headers := make([]string, 0, len(opts.Columns))
		for _, c := range opts.Columns {
			headers = append(headers, c.Header)
		}
		_, _ = fmt.Fprintln(w, strings.Join(headers, "\t"))
	}

	for _, s := range sessions {
		values := make([]string, 0, len(opts.Columns))
		for _, c := range opts.Columns {
			values = append(values, listColumnValue(c.Key, s, home, opts.HeadWidth, true))
		}
		_, _ = fmt.Fprintln(w, strings.Join(values, "\t"))
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
		rendered = colorizeRenderedTable(rendered, sessions, opts.NoHeader, hasHealthColumn(opts.Columns))
	}
	return rendered, nil
}

func writeWithPager(out io.Writer, text string, pager bool, pageSize int, hasHeader bool) error {
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

	header := ""
	footer := ""
	bodyStart := 0
	bodyEnd := len(lines)
	if hasHeader && len(lines) > 0 {
		header = lines[0]
		bodyStart = 1
	}
	if bodyEnd > bodyStart && strings.HasPrefix(stripANSI(strings.TrimSpace(lines[bodyEnd-1])), "showing ") {
		footer = lines[bodyEnd-1]
		bodyEnd--
	}
	body := lines[bodyStart:bodyEnd]
	if len(body) == 0 {
		_, err := io.WriteString(out, text)
		return err
	}

	pages := (len(body) + pageSize - 1) / pageSize
	page := 0
	in := bufio.NewReader(os.Stdin)

	renderPage := func(page int) error {
		// Redraw one terminal-sized page instead of appending pages.
		if _, err := fmt.Fprint(out, "\x1b[H\x1b[2J"); err != nil {
			return err
		}
		start := page * pageSize
		end := start + pageSize
		if end > len(body) {
			end = len(body)
		}
		if header != "" {
			if _, err := fmt.Fprintln(out, header); err != nil {
				return err
			}
		}
		for _, line := range body[start:end] {
			if _, err := fmt.Fprintln(out, line); err != nil {
				return err
			}
		}
		if footer != "" {
			if _, err := fmt.Fprintln(out, footer); err != nil {
				return err
			}
		}
		return nil
	}

	for {
		if err := renderPage(page); err != nil {
			return err
		}
		if page >= pages-1 {
			break
		}

		if _, err := fmt.Fprintf(out, "-- Page %d/%d -- [j next, k back, g first, G last, a all, q quit]: ", page+1, pages); err != nil {
			return err
		}
		choice, err := in.ReadString('\n')
		if err != nil {
			return err
		}
		if _, err := fmt.Fprint(out, "\r\033[2K"); err != nil {
			return err
		}
		nextPage, act := applyPagerChoice(page, pages, choice)
		page = nextPage
		if act == pagerActionQuit {
			break
		}
		if act == pagerActionAll {
			// Stream remaining rows continuously from the current position.
			if _, err := fmt.Fprint(out, "\x1b[H\x1b[2J"); err != nil {
				return err
			}
			if header != "" {
				if _, err := fmt.Fprintln(out, header); err != nil {
					return err
				}
			}
			for p := page; p < pages; p++ {
				start := p * pageSize
				end := start + pageSize
				if end > len(body) {
					end = len(body)
				}
				for _, line := range body[start:end] {
					if _, err := fmt.Fprintln(out, line); err != nil {
						return err
					}
				}
			}
			if footer != "" {
				if _, err := fmt.Fprintln(out, footer); err != nil {
					return err
				}
			}
			break
		}
	}
	return nil
}

type pagerAction int

const (
	pagerActionContinue pagerAction = iota
	pagerActionQuit
	pagerActionAll
)

func applyPagerChoice(page, pages int, rawChoice string) (int, pagerAction) {
	if pages <= 0 {
		return page, pagerActionQuit
	}
	last := pages - 1
	clean := strings.TrimSpace(rawChoice)
	if clean == "G" {
		return last, pagerActionContinue
	}
	c := strings.ToLower(clean)
	switch c {
	case "q", "quit":
		return page, pagerActionQuit
	case "a", "all":
		return page, pagerActionAll
	case "g", "first", "home":
		return 0, pagerActionContinue
	case "b", "back", "p", "prev", "k":
		if page > 0 {
			return page - 1, pagerActionContinue
		}
		return 0, pagerActionContinue
	case "", "j", "n", "next", " ":
		if page < last {
			return page + 1, pagerActionContinue
		}
		return last, pagerActionContinue
	default:
		if page < last {
			return page + 1, pagerActionContinue
		}
		return last, pagerActionContinue
	}
}

var ansiColorRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(v string) string {
	return ansiColorRe.ReplaceAllString(v, "")
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

func colorizeRenderedTable(text string, sessions []session.Session, noHeader, hasHealth bool) string {
	if text == "" {
		return text
	}

	hasTrailingNewline := strings.HasSuffix(text, "\n")
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	dataStart := 0
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !noHeader && i == 0 {
			lines[i] = colorize(line, ansiCyanBold, true)
			dataStart = 1
			continue
		}
		if strings.HasPrefix(line, "showing ") {
			lines[i] = colorize(line, ansiDim, true)
			continue
		}
		if hasHealth {
			idx := i - dataStart
			if idx >= 0 && idx < len(sessions) {
				switch sessions[idx].Health {
				case session.HealthOK:
					lines[i] = colorize(line, ansiGreen, true)
				case session.HealthMissingMeta:
					lines[i] = colorize(line, ansiYellow, true)
				case session.HealthCorrupted:
					lines[i] = colorize(line, ansiRed, true)
				}
			}
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
		return string(s.Health)
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
