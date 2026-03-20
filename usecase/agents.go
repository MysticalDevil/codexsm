package usecase

import (
	"bufio"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"

	"github.com/MysticalDevil/codexsm/internal/core"
)

// AgentsExplainInput controls AGENTS.md explain behavior.
type AgentsExplainInput struct {
	CWD           string
	EffectiveOnly bool
	SourceFilter  string
	RuleFilter    string
}

type AgentsExplainFilters struct {
	EffectiveOnly bool   `json:"effective_only,omitempty"`
	SourceFilter  string `json:"source_filter,omitempty"`
	RuleFilter    string `json:"rule_filter,omitempty"`
}

// AgentsExplainSource is one discovered AGENTS.md source.
type AgentsExplainSource struct {
	Path     string `json:"path"`
	Priority int    `json:"priority"`
}

// AgentsExplainRule is one extracted rule line.
type AgentsExplainRule struct {
	ID         string `json:"id"`
	Key        string `json:"key"`
	Text       string `json:"text"`
	SourcePath string `json:"source_path"`
	Line       int    `json:"line"`
	Priority   int    `json:"priority"`
	Status     string `json:"status"`
	ShadowedBy string `json:"shadowed_by,omitempty"`
}

// AgentsExplainSummary reports aggregate counters.
type AgentsExplainSummary struct {
	Sources   int `json:"sources"`
	Rules     int `json:"rules"`
	Effective int `json:"effective"`
	Shadowed  int `json:"shadowed"`
}

// AgentsExplainResult is the normalized AGENTS.md explain output.
type AgentsExplainResult struct {
	CWD     string                `json:"cwd"`
	Sources []AgentsExplainSource `json:"sources"`
	Rules   []AgentsExplainRule   `json:"rules"`
	Filters AgentsExplainFilters  `json:"filters,omitempty"`
	Summary AgentsExplainSummary  `json:"summary"`
}

// ExplainAgents discovers AGENTS.md sources and computes effective/shadowed rules.
func ExplainAgents(in AgentsExplainInput) (AgentsExplainResult, error) {
	cwd, err := resolveAgentsCWD(in.CWD)
	if err != nil {
		return AgentsExplainResult{}, err
	}

	home, _ := os.UserHomeDir()

	sourcePaths, err := discoverAgentsSources(cwd, home)
	if err != nil {
		return AgentsExplainResult{}, err
	}

	result := AgentsExplainResult{
		CWD:     core.CompactHomePath(cwd, home),
		Sources: make([]AgentsExplainSource, 0, len(sourcePaths)),
		Rules:   make([]AgentsExplainRule, 0, 32),
		Filters: AgentsExplainFilters{
			EffectiveOnly: in.EffectiveOnly,
			SourceFilter:  strings.TrimSpace(in.SourceFilter),
			RuleFilter:    strings.TrimSpace(in.RuleFilter),
		},
	}
	for i, p := range sourcePaths {
		result.Sources = append(result.Sources, AgentsExplainSource{
			Path:     core.CompactHomePath(p, home),
			Priority: i,
		})
	}

	latestByKey := make(map[string]int, 64)

	for _, src := range result.Sources {
		rawPath := expandCompactPath(src.Path, home)

		lines, err := extractAgentsRules(rawPath)
		if err != nil {
			return AgentsExplainResult{}, err
		}

		for _, line := range lines {
			key := normalizeAgentsRuleKey(line.Text)
			if key == "" {
				continue
			}

			rule := AgentsExplainRule{
				ID:         agentsRuleID(src.Path, line.Number, line.Text),
				Key:        key,
				Text:       line.Text,
				SourcePath: src.Path,
				Line:       line.Number,
				Priority:   src.Priority,
				Status:     "effective",
			}
			if prev, ok := latestByKey[key]; ok {
				result.Rules[prev].Status = "shadowed"
				result.Rules[prev].ShadowedBy = fmt.Sprintf("%s:%d", src.Path, line.Number)
			}

			result.Rules = append(result.Rules, rule)
			latestByKey[key] = len(result.Rules) - 1
		}
	}

	result.Rules = filterAgentsRules(result.Rules, result.Filters)
	computeAgentsSummary(&result)

	return result, nil
}

func computeAgentsSummary(result *AgentsExplainResult) {
	result.Summary.Sources = len(result.Sources)
	result.Summary.Rules = len(result.Rules)
	result.Summary.Effective = 0
	result.Summary.Shadowed = 0

	for _, r := range result.Rules {
		if r.Status == "effective" {
			result.Summary.Effective++
		} else {
			result.Summary.Shadowed++
		}
	}
}

func filterAgentsRules(rules []AgentsExplainRule, filters AgentsExplainFilters) []AgentsExplainRule {
	sourceFilter := strings.ToLower(strings.TrimSpace(filters.SourceFilter))
	ruleFilter := strings.ToLower(strings.TrimSpace(filters.RuleFilter))

	if !filters.EffectiveOnly && sourceFilter == "" && ruleFilter == "" {
		return rules
	}

	out := make([]AgentsExplainRule, 0, len(rules))
	for _, r := range rules {
		if filters.EffectiveOnly && r.Status != "effective" {
			continue
		}

		if sourceFilter != "" {
			p := strings.ToLower(r.SourcePath)
			if !strings.Contains(p, sourceFilter) {
				continue
			}
		}

		if ruleFilter != "" {
			k := strings.ToLower(r.Key)

			t := strings.ToLower(r.Text)
			if !strings.Contains(k, ruleFilter) && !strings.Contains(t, ruleFilter) {
				continue
			}
		}

		out = append(out, r)
	}

	return out
}

type agentsRuleLine struct {
	Number int
	Text   string
}

func resolveAgentsCWD(cwd string) (string, error) {
	if strings.TrimSpace(cwd) == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}

		cwd = wd
	}

	abs, err := filepath.Abs(cwd)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}

	if !info.IsDir() {
		abs = filepath.Dir(abs)
	}

	return filepath.Clean(abs), nil
}

func discoverAgentsSources(cwd, home string) ([]string, error) {
	var out []string

	seen := make(map[string]struct{}, 8)
	appendIfExists := func(p string) error {
		if p == "" {
			return nil
		}

		if _, ok := seen[p]; ok {
			return nil
		}

		st, err := os.Stat(p)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}

			return err
		}

		if st.IsDir() {
			return nil
		}

		seen[p] = struct{}{}
		out = append(out, p)

		return nil
	}

	if home != "" {
		if err := appendIfExists(filepath.Join(home, ".codex", "AGENTS.md")); err != nil {
			return nil, err
		}
	}

	ancestors := make([]string, 0, 8)

	cur := cwd
	for {
		ancestors = append(ancestors, cur)

		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}

		cur = parent
	}

	for i := len(ancestors) - 1; i >= 0; i-- {
		if err := appendIfExists(filepath.Join(ancestors[i], "AGENTS.md")); err != nil {
			return nil, err
		}
	}

	return out, nil
}

func extractAgentsRules(path string) ([]agentsRuleLine, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer func() { _ = f.Close() }()

	out := make([]agentsRuleLine, 0, 32)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	inCodeFence := false

	lineNum := 0
	for sc.Scan() {
		lineNum++

		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "```") {
			inCodeFence = !inCodeFence
			continue
		}

		if inCodeFence {
			continue
		}

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		out = append(out, agentsRuleLine{
			Number: lineNum,
			Text:   line,
		})
	}

	if err := sc.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func normalizeAgentsRuleKey(v string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(v)), " "))
}

func agentsRuleID(path string, line int, text string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(path))
	_, _ = h.Write([]byte{0})
	_, _ = fmt.Fprintf(h, "%d", line)
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(text))

	return fmt.Sprintf("%x", h.Sum64())
}

func expandCompactPath(p, home string) string {
	if p == "~" && home != "" {
		return home
	}

	prefix := "~" + string(os.PathSeparator)
	if strings.HasPrefix(p, prefix) && home != "" {
		return filepath.Join(home, strings.TrimPrefix(p, prefix))
	}

	return p
}
