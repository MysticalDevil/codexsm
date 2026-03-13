package usecase

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/MysticalDevil/codexsm/internal/core"
	"github.com/MysticalDevil/codexsm/session"
)

type GroupInput struct {
	SessionsRoot string
	Selector     session.Selector
	Now          time.Time
	By           string
	SortBy       string
	Order        string
	Limit        int
	Repository   core.SessionRepository
}

type GroupStat struct {
	Group     string `json:"group"`
	Count     int    `json:"count"`
	SizeBytes int64  `json:"size_bytes"`
	Latest    string `json:"latest"`
}

func GroupSessions(in GroupInput) ([]GroupStat, error) {
	repo := in.Repository
	if repo == nil {
		repo = core.ScannerRepository{}
	}
	items, err := repo.ScanSessions(in.SessionsRoot)
	if err != nil {
		return nil, err
	}
	now := in.Now
	if now.IsZero() {
		now = time.Now()
	}
	filtered := session.FilterSessions(items, in.Selector, now)
	stats, err := BuildGroupStats(filtered, in.By, in.SortBy, in.Order)
	if err != nil {
		return nil, err
	}
	if in.Limit > 0 && len(stats) > in.Limit {
		stats = stats[:in.Limit]
	}
	return stats, nil
}

func BuildGroupStats(sessions []session.Session, by, sortBy, order string) ([]GroupStat, error) {
	mode := strings.ToLower(strings.TrimSpace(by))
	if mode == "" {
		mode = "day"
	}
	if mode != "day" && mode != "health" {
		return nil, fmt.Errorf("invalid --by %q (allowed: day, health)", by)
	}
	sortMode := strings.ToLower(strings.TrimSpace(sortBy))
	if sortMode == "" || sortMode == "auto" {
		if mode == "day" {
			sortMode = "group"
		} else {
			sortMode = "count"
		}
	}
	switch sortMode {
	case "group", "count", "size", "latest":
	default:
		return nil, fmt.Errorf("invalid --sort %q (allowed: auto, group, count, size, latest)", sortBy)
	}
	desc := true
	switch strings.ToLower(strings.TrimSpace(order)) {
	case "", "desc":
		desc = true
	case "asc":
		desc = false
	default:
		return nil, fmt.Errorf("invalid --order %q (allowed: asc, desc)", order)
	}

	type agg struct {
		count  int
		size   int64
		latest time.Time
	}
	m := make(map[string]*agg)
	for _, s := range sessions {
		var key string
		switch mode {
		case "day":
			key = s.UpdatedAt.Local().Format("2006-01-02")
		case "health":
			key = string(s.Health)
		}
		if key == "" {
			key = "-"
		}
		a := m[key]
		if a == nil {
			a = &agg{}
			m[key] = a
		}
		a.count++
		a.size += s.SizeBytes
		if s.UpdatedAt.After(a.latest) {
			a.latest = s.UpdatedAt
		}
	}

	out := make([]GroupStat, 0, len(m))
	for key, a := range m {
		latest := "-"
		if !a.latest.IsZero() {
			latest = formatDisplayTime(a.latest)
		}
		out = append(out, GroupStat{Group: key, Count: a.count, SizeBytes: a.size, Latest: latest})
	}

	sort.Slice(out, func(i, j int) bool {
		if desc {
			return groupLess(out[j], out[i], sortMode)
		}
		return groupLess(out[i], out[j], sortMode)
	})
	return out, nil
}

func groupLess(a, b GroupStat, sortMode string) bool {
	switch sortMode {
	case "group":
		return a.Group < b.Group
	case "count":
		if a.Count == b.Count {
			return a.Group < b.Group
		}
		return a.Count < b.Count
	case "size":
		if a.SizeBytes == b.SizeBytes {
			return a.Group < b.Group
		}
		return a.SizeBytes < b.SizeBytes
	case "latest":
		if a.Latest == b.Latest {
			return a.Group < b.Group
		}
		return a.Latest < b.Latest
	default:
		return a.Group < b.Group
	}
}

func formatDisplayTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Local().Format("2006-01-02 15:04:05")
}
