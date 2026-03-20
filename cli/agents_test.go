package cli

import (
	"bytes"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentsExplainTableAndShadowed(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	repo := filepath.Join(t.TempDir(), "repo")

	cwd := filepath.Join(repo, "sub")
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatalf("mkdir home codex: %v", err)
	}

	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir cwd: %v", err)
	}

	t.Setenv("HOME", home)

	if err := os.WriteFile(filepath.Join(home, ".codex", "AGENTS.md"), []byte("Prefer rg.\n"), 0o644); err != nil {
		t.Fatalf("write global AGENTS: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("Prefer rg.\n"), 0o644); err != nil {
		t.Fatalf("write repo AGENTS: %v", err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"agents", "explain", "--cwd", cwd})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents explain table execute: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "EFFECTIVE RULES") {
		t.Fatalf("missing effective rules section: %q", out)
	}

	if strings.Contains(out, "SHADOWED RULES") {
		t.Fatalf("did not expect shadowed section by default: %q", out)
	}

	cmd = NewRootCmd()

	stdout.Reset()
	stderr.Reset()
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"agents", "explain", "--cwd", cwd, "--show-shadowed"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents explain --show-shadowed execute: %v", err)
	}

	out = stdout.String()
	if !strings.Contains(out, "SHADOWED RULES") {
		t.Fatalf("expected shadowed section: %q", out)
	}
}

func TestAgentsExplainJSON(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")

	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatalf("mkdir home codex: %v", err)
	}

	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	t.Setenv("HOME", home)

	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("Use ast-grep.\n"), 0o644); err != nil {
		t.Fatalf("write repo AGENTS: %v", err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"agents", "explain", "--cwd", repo, "--format", "json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents explain json execute: %v", err)
	}

	var decoded struct {
		CWD     string `json:"cwd"`
		Summary struct {
			Sources   int `json:"sources"`
			Rules     int `json:"rules"`
			Effective int `json:"effective"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("decode json output: %v, output=%q", err, stdout.String())
	}

	if decoded.Summary.Sources != 1 {
		t.Fatalf("sources=%d, want 1", decoded.Summary.Sources)
	}

	if decoded.Summary.Rules == 0 || decoded.Summary.Effective == 0 {
		t.Fatalf("unexpected summary: %+v", decoded.Summary)
	}
}

func TestAgentsExplainWithFilters(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	repo := filepath.Join(t.TempDir(), "repo")

	cwd := filepath.Join(repo, "sub")
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatalf("mkdir home codex: %v", err)
	}

	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir cwd: %v", err)
	}

	t.Setenv("HOME", home)

	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("Prefer rg for text search.\n"), 0o644); err != nil {
		t.Fatalf("write repo AGENTS: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, "sub", "AGENTS.md"), []byte("Use ast-grep for structural queries.\n"), 0o644); err != nil {
		t.Fatalf("write sub AGENTS: %v", err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"agents", "explain", "--cwd", cwd, "--effective-only", "--source", "sub", "--rule", "ast-grep"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents explain filtered execute: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "filters:") {
		t.Fatalf("expected filters line in output: %q", out)
	}

	if !strings.Contains(strings.ToLower(out), "ast-grep") {
		t.Fatalf("expected filtered ast-grep rule: %q", out)
	}
}

func TestAgentsLintTableAndStrict(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	repo := filepath.Join(t.TempDir(), "repo")

	cwd := filepath.Join(repo, "sub")
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatalf("mkdir home codex: %v", err)
	}

	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir cwd: %v", err)
	}

	t.Setenv("HOME", home)

	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("Prefer rg for text search.\nPrefer rg for text search.\n"), 0o644); err != nil {
		t.Fatalf("write repo AGENTS: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, "sub", "AGENTS.md"), []byte("Prefer rg for text search.\n"), 0o644); err != nil {
		t.Fatalf("write sub AGENTS: %v", err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"agents", "lint", "--cwd", cwd})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents lint execute: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "ISSUES") {
		t.Fatalf("expected issues output: %q", out)
	}

	cmd = NewRootCmd()

	stdout.Reset()
	stderr.Reset()
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"agents", "lint", "--cwd", cwd, "--strict"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected strict lint to fail with warnings")
	}
}
