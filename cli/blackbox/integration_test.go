//go:build integration
// +build integration

package blackbox

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
)

const fixtureName = "rich"

var (
	buildOnce sync.Once
	binPath   string
	buildErr  error
)

type runResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Err      error
}

func TestListExitCode(t *testing.T) {
	_, sessionsRoot, _, _ := fixtureRoots(t)

	res := runCLI(t, []string{"list", "--sessions-root", sessionsRoot, "--limit", "1", "--color", "never"}, nil)
	if res.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q err=%v", res.ExitCode, res.Stderr, res.Err)
	}

	if !strings.Contains(res.Stdout, "ID") {
		t.Fatalf("expected list output header, got: %q", res.Stdout)
	}
}

func TestAgentsLintStrictExitCode(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	repo := filepath.Join(root, "repo")
	cwd := filepath.Join(repo, "sub")

	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatalf("mkdir home codex: %v", err)
	}

	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir cwd: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("Prefer rg for text search.\nPrefer rg for text search.\n"), 0o644); err != nil {
		t.Fatalf("write repo agents: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, "sub", "AGENTS.md"), []byte("Prefer rg for text search.\n"), 0o644); err != nil {
		t.Fatalf("write sub agents: %v", err)
	}

	res := runCLI(t, []string{"agents", "lint", "--cwd", cwd, "--strict"}, []string{"HOME=" + home})
	if res.ExitCode != 1 {
		t.Fatalf("expected exit code 1 for strict lint warnings, got %d, stderr=%q err=%v", res.ExitCode, res.Stderr, res.Err)
	}

	if !strings.Contains(res.Stderr, "strict mode") {
		t.Fatalf("expected strict mode failure in stderr, got: %q", res.Stderr)
	}
}

func TestRestoreConflictExitCode(t *testing.T) {
	_, sessionsRoot, trashRoot, logFile := fixtureRoots(t)

	res := runCLI(t, []string{
		"restore",
		"--sessions-root", sessionsRoot,
		"--trash-root", trashRoot,
		"--log-file", logFile,
		"--batch-id", "b-test",
		"--id", "deadbeef",
	}, nil)
	if res.ExitCode != 1 {
		t.Fatalf("expected exit code 1 for conflicting selector flags, got %d stderr=%q err=%v", res.ExitCode, res.Stderr, res.Err)
	}

	if !strings.Contains(res.Stderr, "cannot be combined") {
		t.Fatalf("expected conflict hint in stderr, got %q", res.Stderr)
	}
}

func TestErrorWritesToStderrOnly(t *testing.T) {
	_, sessionsRoot, trashRoot, logFile := fixtureRoots(t)

	res := runCLI(t, []string{
		"restore",
		"--sessions-root", sessionsRoot,
		"--trash-root", trashRoot,
		"--log-file", logFile,
		"--batch-id", "b-test",
		"--id", "deadbeef",
	}, nil)

	if res.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d stderr=%q err=%v", res.ExitCode, res.Stderr, res.Err)
	}

	if strings.TrimSpace(res.Stdout) != "" {
		t.Fatalf("expected stdout to be empty on failure, got %q", res.Stdout)
	}

	if strings.TrimSpace(res.Stderr) == "" {
		t.Fatalf("expected stderr to contain error details")
	}
}

func TestSuccessKeepsStderrEmpty(t *testing.T) {
	_, sessionsRoot, _, _ := fixtureRoots(t)

	res := runCLI(t, []string{"list", "--sessions-root", sessionsRoot, "--limit", "1", "--color", "never"}, nil)
	if res.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q err=%v", res.ExitCode, res.Stderr, res.Err)
	}

	if strings.TrimSpace(res.Stderr) != "" {
		t.Fatalf("expected stderr to be empty on success, got %q", res.Stderr)
	}
}

func TestConfigSessionsRootUsedWhenFlagMissing(t *testing.T) {
	workspace, sessionsRoot, _, _ := fixtureRoots(t)

	configFile := filepath.Join(workspace, "config.json")
	configBody := []byte(`{"sessions_root":"` + sessionsRoot + `"}`)
	if err := os.WriteFile(configFile, configBody, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	res := runCLI(t, []string{"list", "--limit", "1", "--color", "never"}, []string{"CSM_CONFIG=" + configFile})
	if res.ExitCode != 0 {
		t.Fatalf("expected exit code 0 with config sessions_root, got %d stderr=%q err=%v", res.ExitCode, res.Stderr, res.Err)
	}

	if !strings.Contains(res.Stdout, "ID") {
		t.Fatalf("expected list output header, got: %q", res.Stdout)
	}
}

func TestFlagOverridesConfigSessionsRoot(t *testing.T) {
	workspace, sessionsRoot, _, _ := fixtureRoots(t)

	invalidRoot := filepath.Join(workspace, "does-not-exist")
	configFile := filepath.Join(workspace, "config.json")
	configBody := []byte(`{"sessions_root":"` + invalidRoot + `"}`)
	if err := os.WriteFile(configFile, configBody, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	res := runCLI(t, []string{"list", "--sessions-root", sessionsRoot, "--limit", "1", "--color", "never"}, []string{"CSM_CONFIG=" + configFile})
	if res.ExitCode != 0 {
		t.Fatalf("expected flag to override config sessions_root, got exit code %d stderr=%q err=%v", res.ExitCode, res.Stderr, res.Err)
	}

	if !strings.Contains(res.Stdout, "ID") {
		t.Fatalf("expected list output header, got: %q", res.Stdout)
	}
}

func TestDeleteDryRunDoesNotMoveSession(t *testing.T) {
	_, sessionsRoot, trashRoot, logFile := fixtureRoots(t)

	item, err := firstSessionFromList(t, sessionsRoot)
	if err != nil {
		t.Fatalf("firstSessionFromList: %v", err)
	}

	id := item.SessionID
	beforeSessionPath := item.Path

	res := runCLI(t, []string{
		"delete",
		"--sessions-root", sessionsRoot,
		"--trash-root", trashRoot,
		"--log-file", logFile,
		"--id", id,
		"--dry-run",
	}, nil)
	if res.ExitCode != 0 {
		t.Fatalf("expected delete dry-run exit code 0, got %d stderr=%q err=%v", res.ExitCode, res.Stderr, res.Err)
	}

	if !strings.Contains(res.Stdout, "simulation=true") {
		t.Fatalf("expected simulation summary in stdout, got %q", res.Stdout)
	}

	if _, err := os.Stat(beforeSessionPath); err != nil {
		t.Fatalf("expected session to remain after dry-run: %v", err)
	}

	if _, moved, err := lookupSessionByID(t, filepath.Join(trashRoot, "sessions"), id); err != nil {
		t.Fatalf("lookup trashed session: %v", err)
	} else if moved {
		t.Fatalf("session %q unexpectedly moved to trash on dry-run", id)
	}
}

func TestRestoreDryRunDoesNotMoveSessionBack(t *testing.T) {
	_, sessionsRoot, trashRoot, logFile := fixtureRoots(t)

	item, err := firstSessionFromList(t, sessionsRoot)
	if err != nil {
		t.Fatalf("firstSessionFromList: %v", err)
	}

	id := item.SessionID

	deleteRes := runCLI(t, []string{
		"delete",
		"--sessions-root", sessionsRoot,
		"--trash-root", trashRoot,
		"--log-file", logFile,
		"--id", id,
		"--dry-run=false",
		"--confirm",
		"--yes",
	}, nil)
	if deleteRes.ExitCode != 0 {
		t.Fatalf("expected real delete exit code 0, got %d stderr=%q err=%v", deleteRes.ExitCode, deleteRes.Stderr, deleteRes.Err)
	}

	if _, inSessions, err := lookupSessionByID(t, sessionsRoot, id); err != nil {
		t.Fatalf("lookup session in sessions root: %v", err)
	} else if inSessions {
		t.Fatalf("expected session %q to be moved out of sessions after real delete", id)
	}

	trashSessionsRoot := filepath.Join(trashRoot, "sessions")
	trashItem, inTrash, err := lookupSessionByID(t, trashSessionsRoot, id)
	if err != nil {
		t.Fatalf("lookup session in trash root: %v", err)
	}

	if !inTrash {
		t.Fatalf("expected session %q to exist in trash after real delete", id)
	}

	restoreRes := runCLI(t, []string{
		"restore",
		"--sessions-root", sessionsRoot,
		"--trash-root", trashRoot,
		"--log-file", logFile,
		"--id", id,
		"--dry-run",
	}, nil)
	if restoreRes.ExitCode != 0 {
		t.Fatalf("expected restore dry-run exit code 0, got %d stderr=%q err=%v", restoreRes.ExitCode, restoreRes.Stderr, restoreRes.Err)
	}

	if !strings.Contains(restoreRes.Stdout, "simulation=true") {
		t.Fatalf("expected simulation summary in stdout, got %q", restoreRes.Stdout)
	}

	if _, err := os.Stat(trashItem.Path); err != nil {
		t.Fatalf("expected trashed file to remain after restore dry-run: %v", err)
	}

	if _, inSessionsAfter, err := lookupSessionByID(t, sessionsRoot, id); err != nil {
		t.Fatalf("lookup session after restore dry-run: %v", err)
	} else if inSessionsAfter {
		t.Fatalf("session %q unexpectedly restored during dry-run", id)
	}
}

func TestAgentsLintStrictJSONExitAndPayload(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	repo := filepath.Join(t.TempDir(), "repo")
	cwd := filepath.Join(repo, "sub")

	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatalf("mkdir home codex: %v", err)
	}

	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir cwd: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("Prefer rg for text search.\nPrefer rg for text search.\n"), 0o644); err != nil {
		t.Fatalf("write repo agents: %v", err)
	}

	res := runCLI(t, []string{"agents", "lint", "--cwd", cwd, "--strict", "--format", "json"}, []string{"HOME=" + home})
	if res.ExitCode != 1 {
		t.Fatalf("expected exit code 1 for strict lint warnings, got %d stderr=%q err=%v", res.ExitCode, res.Stderr, res.Err)
	}

	if strings.TrimSpace(res.Stdout) == "" {
		t.Fatal("expected JSON payload on stdout for strict lint")
	}

	var payload struct {
		Summary struct {
			Warnings int `json:"warnings"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(res.Stdout), &payload); err != nil {
		t.Fatalf("decode lint json payload: %v stdout=%q", err, res.Stdout)
	}

	if payload.Summary.Warnings == 0 {
		t.Fatalf("expected warnings in lint payload: %+v", payload)
	}

	if !strings.Contains(res.Stderr, "strict mode") {
		t.Fatalf("expected strict mode error in stderr, got %q", res.Stderr)
	}
}

func fixtureRoots(t *testing.T) (workspace, sessionsRoot, trashRoot, logFile string) {
	t.Helper()

	workspace = testsupport.PrepareFixtureSandbox(t, fixtureName)
	sessionsRoot = filepath.Join(workspace, "sessions")
	trashRoot = filepath.Join(workspace, "trash")
	logFile = filepath.Join(workspace, "logs", "actions.log")

	return workspace, sessionsRoot, trashRoot, logFile
}

func runCLI(t *testing.T, args []string, extraEnv []string) runResult {
	t.Helper()

	binary := buildBinary(t)

	cmd := exec.Command(binary, args...)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), extraEnv...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return runResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Err:      err,
	}
}

func buildBinary(t *testing.T) string {
	t.Helper()

	buildOnce.Do(func() {
		repo := repoRoot(t)
		workspace, err := os.MkdirTemp("", "codexsm-blackbox-build-")
		if err != nil {
			buildErr = err
			return
		}

		binPath = filepath.Join(workspace, "codexsm-blackbox")

		cmd := exec.Command("go", "build", "-o", binPath, ".")
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GOEXPERIMENT=jsonv2")

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			buildErr = errors.New(stderr.String())
		}
	})

	if buildErr != nil {
		t.Fatalf("build blackbox binary: %v", buildErr)
	}

	return binPath
}

func repoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	cur := wd
	for {
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			return cur
		}

		next := filepath.Dir(cur)
		if next == cur {
			t.Fatalf("cannot locate repo root from %q", wd)
		}

		cur = next
	}
}

type listedSession struct {
	SessionID string `json:"session_id"`
	Path      string `json:"path"`
}

func firstSessionFromList(t *testing.T, sessionsRoot string) (listedSession, error) {
	t.Helper()

	res := runCLI(t, []string{"list", "--sessions-root", sessionsRoot, "--format", "json", "--limit", "50"}, nil)
	if res.ExitCode != 0 {
		return listedSession{}, errors.New("list command failed")
	}

	var items []listedSession
	if err := json.Unmarshal([]byte(res.Stdout), &items); err != nil {
		return listedSession{}, err
	}

	if len(items) == 0 {
		return listedSession{}, errors.New("no sessions returned by list")
	}

	for _, item := range items {
		if strings.TrimSpace(item.SessionID) == "" {
			continue
		}

		if strings.TrimSpace(item.Path) == "" {
			continue
		}

		return item, nil
	}

	return listedSession{}, errors.New("no selectable session with non-empty session_id")
}

func lookupSessionByID(t *testing.T, sessionsRoot, id string) (listedSession, bool, error) {
	t.Helper()

	res := runCLI(t, []string{"list", "--sessions-root", sessionsRoot, "--format", "json", "--limit", "1", "--id", id}, nil)
	if res.ExitCode != 0 {
		if strings.Contains(res.Stderr, "no sessions matched") {
			return listedSession{}, false, nil
		}

		return listedSession{}, false, errors.New("list lookup command failed")
	}

	var items []listedSession
	if err := json.Unmarshal([]byte(res.Stdout), &items); err != nil {
		return listedSession{}, false, err
	}

	if len(items) == 0 {
		return listedSession{}, false, nil
	}

	return items[0], true, nil
}
