// Package session implements scanning, filtering, and deletion for Codex sessions.
package session

import (
	"strings"
	"time"
)

// Health describes scanner quality classification for a session file.
type Health string

const (
	// HealthOK means the session file is readable and has a valid session_meta line.
	HealthOK Health = "ok"
	// HealthCorrupted means the session file cannot be parsed as expected.
	HealthCorrupted Health = "corrupted"
	// HealthMissingMeta means the file is readable but does not contain valid session metadata.
	HealthMissingMeta Health = "missing-meta"
)

// Session is the normalized metadata row returned by scanner and list commands.
type Session struct {
	SessionID string    `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	SizeBytes int64     `json:"size_bytes"`
	Path      string    `json:"path"`
	Health    Health    `json:"health"`
}

// Selector defines user-provided filters used by list and delete operations.
type Selector struct {
	ID           string        `json:"id,omitempty"`
	IDPrefix     string        `json:"id_prefix,omitempty"`
	OlderThan    time.Duration `json:"older_than,omitempty"`
	HasOlderThan bool          `json:"has_older_than,omitempty"`
	Health       Health        `json:"health,omitempty"`
	HasHealth    bool          `json:"has_health,omitempty"`
}

// HasAnyFilter reports whether at least one filter is set.
func (s Selector) HasAnyFilter() bool {
	return strings.TrimSpace(s.ID) != "" ||
		strings.TrimSpace(s.IDPrefix) != "" ||
		s.HasOlderThan ||
		s.HasHealth
}
