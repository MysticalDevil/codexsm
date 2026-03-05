//go:build integration
// +build integration

package cli

import (
	"bytes"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
)

func TestConfigShowMissingFile(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, fixtureName)
	cfgPath := filepath.Join(workspace, "config", "missing.json")
	t.Setenv("CSM_CONFIG", cfgPath)
	t.Setenv("SESSIONS_ROOT", "")

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"config", "show"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("config show execute: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("config show invalid json: %v\n%s", err, stdout.String())
	}
	if out["path"] != cfgPath {
		t.Fatalf("unexpected path: %+v", out["path"])
	}
	exists, ok := out["exists"].(bool)
	if !ok || exists {
		t.Fatalf("expected exists=false, got: %+v", out["exists"])
	}
}

func TestConfigInitShowValidate(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, fixtureName)
	cfgPath := filepath.Join(workspace, "config", "config.json")
	t.Setenv("CSM_CONFIG", cfgPath)
	t.Setenv("SESSIONS_ROOT", "")

	// dry-run does not write.
	dryCmd := NewRootCmd()
	dryOut := &bytes.Buffer{}
	dryErr := &bytes.Buffer{}
	dryCmd.SetOut(dryOut)
	dryCmd.SetErr(dryErr)
	dryCmd.SetArgs([]string{"config", "init", "--dry-run"})
	if err := dryCmd.Execute(); err != nil {
		t.Fatalf("config init --dry-run execute: %v", err)
	}
	if !strings.Contains(dryErr.String(), "dry-run: would write") {
		t.Fatalf("unexpected dry-run stderr: %q", dryErr.String())
	}
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Fatalf("dry-run should not create file, stat err: %v", err)
	}

	// init writes file.
	initCmd := NewRootCmd()
	initOut := &bytes.Buffer{}
	initErr := &bytes.Buffer{}
	initCmd.SetOut(initOut)
	initCmd.SetErr(initErr)
	initCmd.SetArgs([]string{"config", "init"})
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("config init execute: %v", err)
	}
	if !strings.Contains(initOut.String(), "initialized config:") {
		t.Fatalf("unexpected init output: %q", initOut.String())
	}
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("expected config file created: %v", err)
	}

	// show resolved includes effective runtime values.
	showCmd := NewRootCmd()
	showOut := &bytes.Buffer{}
	showErr := &bytes.Buffer{}
	showCmd.SetOut(showOut)
	showCmd.SetErr(showErr)
	showCmd.SetArgs([]string{"config", "show", "--resolved"})
	if err := showCmd.Execute(); err != nil {
		t.Fatalf("config show --resolved execute: %v", err)
	}
	var showDoc map[string]any
	if err := json.Unmarshal(showOut.Bytes(), &showDoc); err != nil {
		t.Fatalf("config show invalid json: %v\n%s", err, showOut.String())
	}
	if showDoc["path"] != cfgPath {
		t.Fatalf("unexpected path: %+v", showDoc["path"])
	}
	if _, ok := showDoc["effective"].(map[string]any); !ok {
		t.Fatalf("expected effective block, got: %+v", showDoc["effective"])
	}

	// validate succeeds for generated config.
	validateCmd := NewRootCmd()
	validateOut := &bytes.Buffer{}
	validateErr := &bytes.Buffer{}
	validateCmd.SetOut(validateOut)
	validateCmd.SetErr(validateErr)
	validateCmd.SetArgs([]string{"config", "validate"})
	if err := validateCmd.Execute(); err != nil {
		t.Fatalf("config validate execute: %v", err)
	}
	if !strings.Contains(validateOut.String(), "valid:") {
		t.Fatalf("unexpected validate output: %q", validateOut.String())
	}
}

func TestConfigValidateFailsForInvalidJSON(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, fixtureName)
	cfgPath := filepath.Join(workspace, "config", "invalid.json")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(cfgPath, []byte(`{"sessions_root"`), 0o644); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}
	t.Setenv("CSM_CONFIG", cfgPath)

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"config", "validate"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validate parse error")
	}
	if !strings.Contains(err.Error(), "parse config") {
		t.Fatalf("unexpected validate error: %v", err)
	}
}
