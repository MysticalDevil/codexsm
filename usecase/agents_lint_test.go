package usecase

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLintAgentsFindsShadowedAndDuplicates(t *testing.T) {
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

	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("Prefer rg for text search.\nPrefer rg for text search.\n"), 0o644); err != nil {
		t.Fatalf("write repo AGENTS: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, "sub", "AGENTS.md"), []byte("Prefer rg for text search.\n"), 0o644); err != nil {
		t.Fatalf("write sub AGENTS: %v", err)
	}

	out, err := LintAgents(AgentsLintInput{CWD: cwd})
	if err != nil {
		t.Fatalf("LintAgents: %v", err)
	}

	if out.Summary.Warnings < 2 {
		t.Fatalf("expected warnings >= 2, got %d (%+v)", out.Summary.Warnings, out.Issues)
	}

	var hasShadowed, hasDuplicate bool

	for _, issue := range out.Issues {
		if issue.Code == "shadowed_rule" {
			hasShadowed = true
		}

		if issue.Code == "duplicate_rule_in_source" {
			hasDuplicate = true
		}
	}

	if !hasShadowed {
		t.Fatal("expected shadowed_rule issue")
	}

	if !hasDuplicate {
		t.Fatal("expected duplicate_rule_in_source issue")
	}
}

func TestLintAgentsNoSources(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	t.Setenv("HOME", home)

	cwd := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir cwd: %v", err)
	}

	out, err := LintAgents(AgentsLintInput{CWD: cwd})
	if err != nil {
		t.Fatalf("LintAgents: %v", err)
	}

	if out.Summary.Warnings == 0 {
		t.Fatalf("expected warning for no sources: %+v", out)
	}
}
