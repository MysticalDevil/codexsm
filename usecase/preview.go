package usecase

import (
	"bufio"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"os"
	"strings"
)

const (
	DefaultPreviewMaxMessages = 600
	defaultPreviewMaxLineSize = 1024 * 1024
)

var ErrPreviewEntryTooLong = errors.New("preview entry exceeds max line size")

type PreviewMessage struct {
	Role string
	Text string
}

func ExtractPreviewMessages(path string, maxMessages int) ([]PreviewMessage, error) {
	if maxMessages <= 0 {
		maxMessages = DefaultPreviewMaxMessages
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	out := make([]PreviewMessage, 0, maxMessages)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), defaultPreviewMaxLineSize)
	for sc.Scan() && len(out) < maxMessages {
		role, text := ParsePreviewLine(sc.Text())
		if strings.TrimSpace(text) == "" || IsPreviewNoiseText(text) {
			continue
		}
		out = append(out, PreviewMessage{Role: role, Text: text})
	}
	if err := sc.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return out, ErrPreviewEntryTooLong
		}
		return out, err
	}
	return out, nil
}

func ParsePreviewLine(line string) (role string, text string) {
	var item struct {
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
	raw := []byte(strings.TrimSpace(line))
	if len(raw) == 0 || !jsontext.Value(raw).IsValid() {
		return "", ""
	}
	if err := json.Unmarshal(raw, &item); err != nil {
		return "", ""
	}
	if item.Type != "response_item" || item.Payload.Type != "message" {
		return "", ""
	}

	for _, c := range item.Payload.Content {
		if v := compactPreviewText(c.Text); v != "" {
			return item.Payload.Role, v
		}
	}
	return item.Payload.Role, compactPreviewText(item.Payload.Text)
}

func IsPreviewNoiseText(v string) bool {
	l := strings.ToLower(strings.TrimSpace(v))
	if l == "" {
		return true
	}
	noise := []string{
		"agents.md",
		"instructions for",
		"filesystem sandboxing",
		"approved command prefix",
		"global agent rules",
		"tooling priorities",
		"validation before finish",
		"use-modern-go",
	}
	for _, k := range noise {
		if strings.Contains(l, k) {
			return true
		}
	}
	return false
}

func compactPreviewText(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	return strings.Join(strings.Fields(v), " ")
}
