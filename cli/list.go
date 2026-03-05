package cli

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/MysticalDevil/codexsm/config"
	"github.com/MysticalDevil/codexsm/session"
	"github.com/MysticalDevil/codexsm/util"

	"github.com/spf13/cobra"
)

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
		hostContains string
		pathContains string
		headContains string
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
		Example: "  codexsm list\n" +
			"  codexsm list --detailed\n" +
			"  codexsm list --head-width 48\n" +
			"  codexsm list --limit 0 --pager\n" +
			"  codexsm list --sort size --order asc --limit 20\n" +
			"  codexsm list --host-contains /workspace --head-contains fixture\n" +
			"  codexsm list --id-prefix 019ca9 --format json\n" +
			"  codexsm list --format csv --column session_id,updated_at,size",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if strings.TrimSpace(sessionsRoot) == "" {
				sessionsRoot, err = runtimeSessionsRoot()
				if err != nil {
					return err
				}
			} else {
				sessionsRoot, err = config.ResolvePath(sessionsRoot)
				if err != nil {
					return err
				}
			}

			sel, err := buildSelector(id, idPrefix, hostContains, pathContains, headContains, olderThan, health)
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
				out := cmd.OutOrStdout()
				table, err := renderTable(filtered, total, listRenderOptions{
					Detailed:  detailed,
					NoHeader:  noHeader,
					ColorMode: colorMode,
					Out:       out,
					Columns:   columns,
					HeadWidth: headWidth,
				})
				if err != nil {
					return err
				}
				return writeWithPager(out, table, pager, pageSize, !noHeader)
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
	cmd.Flags().StringVarP(&id, "id", "i", "", "exact session id")
	cmd.Flags().StringVarP(&idPrefix, "id-prefix", "p", "", "session id prefix")
	cmd.Flags().StringVar(&hostContains, "host-contains", "", "case-insensitive substring match against host path")
	cmd.Flags().StringVar(&pathContains, "path-contains", "", "case-insensitive substring match against session file path")
	cmd.Flags().StringVar(&headContains, "head-contains", "", "case-insensitive substring match against preview head text")
	cmd.Flags().StringVarP(&olderThan, "older-than", "o", "", "select sessions older than duration (e.g. 30d, 12h)")
	cmd.Flags().StringVarP(&health, "health", "H", "", "health filter: ok|corrupted|missing-meta")
	cmd.Flags().StringVarP(&format, "format", "f", "table", "output format: table|json|csv|tsv")
	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "max rows to print (0 means unlimited)")
	cmd.Flags().BoolVar(&detailed, "detailed", false, "show detailed columns")
	cmd.Flags().BoolVar(&pager, "pager", false, "enable interactive pager")
	cmd.Flags().IntVar(&pageSize, "page-size", 10, "rows per page when --pager is enabled")
	cmd.Flags().StringVar(&colorMode, "color", "always", "color mode: auto|always|never")
	cmd.Flags().BoolVar(&noHeader, "no-header", false, "hide header row for table/csv/tsv")
	cmd.Flags().StringVar(&column, "column", "", "comma-separated columns (e.g. session_id,updated_at,size)")
	cmd.Flags().IntVar(&headWidth, "head-width", 36, "max HEAD width in table format (0 means no truncation)")
	cmd.Flags().StringVarP(&sortBy, "sort", "s", "updated_at", "sort field: updated_at|created_at|size|health|id|session_id")
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

func buildSelector(id, idPrefix, hostContains, pathContains, headContains, olderThan, health string) (session.Selector, error) {
	sel := session.Selector{
		ID:           strings.TrimSpace(id),
		IDPrefix:     strings.TrimSpace(idPrefix),
		HostContains: strings.TrimSpace(hostContains),
		PathContains: strings.TrimSpace(pathContains),
		HeadContains: strings.TrimSpace(headContains),
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
