package session

import (
	"bufio"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

var idInFilenameRe = regexp.MustCompile(`([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})\\.jsonl$`)

type metaLine struct {
	Type    string `json:"type"`
	Payload struct {
		ID        string `json:"id"`
		Timestamp string `json:"timestamp"`
		Cwd       string `json:"cwd"`
	} `json:"payload"`
}

type responseItemLine struct {
	Type    string `json:"type"`
	Payload struct {
		Type    string `json:"type"`
		Role    string `json:"role"`
		Text    string `json:"text"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
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
	if !jsontext.Value(line).IsValid() {
		s.Health = HealthCorrupted
		s.CreatedAt = s.UpdatedAt
		closeScanFile()
		return s, nil
	}
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
	s.HostDir = strings.TrimSpace(m.Payload.Cwd)
	if ts, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(m.Payload.Timestamp)); err == nil {
		s.CreatedAt = ts
	} else {
		s.CreatedAt = s.UpdatedAt
	}
	s.Head = readConversationHead(r)

	closeScanFile()
	return s, nil
}

func readConversationHead(r *bufio.Reader) string {
	const maxLines = 1024
	const maxCandidates = 24
	candidates := make([]string, 0, maxCandidates)
	for i := 0; i < maxLines; i++ {
		line, err := r.ReadBytes('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			break
		}
		line = []byte(strings.TrimSpace(string(line)))
		if len(line) > 0 {
			head := conversationHeadFromLine(line)
			if head != "" {
				candidates = append(candidates, head)
				if len(candidates) >= maxCandidates {
					break
				}
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
	}
	return pickBestHead(candidates)
}

func conversationHeadFromLine(line []byte) string {
	var item responseItemLine
	if !jsontext.Value(line).IsValid() {
		return ""
	}
	if err := json.Unmarshal(line, &item); err != nil {
		return ""
	}
	if item.Type != "response_item" {
		return ""
	}
	if item.Payload.Type != "message" || item.Payload.Role != "user" {
		return ""
	}

	for _, c := range item.Payload.Content {
		if v := compactText(c.Text); v != "" {
			return v
		}
	}
	return compactText(item.Payload.Text)
}

func compactText(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	return strings.Join(strings.Fields(v), " ")
}

func pickBestHead(candidates []string) string {
	if len(candidates) == 0 {
		return ""
	}
	best := ""
	bestScore := -1
	firstUseful := ""
	for _, c := range candidates {
		if firstUseful == "" && !isLikelyHeadNoise(c) {
			firstUseful = c
		}
		score := scoreHeadCandidate(c)
		if score > bestScore {
			best = c
			bestScore = score
		}
	}
	if bestScore > 0 {
		return best
	}
	if firstUseful != "" {
		return firstUseful
	}
	return candidates[0]
}

func scoreHeadCandidate(v string) int {
	s := strings.TrimSpace(v)
	if s == "" {
		return 0
	}
	if isLikelyHeadNoise(s) {
		return 0
	}

	runes := utf8.RuneCountInString(s)
	score := 10
	switch {
	case runes >= 12 && runes <= 96:
		score += 45
	case runes >= 8 && runes <= 128:
		score += 25
	default:
		score += 5
	}

	lower := strings.ToLower(s)
	for _, kw := range []string{
		"fix", "add", "implement", "optimize", "refactor", "improve", "support",
		"实现", "增加", "优化", "修复", "支持", "改", "新增",
	} {
		if strings.Contains(lower, kw) {
			score += 15
			break
		}
	}

	if strings.Contains(s, "?") || strings.Contains(s, "？") {
		score += 8
	}
	return score
}

func isLikelyHeadNoise(v string) bool {
	lower := strings.ToLower(strings.TrimSpace(v))
	if lower == "" {
		return true
	}
	noiseMarkers := []string{
		"agents.md instructions",
		"<instructions>",
		"repository guidelines",
		"global agent rules",
		"## instruction hierarchy",
		"## tooling priorities",
		"## validation before finish",
		"you are codex",
		"you are an awaiter",
	}
	if slices.ContainsFunc(noiseMarkers, func(m string) bool { return strings.Contains(lower, m) }) {
		return true
	}
	return utf8.RuneCountInString(lower) > 200
}

func sessionIDFromFilename(base string) string {
	m := idInFilenameRe.FindStringSubmatch(strings.ToLower(base))
	if len(m) != 2 {
		return ""
	}
	return m[1]
}
