package group

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
)

func TestNewCommandCSVWritesDelimitedOutput(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	sessionsRoot := filepath.Join(workspace, "sessions")

	cmd := NewCommand(func() (string, error) { return sessionsRoot, nil })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--by", "day", "--format", "csv"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("group csv execute: %v stderr=%q", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "group,count,size_bytes,latest") {
		t.Fatalf("expected csv header, got: %q", stdout.String())
	}
}

func TestNewCommandJSONWritesStatsArray(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	sessionsRoot := filepath.Join(workspace, "sessions")

	cmd := NewCommand(func() (string, error) { return sessionsRoot, nil })

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--by", "health", "--format", "json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("group json execute: %v stderr=%q", err, stderr.String())
	}

	text := strings.TrimSpace(stdout.String())
	if !strings.HasPrefix(text, "[") {
		t.Fatalf("expected json array, got: %q", text)
	}
}
