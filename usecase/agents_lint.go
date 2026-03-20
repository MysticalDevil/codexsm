package usecase

import (
	"fmt"
)

type AgentsLintInput struct {
	CWD string
}

type AgentsLintIssue struct {
	Level      string `json:"level"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Key        string `json:"key,omitempty"`
	SourcePath string `json:"source_path,omitempty"`
	Line       int    `json:"line,omitempty"`
}

type AgentsLintSummary struct {
	Sources  int `json:"sources"`
	Rules    int `json:"rules"`
	Warnings int `json:"warnings"`
	Errors   int `json:"errors"`
}

type AgentsLintResult struct {
	CWD     string            `json:"cwd"`
	Issues  []AgentsLintIssue `json:"issues"`
	Summary AgentsLintSummary `json:"summary"`
}

func LintAgents(in AgentsLintInput) (AgentsLintResult, error) {
	explain, err := ExplainAgents(AgentsExplainInput{CWD: in.CWD})
	if err != nil {
		return AgentsLintResult{}, err
	}

	result := AgentsLintResult{
		CWD:    explain.CWD,
		Issues: make([]AgentsLintIssue, 0, len(explain.Rules)),
		Summary: AgentsLintSummary{
			Sources: explain.Summary.Sources,
			Rules:   explain.Summary.Rules,
		},
	}

	if len(explain.Sources) == 0 {
		result.Issues = append(result.Issues, AgentsLintIssue{
			Level:   "warning",
			Code:    "no_sources",
			Message: "no AGENTS.md sources discovered",
		})
	}

	seenInSource := make(map[string]AgentsLintIssue, len(explain.Rules))
	for _, r := range explain.Rules {
		if r.Status == "shadowed" {
			result.Issues = append(result.Issues, AgentsLintIssue{
				Level:      "warning",
				Code:       "shadowed_rule",
				Message:    fmt.Sprintf("rule is shadowed by %s", r.ShadowedBy),
				Key:        r.Key,
				SourcePath: r.SourcePath,
				Line:       r.Line,
			})
		}

		dupKey := r.SourcePath + "\x00" + r.Key
		if prev, ok := seenInSource[dupKey]; ok {
			result.Issues = append(result.Issues, AgentsLintIssue{
				Level:      "warning",
				Code:       "duplicate_rule_in_source",
				Message:    fmt.Sprintf("duplicate normalized rule in same source (first at %s:%d)", prev.SourcePath, prev.Line),
				Key:        r.Key,
				SourcePath: r.SourcePath,
				Line:       r.Line,
			})

			continue
		}

		seenInSource[dupKey] = AgentsLintIssue{SourcePath: r.SourcePath, Line: r.Line}
	}

	for _, issue := range result.Issues {
		switch issue.Level {
		case "error":
			result.Summary.Errors++
		default:
			result.Summary.Warnings++
		}
	}

	return result, nil
}
