package doctor_test

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/cli"
	cliutil "github.com/MysticalDevil/codexsm/cli/util"
	"github.com/MysticalDevil/codexsm/internal/testsupport"
	"github.com/MysticalDevil/codexsm/usecase"
)

func TestDoctorCommandNonStrict(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	sessionsRoot := filepath.Join(workspace, "sessions")
	t.Setenv("SESSIONS_ROOT", sessionsRoot)
	t.Setenv("CSM_CONFIG", filepath.Join(workspace, "missing-config.json"))

	cmd := cli.NewRootCmd()
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

	cmd := cli.NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"doctor", "--strict"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected strict doctor failure")
	}
}

func TestDoctorRiskCommandReturnsFailureWhenRiskFound(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "sessions")
	cmd := cli.NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"doctor", "risk", "--sessions-root", root, "--sample-limit", "5"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected risk command to fail when risky sessions exist")
	}

	var ex *cliutil.ExitError
	if !errors.As(err, &ex) || ex.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got err=%v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "RISK SUMMARY") || !strings.Contains(out, "risk_total=") {
		t.Fatalf("unexpected risk output: %q", out)
	}
}

func TestDoctorRiskCommandPassesWhenNoRiskFound(t *testing.T) {
	sessionsRoot := t.TempDir()
	writeDoctorSessionFixture(t, sessionsRoot, "ok1", t.TempDir())
	writeDoctorSessionFixture(t, sessionsRoot, "ok2", t.TempDir())

	cmd := cli.NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"doctor", "risk", "--sessions-root", sessionsRoot})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("doctor risk execute: %v", err)
	}

	if !strings.Contains(stdout.String(), "no risky sessions found") {
		t.Fatalf("expected no-risk output, got: %q", stdout.String())
	}
}

func TestDoctorRiskCommandJSONFormat(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "sessions")
	cmd := cli.NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"doctor", "risk", "--sessions-root", root, "--format", "json", "--sample-limit", "2"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected non-zero when risky sessions exist")
	}

	var payload struct {
		SessionsTotal int `json:"sessions_total"`
		RiskTotal     int `json:"risk_total"`
		SampleLimit   int `json:"sample_limit"`
		Samples       []struct {
			Level string `json:"level"`
			Path  string `json:"path"`
		} `json:"samples"`
	}
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &payload); unmarshalErr != nil {
		t.Fatalf("json unmarshal: %v output=%q", unmarshalErr, stdout.String())
	}

	if payload.SessionsTotal == 0 || payload.RiskTotal == 0 {
		t.Fatalf("unexpected json payload: %+v", payload)
	}

	if payload.SampleLimit != 2 {
		t.Fatalf("expected sample limit=2, got %+v", payload)
	}
}

func TestDoctorRiskCommandIntegrityMismatchIsRisk(t *testing.T) {
	sessionsRoot := t.TempDir()
	host := t.TempDir()
	writeDoctorSessionFixture(t, sessionsRoot, "oksha", host)

	p := filepath.Join(sessionsRoot, "2026", "03", "08", "oksha.jsonl")
	if err := os.WriteFile(p+".sha256", []byte(strings.Repeat("0", 64)+"  oksha.jsonl\n"), 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}

	cmd := cli.NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"doctor", "risk", "--sessions-root", sessionsRoot, "--format", "json", "--integrity-check=true"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected risk detection")
	}

	var payload struct {
		RiskTotal int `json:"risk_total"`
		High      int `json:"high"`
	}
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &payload); unmarshalErr != nil {
		t.Fatalf("json unmarshal: %v output=%q", unmarshalErr, stdout.String())
	}

	if payload.RiskTotal == 0 || payload.High == 0 {
		t.Fatalf("expected high risk from integrity mismatch, got %+v", payload)
	}
}

func TestDoctorRiskCommandExtremeStaticJSON(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "extreme-static")
	root := filepath.Join(workspace, "sessions")

	cmd := cli.NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"doctor", "risk", "--sessions-root", root, "--format", "json", "--sample-limit", "4"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected non-zero when risky sessions exist")
	}

	var payload struct {
		SessionsTotal int `json:"sessions_total"`
		RiskTotal     int `json:"risk_total"`
		SampleLimit   int `json:"sample_limit"`
	}
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &payload); unmarshalErr != nil {
		t.Fatalf("json unmarshal: %v output=%q", unmarshalErr, stdout.String())
	}

	if payload.SessionsTotal != 6 {
		t.Fatalf("expected 6 sessions, got %+v", payload)
	}

	if payload.RiskTotal == 0 {
		t.Fatalf("expected non-zero risk total, got %+v", payload)
	}

	if payload.SampleLimit != 4 {
		t.Fatalf("expected sample limit=4, got %+v", payload)
	}
}

func TestCheckSessionHostPathsWarnsWhenHostMissing(t *testing.T) {
	sessionsRoot := t.TempDir()
	existingHost := t.TempDir()
	missingHost := filepath.Join(t.TempDir(), "missing-host-dir")

	writeDoctorSessionFixture(t, sessionsRoot, "s1", existingHost)
	writeDoctorSessionFixture(t, sessionsRoot, "s2", missingHost)

	got := usecase.CheckSessionHostPaths(usecase.DoctorHostPathInput{
		SessionsRoot: sessionsRoot,
		SessionsErr:  nil,
		CompactPath:  compactDoctorPathForTest,
	})
	if got.Level != usecase.DoctorWarn {
		t.Fatalf("expected warn, got %s detail=%q", got.Level, got.Detail)
	}

	if !strings.Contains(got.Detail, "recommended_actions:") {
		t.Fatalf("expected action block in detail, got: %q", got.Detail)
	}

	if !strings.Contains(got.Detail, "migrate (soft-delete): codexsm delete --host-contains") {
		t.Fatalf("expected delete suggestion in detail, got: %q", got.Detail)
	}
}

func TestCheckSessionHostPathsPassWhenAllHostsExist(t *testing.T) {
	sessionsRoot := t.TempDir()
	hostA := t.TempDir()
	hostB := t.TempDir()

	writeDoctorSessionFixture(t, sessionsRoot, "s1", hostA)
	writeDoctorSessionFixture(t, sessionsRoot, "s2", hostB)

	got := usecase.CheckSessionHostPaths(usecase.DoctorHostPathInput{
		SessionsRoot: sessionsRoot,
		SessionsErr:  nil,
		CompactPath:  compactDoctorPathForTest,
	})
	if got.Level != usecase.DoctorPass {
		t.Fatalf("expected pass, got %s detail=%q", got.Level, got.Detail)
	}

	if !strings.Contains(got.Detail, "all host paths exist") {
		t.Fatalf("unexpected pass detail: %q", got.Detail)
	}
}

func writeDoctorSessionFixture(t *testing.T, sessionsRoot, id, host string) {
	t.Helper()

	dir := filepath.Join(sessionsRoot, "2026", "03", "08")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir sessions fixture: %v", err)
	}

	path := filepath.Join(dir, id+".jsonl")

	line := fmt.Sprintf(
		`{"type":"session_meta","payload":{"id":"%s","cwd":"%s","timestamp":"%s"}}`+"\n",
		id,
		host,
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err := os.WriteFile(path, []byte(line), 0o644); err != nil {
		t.Fatalf("write session fixture: %v", err)
	}
}

func compactDoctorPathForTest(path string, maxLen int) string {
	p := strings.TrimSpace(path)
	if p == "" || maxLen <= 0 || len(p) <= maxLen {
		return p
	}

	return p[:maxLen]
}
