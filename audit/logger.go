// Package audit provides JSONL action logging for session delete operations.
package audit

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/MysticalDevil/codex-sm/config"
	"github.com/MysticalDevil/codex-sm/session"
)

// SessionRef records a stable pointer to a session target for audit.
type SessionRef struct {
	SessionID string `json:"session_id"`
	Path      string `json:"path"`
}

// ActionRecord is one JSONL audit record for a delete command invocation.
type ActionRecord struct {
	Timestamp     time.Time              `json:"timestamp"`
	Action        string                 `json:"action"`
	Simulation    bool                   `json:"simulation"`
	Selector      session.Selector       `json:"selector"`
	MatchedCount  int                    `json:"matched_count"`
	AffectedBytes int64                  `json:"affected_bytes"`
	Sessions      []SessionRef           `json:"sessions"`
	Results       []session.DeleteResult `json:"results"`
	ErrorSummary  string                 `json:"error_summary,omitempty"`
}

type actionRecordJSON struct {
	Timestamp     time.Time              `json:"timestamp"`
	Action        string                 `json:"action"`
	Simulation    bool                   `json:"simulation"`
	Selector      selectorRecord         `json:"selector"`
	MatchedCount  int                    `json:"matched_count"`
	AffectedBytes int64                  `json:"affected_bytes"`
	Sessions      []SessionRef           `json:"sessions"`
	Results       []session.DeleteResult `json:"results"`
	ErrorSummary  string                 `json:"error_summary,omitempty"`
}

type selectorRecord struct {
	ID           string         `json:"id,omitempty"`
	IDPrefix     string         `json:"id_prefix,omitempty"`
	OlderThan    string         `json:"older_than,omitempty"`
	HasOlderThan bool           `json:"has_older_than,omitempty"`
	Health       session.Health `json:"health,omitempty"`
	HasHealth    bool           `json:"has_health,omitempty"`
}

func newSelectorRecord(sel session.Selector) selectorRecord {
	out := selectorRecord{
		ID:           sel.ID,
		IDPrefix:     sel.IDPrefix,
		HasOlderThan: sel.HasOlderThan,
		Health:       sel.Health,
		HasHealth:    sel.HasHealth,
	}
	if sel.HasOlderThan {
		out.OlderThan = sel.OlderThan.String()
	}
	return out
}

// WriteActionLog appends a single JSON line action record to the log file.
func WriteActionLog(logFile string, rec ActionRecord) error {
	if err := config.EnsureDirForFile(logFile); err != nil {
		return err
	}

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	payload := actionRecordJSON{
		Timestamp:     rec.Timestamp,
		Action:        rec.Action,
		Simulation:    rec.Simulation,
		Selector:      newSelectorRecord(rec.Selector),
		MatchedCount:  rec.MatchedCount,
		AffectedBytes: rec.AffectedBytes,
		Sessions:      rec.Sessions,
		Results:       rec.Results,
		ErrorSummary:  rec.ErrorSummary,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		writeErr := fmt.Errorf("write log line: %w", err)
		if closeErr := f.Close(); closeErr != nil {
			return errors.Join(writeErr, fmt.Errorf("close log file: %w", closeErr))
		}
		return writeErr
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		writeErr := fmt.Errorf("write log line: %w", err)
		if closeErr := f.Close(); closeErr != nil {
			return errors.Join(writeErr, fmt.Errorf("close log file: %w", closeErr))
		}
		return writeErr
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close log file: %w", err)
	}
	return nil
}
