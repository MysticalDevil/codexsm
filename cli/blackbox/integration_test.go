//go:build integration
// +build integration

package blackbox

import (
	"bytes"
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
