package session

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var idInFilenameRe = regexp.MustCompile(`([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})\\.jsonl$`)

type metaLine struct {
	Type    string `json:"type"`
	Payload struct {
		ID        string `json:"id"`
		Timestamp string `json:"timestamp"`
	} `json:"payload"`
}

// ScanSessions walks the sessions root and parses each .jsonl file into Session metadata.
func ScanSessions(root string) ([]Session, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("sessions root is empty")
	}

	var out []Session
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".jsonl" {
			return nil
		}
		s, err := scanOne(path)
		if err != nil {
			return err
		}
		out = append(out, s)
		return nil
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Session{}, nil
		}
		return nil, err
	}
	return out, nil
}

func scanOne(path string) (Session, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Session{}, err
	}

	s := Session{
		Path:      path,
		UpdatedAt: info.ModTime(),
		SizeBytes: info.Size(),
		Health:    HealthOK,
	}

	fallbackID := sessionIDFromFilename(filepath.Base(path))
	if fallbackID != "" {
		s.SessionID = fallbackID
	}

	f, err := os.Open(path)
	if err != nil {
		s.Health = HealthCorrupted
		if s.CreatedAt.IsZero() {
			s.CreatedAt = s.UpdatedAt
		}
		return s, nil
	}
	closeScanFile := func() {
		if closeErr := f.Close(); closeErr != nil {
			s.Health = HealthCorrupted
			if s.CreatedAt.IsZero() {
				s.CreatedAt = s.UpdatedAt
			}
		}
	}

	r := bufio.NewReader(f)
	line, err := r.ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		s.Health = HealthCorrupted
		if s.CreatedAt.IsZero() {
			s.CreatedAt = s.UpdatedAt
		}
		closeScanFile()
		return s, nil
	}
	line = []byte(strings.TrimSpace(string(line)))
	if len(line) == 0 {
		s.Health = HealthMissingMeta
		s.CreatedAt = s.UpdatedAt
		closeScanFile()
		return s, nil
	}

	var m metaLine
	if err := json.Unmarshal(line, &m); err != nil {
		s.Health = HealthCorrupted
		s.CreatedAt = s.UpdatedAt
		closeScanFile()
		return s, nil
	}

	if m.Type != "session_meta" || strings.TrimSpace(m.Payload.ID) == "" {
		s.Health = HealthMissingMeta
		s.CreatedAt = s.UpdatedAt
		closeScanFile()
		return s, nil
	}

	s.SessionID = m.Payload.ID
	if ts, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(m.Payload.Timestamp)); err == nil {
		s.CreatedAt = ts
	} else {
		s.CreatedAt = s.UpdatedAt
	}

	closeScanFile()
	return s, nil
}

func sessionIDFromFilename(base string) string {
	m := idInFilenameRe.FindStringSubmatch(strings.ToLower(base))
	if len(m) != 2 {
		return ""
	}
	return m[1]
}
