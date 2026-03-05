package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

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
