package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("version execute: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "codexsm "+Version) {
		t.Fatalf("unexpected version output: %q", out)
	}
}

func TestVersionCommandShort(t *testing.T) {
	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"version", "--short"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("version --short execute: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != Version {
		t.Fatalf("unexpected short version output: %q", got)
	}
}
