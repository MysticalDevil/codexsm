package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePath(t *testing.T) {
	t.Setenv("HOME", "/tmp/home-sim")

	got, err := ResolvePath("")
	if err != nil {
		t.Fatalf("ResolvePath empty: %v", err)
	}
	if got != "" {
		t.Fatalf("unexpected empty resolve: %q", got)
	}

	got, err = ResolvePath("~/work")
	if err != nil {
		t.Fatalf("ResolvePath ~/work: %v", err)
	}
	if got != "/tmp/home-sim/work" {
		t.Fatalf("unexpected resolved path: %q", got)
	}

	got, err = ResolvePath("/var/tmp/x")
	if err != nil {
		t.Fatalf("ResolvePath absolute: %v", err)
	}
	if got != "/var/tmp/x" {
		t.Fatalf("unexpected absolute resolve: %q", got)
	}
}

func TestDefaultPaths(t *testing.T) {
	t.Setenv("HOME", "/tmp/home-sim")

	t.Setenv("SESSIONS_ROOT", "~/custom-sessions")
	sessionsRoot, err := DefaultSessionsRoot()
	if err != nil {
		t.Fatalf("DefaultSessionsRoot env: %v", err)
	}
	if sessionsRoot != "/tmp/home-sim/custom-sessions" {
		t.Fatalf("unexpected sessions root: %q", sessionsRoot)
	}

	t.Setenv("SESSIONS_ROOT", "")
	sessionsRoot, err = DefaultSessionsRoot()
	if err != nil {
		t.Fatalf("DefaultSessionsRoot default: %v", err)
	}
	if sessionsRoot != "/tmp/home-sim/.codex/sessions" {
		t.Fatalf("unexpected default sessions root: %q", sessionsRoot)
	}

	trashRoot, err := DefaultTrashRoot()
	if err != nil {
		t.Fatalf("DefaultTrashRoot: %v", err)
	}
	if trashRoot != "/tmp/home-sim/.codex/trash" {
		t.Fatalf("unexpected trash root: %q", trashRoot)
	}

	logFile, err := DefaultLogFile()
	if err != nil {
		t.Fatalf("DefaultLogFile: %v", err)
	}
	if logFile != "/tmp/home-sim/.codex/codexsm/logs/actions.log" {
		t.Fatalf("unexpected log file: %q", logFile)
	}
}

func TestEnsureDirForFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "a", "b", "c", "actions.log")
	if err := EnsureDirForFile(target); err != nil {
		t.Fatalf("EnsureDirForFile nested: %v", err)
	}
	if st, err := os.Stat(filepath.Dir(target)); err != nil || !st.IsDir() {
		t.Fatalf("expected dir created, err=%v", err)
	}

	if err := EnsureDirForFile("actions.log"); err != nil {
		t.Fatalf("EnsureDirForFile current dir: %v", err)
	}
}
