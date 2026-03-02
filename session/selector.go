package session

import (
	"sort"
	"strings"
	"time"
)

// FilterSessions applies selector constraints and returns results ordered by UpdatedAt desc.
func FilterSessions(sessions []Session, sel Selector, now time.Time) []Session {
	out := make([]Session, 0, len(sessions))
	for _, s := range sessions {
		if !matches(s, sel, now) {
			continue
		}
		out = append(out, s)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

func matches(s Session, sel Selector, now time.Time) bool {
	if sel.ID != "" && s.SessionID != sel.ID {
		return false
	}
	if sel.IDPrefix != "" && !strings.HasPrefix(s.SessionID, sel.IDPrefix) {
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
