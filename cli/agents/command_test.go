package agents

import (
	"bytes"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MysticalDevil/codexsm/usecase"
)

func TestInvalidFormat(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "explain", args: []string{"explain", "--format", "yaml"}},
		{name: "lint", args: []string{"lint", "--format", "yaml"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewCommand()
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			cmd.SetOut(stdout)
			cmd.SetErr(stderr)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected invalid format error")
			}

			if !strings.Contains(err.Error(), "invalid --format") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLintJSONOutput(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")

	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatalf("mkdir home codex: %v", err)
	}

	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	t.Setenv("HOME", home)

	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("Prefer rg for text search.\nPrefer rg for text search.\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS: %v", err)
	}

	cmd := NewCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"lint", "--cwd", repo, "--format", "json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents lint json execute: %v", err)
	}

	var out struct {
		Summary struct {
			Sources  int `json:"sources"`
			Rules    int `json:"rules"`
			Warnings int `json:"warnings"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("decode json output: %v, output=%q", err, stdout.String())
	}

	if out.Summary.Sources != 1 {
		t.Fatalf("sources=%d, want 1", out.Summary.Sources)
	}

	if out.Summary.Rules == 0 || out.Summary.Warnings == 0 {
		t.Fatalf("unexpected summary: %+v", out.Summary)
	}
}

func TestRenderHelpers(t *testing.T) {
	explainText := renderExplainTable(usecase.AgentsExplainResult{}, false)
	if !strings.Contains(explainText, "no AGENTS.md sources discovered") {
		t.Fatalf("unexpected explain table: %q", explainText)
	}

	lintText := renderLintTable(usecase.AgentsLintResult{})
	if !strings.Contains(lintText, "no issues") {
		t.Fatalf("unexpected lint table: %q", lintText)
	}
}
