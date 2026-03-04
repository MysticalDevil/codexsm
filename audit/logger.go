// Package audit provides JSONL action logging and rollback lookup helpers.
package audit

import (
	"bufio"
	"crypto/rand"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/MysticalDevil/codexsm/config"
	"github.com/MysticalDevil/codexsm/session"
)

// SessionRef records a stable pointer to a session target for audit.
type SessionRef struct {
	SessionID string `json:"session_id"`
	Path      string `json:"path"`
}

// ActionRecord is one JSONL audit record for a delete command invocation.
type ActionRecord struct {
	BatchID       string                 `json:"batch_id,omitempty"`
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
	BatchID       string                 `json:"batch_id,omitempty"`
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
	HostContains string         `json:"host_contains,omitempty"`
	PathContains string         `json:"path_contains,omitempty"`
	HeadContains string         `json:"head_contains,omitempty"`
	OlderThan    string         `json:"older_than,omitempty"`
	HasOlderThan bool           `json:"has_older_than,omitempty"`
	Health       session.Health `json:"health,omitempty"`
	HasHealth    bool           `json:"has_health,omitempty"`
}

func newSelectorRecord(sel session.Selector) selectorRecord {
	out := selectorRecord{
		ID:           sel.ID,
		IDPrefix:     sel.IDPrefix,
		HostContains: sel.HostContains,
		PathContains: sel.PathContains,
		HeadContains: sel.HeadContains,
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
		BatchID:       rec.BatchID,
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

// NewBatchID returns a compact batch identifier for grouping one command invocation.
func NewBatchID() (string, error) {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate batch id: %w", err)
	}
	return fmt.Sprintf("b-%s-%x", time.Now().UTC().Format("20060102T150405Z"), b[:]), nil
}

// SessionIDsForBatchRollback returns unique session IDs soft-deleted by one batch.
func SessionIDsForBatchRollback(logFile, batchID string) ([]string, error) {
	batchID = strings.TrimSpace(batchID)
	if batchID == "" {
		return nil, errors.New("batch id is required")
	}
	records, err := readActionLog(logFile)
	if err != nil {
		return nil, err
	}

	seenBatch := false
	ids := map[string]struct{}{}
	for _, rec := range records {
		if strings.TrimSpace(rec.BatchID) != batchID {
			continue
		}
		seenBatch = true
		if rec.Simulation || rec.Action != "soft-delete" {
			continue
		}
		for _, r := range rec.Results {
			if r.Status != "deleted" || r.Destination == "" || strings.TrimSpace(r.SessionID) == "" {
				continue
			}
			ids[r.SessionID] = struct{}{}
		}
	}
	if !seenBatch {
		return nil, fmt.Errorf("batch id %q not found in log", batchID)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("batch id %q has no restorable soft-delete results", batchID)
	}
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	slices.Sort(out)
	return out, nil
}

func readActionLog(logFile string) ([]ActionRecord, error) {
	f, err := os.Open(logFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	records := make([]ActionRecord, 0, 64)
	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var raw actionRecordJSON
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			return nil, fmt.Errorf("parse log line %d: %w", lineNo, err)
		}
		sel := session.Selector{
			ID:           raw.Selector.ID,
			IDPrefix:     raw.Selector.IDPrefix,
			HostContains: raw.Selector.HostContains,
			PathContains: raw.Selector.PathContains,
			HeadContains: raw.Selector.HeadContains,
			HasOlderThan: raw.Selector.HasOlderThan,
			Health:       raw.Selector.Health,
			HasHealth:    raw.Selector.HasHealth,
		}
		if raw.Selector.HasOlderThan && strings.TrimSpace(raw.Selector.OlderThan) != "" {
			d, err := time.ParseDuration(raw.Selector.OlderThan)
			if err != nil {
				return nil, fmt.Errorf("parse log line %d older_than: %w", lineNo, err)
			}
			sel.OlderThan = d
		}
		records = append(records, ActionRecord{
			BatchID:       raw.BatchID,
			Timestamp:     raw.Timestamp,
			Action:        raw.Action,
			Simulation:    raw.Simulation,
			Selector:      sel,
			MatchedCount:  raw.MatchedCount,
			AffectedBytes: raw.AffectedBytes,
			Sessions:      raw.Sessions,
			Results:       raw.Results,
			ErrorSummary:  raw.ErrorSummary,
		})
	}
	if err := sc.Err(); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read log file: %w", err)
	}
	return records, nil
}
