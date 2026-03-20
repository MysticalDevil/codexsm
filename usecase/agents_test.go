package usecase

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExplainAgentsLayeringAndShadowing(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatalf("mkdir home codex: %v", err)
	}

	t.Setenv("HOME", home)

	repo := filepath.Join(t.TempDir(), "repo")

	cwd := filepath.Join(repo, "sub", "deep")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir repo cwd: %v", err)
	}

	globalText := strings.Join([]string{
		"# Global",
		"Prefer rg for text search.",
		"```md",
		"inside code fence should be ignored",
		"```",
	}, "\n") + "\n"
	rootText := strings.Join([]string{
		"# Repo",
		"Prefer rg for text search.",
		"Use go test ./... before release.",
	}, "\n") + "\n"
	subText := strings.Join([]string{
		"# Sub",
		"Prefer rg for text search.",
		"Use ast-grep for structural queries.",
	}, "\n") + "\n"

	if err := os.WriteFile(filepath.Join(home, ".codex", "AGENTS.md"), []byte(globalText), 0o644); err != nil {
		t.Fatalf("write global AGENTS: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte(rootText), 0o644); err != nil {
		t.Fatalf("write root AGENTS: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, "sub", "AGENTS.md"), []byte(subText), 0o644); err != nil {
		t.Fatalf("write sub AGENTS: %v", err)
	}

	got, err := ExplainAgents(AgentsExplainInput{CWD: cwd})
	if err != nil {
		t.Fatalf("ExplainAgents: %v", err)
	}

	if got.Summary.Sources != 3 {
		t.Fatalf("sources=%d, want 3", got.Summary.Sources)
	}

	if len(got.Sources) != 3 {
		t.Fatalf("len(sources)=%d, want 3", len(got.Sources))
	}

	if got.Sources[0].Priority != 0 || got.Sources[1].Priority != 1 || got.Sources[2].Priority != 2 {
		t.Fatalf("unexpected priorities: %+v", got.Sources)
	}

	if !strings.HasPrefix(got.Sources[0].Path, "~/.codex/") {
		t.Fatalf("expected compact global path, got %q", got.Sources[0].Path)
	}

	var (
		rgEffective   int
		rgShadowed    int
		foundCodeText bool
	)

	for _, rule := range got.Rules {
		if strings.Contains(rule.Text, "inside code fence") {
			foundCodeText = true
		}

		if rule.Key == "prefer rg for text search." {
			switch rule.Status {
			case "effective":
				rgEffective++

				if rule.Priority != 2 {
					t.Fatalf("effective rg rule priority=%d, want 2", rule.Priority)
				}
			case "shadowed":
				rgShadowed++

				if strings.TrimSpace(rule.ShadowedBy) == "" {
					t.Fatalf("shadowed rule should include shadowed_by: %+v", rule)
				}
			}
		}
	}

	if foundCodeText {
		t.Fatalf("code-fence line should be ignored")
	}

	if rgEffective != 1 || rgShadowed != 2 {
		t.Fatalf("rg rule effective/shadowed mismatch: effective=%d shadowed=%d", rgEffective, rgShadowed)
	}

	if got.Summary.Effective == 0 {
		t.Fatalf("expected effective rules > 0")
	}

	if got.Summary.Shadowed == 0 {
		t.Fatalf("expected shadowed rules > 0")
	}
}

func TestExplainAgentsNoSources(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	t.Setenv("HOME", home)

	cwd := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir cwd: %v", err)
	}

	got, err := ExplainAgents(AgentsExplainInput{CWD: cwd})
	if err != nil {
		t.Fatalf("ExplainAgents: %v", err)
	}

	if got.Summary.Sources != 0 || got.Summary.Rules != 0 {
		t.Fatalf("expected empty result, got summary=%+v", got.Summary)
	}
}

func TestExplainAgentsFilters(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatalf("mkdir home codex: %v", err)
	}

	t.Setenv("HOME", home)

	repo := filepath.Join(t.TempDir(), "repo")

	cwd := filepath.Join(repo, "sub")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir cwd: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("Prefer rg for text search.\nUse go test ./... before release.\n"), 0o644); err != nil {
		t.Fatalf("write repo agents: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, "sub", "AGENTS.md"), []byte("Prefer rg for text search.\nUse ast-grep for structural queries.\n"), 0o644); err != nil {
		t.Fatalf("write sub agents: %v", err)
	}

	got, err := ExplainAgents(AgentsExplainInput{
		CWD:           cwd,
		EffectiveOnly: true,
		SourceFilter:  "sub/AGENTS.md",
		RuleFilter:    "ast-grep",
	})
	if err != nil {
		t.Fatalf("ExplainAgents: %v", err)
	}

	if got.Summary.Rules != 1 || got.Summary.Effective != 1 || got.Summary.Shadowed != 0 {
		t.Fatalf("unexpected summary: %+v", got.Summary)
	}

	if len(got.Rules) != 1 {
		t.Fatalf("expected one filtered rule, got %d", len(got.Rules))
	}

	if !strings.Contains(strings.ToLower(got.Rules[0].Text), "ast-grep") {
		t.Fatalf("unexpected filtered rule: %+v", got.Rules[0])
	}
}
