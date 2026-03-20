package session

import (
	"strings"
	"time"
)

// FilterSessions applies selector constraints and preserves input order.
func FilterSessions(sessions []Session, sel Selector, now time.Time) []Session {
	out := make([]Session, 0, len(sessions))
	for _, s := range sessions {
		if !matches(s, sel, now) {
			continue
		}

		out = append(out, s)
	}

	return out
}

func matches(s Session, sel Selector, now time.Time) bool {
	if sel.ID != "" && s.SessionID != sel.ID {
		return false
	}

	if sel.IDPrefix != "" && !strings.HasPrefix(s.SessionID, sel.IDPrefix) {
		return false
	}

	if sel.HostContains != "" && !containsFold(s.HostDir, sel.HostContains) {
		return false
	}

	if sel.PathContains != "" && !containsFold(s.Path, sel.PathContains) {
		return false
	}

	if sel.HeadContains != "" && !containsFold(s.Head, sel.HeadContains) {
		return false
	}

	if sel.HasHealth && s.Health != sel.Health {
		return false
	}

	if sel.HasOlderThan {
		cutoff := now.Add(-sel.OlderThan)
		if s.UpdatedAt.After(cutoff) {
			return false
		}
	}

	return true
}

func containsFold(src, sub string) bool {
	return strings.Contains(strings.ToLower(src), strings.ToLower(sub))
}
