// Package audit provides JSONL action logging for session delete operations.
package audit

import (
	"encoding/json"
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

// WriteActionLog appends a single JSON line action record to the log file.
func WriteActionLog(logFile string, rec ActionRecord) error {
	if err := config.EnsureDirForFile(logFile); err != nil {
		return err
	}

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	enc := json.NewEncoder(f)
	if err := enc.Encode(rec); err != nil {
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
