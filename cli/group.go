package cli

import (
	"bytes"
	"encoding/csv"
	"encoding/json/v2"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/MysticalDevil/codexsm/internal/core"
	"github.com/MysticalDevil/codexsm/usecase"

	"github.com/spf13/cobra"
)

type groupStat = usecase.GroupStat

func newGroupCmd() *cobra.Command {
	var (
		sessionsRoot string
		id           string
		idPrefix     string
		hostContains string
		pathContains string
		headContains string
		olderThan    string
		health       string
		by           string
		sortBy       string
		order        string
		offset       int
		limit        int
		format       string
		pager        bool
		pageSize     int
		colorMode    string
	)

	cmd := &cobra.Command{
		Use:   "group",
		Short: "Group sessions by day or health",
		Example: "  codexsm group --by day\n" +
			"  codexsm group --by health\n" +
			"  codexsm group --by day --older-than 30d\n" +
			"  codexsm group --by health --host-contains /workspace --head-contains fixture\n" +
			"  codexsm group --by health --sort size --order desc --format csv",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			sessionsRoot, err = resolveOrDefault(sessionsRoot, runtimeSessionsRoot)
			if err != nil {
				return err
			}

			sel, err := buildSelector(id, idPrefix, hostContains, pathContains, headContains, olderThan, health)
			if err != nil {
				return err
			}

			stats, err := usecase.GroupSessions(usecase.GroupInput{
				SessionsRoot: sessionsRoot,
				Selector:     sel,
				By:           by,
				SortBy:       sortBy,
				Order:        order,
				Offset:       offset,
				Limit:        limit,
			})
			if err != nil {
				return err
			}

			switch strings.ToLower(strings.TrimSpace(format)) {
			case "", "table":
				table, err := renderGroupTable(stats, by, colorMode, cmd.OutOrStdout())
				if err != nil {
					return err
				}
				return writeWithPager(cmd.OutOrStdout(), table, pager, pageSize, true)
			case "json":
				b, err := json.Marshal(stats)
				if err != nil {
					return err
				}
				if _, err := cmd.OutOrStdout().Write(append(b, '\n')); err != nil {
					return err
				}
				return nil
			case "csv":
				return writeGroupDelimited(cmd.OutOrStdout(), stats, ',')
			case "tsv":
				return writeGroupDelimited(cmd.OutOrStdout(), stats, '\t')
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
	cmd.Flags().StringVarP(&by, "by", "b", "day", "group key: day|health")
	cmd.Flags().StringVarP(&sortBy, "sort", "s", "auto", "sort by: auto|group|count|size|latest")
	cmd.Flags().StringVar(&order, "order", "desc", "sort order: asc|desc")
	cmd.Flags().IntVar(&offset, "offset", 0, "skip first N groups before printing")
	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "max groups to print (0 means unlimited)")
	cmd.Flags().StringVarP(&format, "format", "f", "table", "output format: table|json|csv|tsv")
	cmd.Flags().BoolVar(&pager, "pager", false, "enable interactive pager")
	cmd.Flags().IntVar(&pageSize, "page-size", 10, "rows per page when --pager is enabled")
	cmd.Flags().StringVar(&colorMode, "color", "always", "color mode: auto|always|never")

	return cmd
}

func renderGroupTable(stats []groupStat, by, colorMode string, out io.Writer) (string, error) {
	useColor := shouldUseColor(colorMode, out)
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 2, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "GROUP\tCOUNT\tSIZE\tLATEST")
	for _, g := range stats {
		_, _ = fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", g.Group, g.Count, core.FormatBytesIEC(g.SizeBytes), g.Latest)
	}
	if err := w.Flush(); err != nil {
		return "", err
	}
	_, _ = fmt.Fprintf(&buf, "groups=%d by=%s\n", len(stats), strings.ToLower(strings.TrimSpace(by)))
	text := buf.String()
	if !useColor {
		return text, nil
	}
	return colorizeGroupedTable(text), nil
}

func colorizeGroupedTable(text string) string {
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
		if strings.HasPrefix(line, "groups=") {
			lines[i] = colorize(line, ansiDim, true)
		}
	}
	out := strings.Join(lines, "\n")
	if hasTrailingNewline {
		out += "\n"
	}
	return out
}

func writeGroupDelimited(out io.Writer, stats []groupStat, sep rune) error {
	w := csv.NewWriter(out)
	w.Comma = sep
	if err := w.Write([]string{"group", "count", "size_bytes", "latest"}); err != nil {
		return err
	}
	for _, g := range stats {
		if err := w.Write([]string{
			g.Group,
			fmt.Sprintf("%d", g.Count),
			fmt.Sprintf("%d", g.SizeBytes),
			g.Latest,
		}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}
