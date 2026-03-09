package session

import (
	"bufio"
	"errors"
	"io"
	"slices"
	"strings"
	"unicode/utf8"
)

func readConversationHead(r *bufio.Reader) string {
	const maxLines = 1024
	const maxCandidates = 24
	candidates := make([]string, 0, maxCandidates)
	for i := 0; i < maxLines; i++ {
		line, truncated, err := readBoundedLine(r, maxSessionHeadLineBytes)
		if err != nil && !errors.Is(err, io.EOF) {
			break
		}
		if !truncated && len(line) > 0 {
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
