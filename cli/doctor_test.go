package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
)

func TestDoctorCommandNonStrict(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	sessionsRoot := filepath.Join(workspace, "sessions")
	t.Setenv("SESSIONS_ROOT", sessionsRoot)
	t.Setenv("CSM_CONFIG", filepath.Join(workspace, "missing-config.json"))

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"doctor"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("doctor execute: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "CHECK") || !strings.Contains(out, "sessions_root") {
		t.Fatalf("unexpected doctor output: %q", out)
	}
}

func TestDoctorCommandStrictFailsOnWarn(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	sessionsRoot := filepath.Join(workspace, "sessions")
	t.Setenv("SESSIONS_ROOT", sessionsRoot)
	t.Setenv("CSM_CONFIG", filepath.Join(workspace, "missing-config.json"))

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"doctor", "--strict"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected strict doctor failure")
	}
}
