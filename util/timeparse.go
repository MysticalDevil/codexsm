// Package util contains shared helpers used by multiple internal packages.
package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var simpleDurationRe = regexp.MustCompile(`^(\d+)([dhmsDHMS])$`)

// ParseOlderThan parses a user-facing duration string used by --older-than.
// It supports Go duration format (e.g. 12h, 90m) and day shorthand (e.g. 30d).
func ParseOlderThan(input string) (time.Duration, error) {
	v := strings.TrimSpace(input)
	if v == "" {
		return 0, nil
	}
	if d, err := time.ParseDuration(v); err == nil {
		if d < 0 {
			return 0, fmt.Errorf("duration must be non-negative")
		}
		return d, nil
	}
	m := simpleDurationRe.FindStringSubmatch(v)
	if m == nil {
		return 0, fmt.Errorf("invalid duration %q (examples: 30d, 12h)", input)
	}
	n, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse duration number: %w", err)
	}
	switch strings.ToLower(m[2]) {
	case "d":
		return time.Duration(n) * 24 * time.Hour, nil
	case "h":
		return time.Duration(n) * time.Hour, nil
	case "m":
		return time.Duration(n) * time.Minute, nil
	case "s":
		return time.Duration(n) * time.Second, nil
	default:
		return 0, fmt.Errorf("unsupported duration unit %q", m[2])
	}
}
