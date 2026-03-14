package cli

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestMigrateDryRunAndRealExecution(t *testing.T) {
	root := t.TempDir()
	sessionsRoot := filepath.Join(root, "sessions")

	srcRollout := filepath.Join(sessionsRoot, "2026", "03", "10", "rollout-2026-03-10-old-id.jsonl")
	if err := os.MkdirAll(filepath.Dir(srcRollout), 0o755); err != nil {
		t.Fatal(err)
	}

	content := strings.Join([]string{
		`{"type":"session_meta","payload":{"id":"old-id","timestamp":"2026-03-10T01:00:00Z","cwd":"/old"}}`,
		`{"type":"turn_context","payload":{"cwd":"/old"}}`,
		`{"type":"response_item","payload":{"type":"message","role":"user","text":"hello"}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(srcRollout, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(root, "state.sqlite")
	createCLIMigrationDB(t, dbPath)
	insertCLIMigrationRow(t, dbPath, "old-id", srcRollout, "/old", "main")

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"session", "migrate",
		"--from", "/old",
		"--to", filepath.Join(root, "new-main"),
		"--sessions-root", sessionsRoot,
		"--codex-state-db", dbPath,
		"--print-created",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("dry-run execute: %v", err)
	}

	if !strings.Contains(stdout.String(), "session-migrate: action=dry-run matched=1") {
		t.Fatalf("unexpected dry-run output: %s", stdout.String())
	}

	cmd = NewRootCmd()

	stdout.Reset()
	stderr.Reset()
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"session", "migrate",
		"--from", "/old",
		"--to", filepath.Join(root, "new-main"),
		"--sessions-root", sessionsRoot,
		"--codex-state-db", dbPath,
		"--dry-run=false",
		"--confirm",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("real execute: %v", err)
	}

	if !strings.Contains(stdout.String(), "session-migrate: action=migrate matched=1 planned=1 created=1") {
		t.Fatalf("unexpected migrate output: %s", stdout.String())
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow(`select count(*) from threads where cwd = ?`, filepath.Join(root, "new-main")).Scan(&count); err != nil {
		t.Fatal(err)
	}

	if count != 1 {
		t.Fatalf("expected one migrated thread row, got %d", count)
	}
}

func TestMigrateFileDryRun(t *testing.T) {
	root := t.TempDir()
	sessionsRoot := filepath.Join(root, "sessions")

	srcRollout := filepath.Join(sessionsRoot, "2026", "03", "10", "rollout-2026-03-10-old-id.jsonl")
	if err := os.MkdirAll(filepath.Dir(srcRollout), 0o755); err != nil {
		t.Fatal(err)
	}

	content := strings.Join([]string{
		`{"type":"session_meta","payload":{"id":"old-id","timestamp":"2026-03-10T01:00:00Z","cwd":"/old"}}`,
		`{"type":"turn_context","payload":{"cwd":"/old"}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(srcRollout, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(root, "state.sqlite")
	createCLIMigrationDB(t, dbPath)
	insertCLIMigrationRow(t, dbPath, "old-id", srcRollout, "/old", "main")

	filePath := filepath.Join(root, "migrate.toml")

	fileContent := `
[[mapping]]
from = "/old"
to = "` + filepath.Join(root, "new-main") + `"

[[mapping]]
from = "/missing"
to = "` + filepath.Join(root, "missing-main") + `"
`
	if err := os.WriteFile(filePath, []byte(fileContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{
		"session", "migrate",
		"--file", filePath,
		"--sessions-root", sessionsRoot,
		"--codex-state-db", dbPath,
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected dry-run batch error")
	}

	if !strings.Contains(stdout.String(), "session-migrate-batch: action=dry-run mappings=2 succeeded=1 failed=1") {
		t.Fatalf("unexpected batch output: %s", stdout.String())
	}

	if !strings.Contains(err.Error(), "1 migration mapping(s) failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMigrateRejectsMixedFileAndDirectFlags(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"session", "migrate",
		"--file", "migrate.toml",
		"--from", "/old",
		"--to", "/new",
	})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--file cannot be combined with --from or --to") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func createCLIMigrationDB(t *testing.T, path string) {
	t.Helper()

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
CREATE TABLE threads (
    id TEXT PRIMARY KEY,
    rollout_path TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    source TEXT NOT NULL,
    model_provider TEXT NOT NULL,
    cwd TEXT NOT NULL,
    title TEXT NOT NULL,
    sandbox_policy TEXT NOT NULL,
    approval_mode TEXT NOT NULL,
    tokens_used INTEGER NOT NULL DEFAULT 0,
    has_user_event INTEGER NOT NULL DEFAULT 0,
    archived INTEGER NOT NULL DEFAULT 0,
    archived_at INTEGER,
    git_sha TEXT,
    git_branch TEXT,
    git_origin_url TEXT,
    cli_version TEXT NOT NULL DEFAULT '',
    first_user_message TEXT NOT NULL DEFAULT '',
    agent_nickname TEXT,
    agent_role TEXT,
    memory_mode TEXT NOT NULL DEFAULT 'enabled'
)`)
	if err != nil {
		t.Fatal(err)
	}
}

func insertCLIMigrationRow(t *testing.T, dbPath, id, rolloutPath, cwd, branch string) {
	t.Helper()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ts := time.Unix(1773072000, 0).UTC()

	_, err = db.Exec(`
INSERT INTO threads (
    id, rollout_path, created_at, updated_at, source, model_provider, cwd, title,
    sandbox_policy, approval_mode, tokens_used, has_user_event, archived, archived_at,
    git_sha, git_branch, git_origin_url, cli_version, first_user_message, agent_nickname, agent_role, memory_mode
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, rolloutPath, ts.Unix(), ts.Unix(), "cli", "openai", cwd, "title",
		"{}", "on-request", 11, 1, 0, nil, nil, branch, nil, "0.112.0", "hello", nil, nil, "enabled",
	)
	if err != nil {
		t.Fatal(err)
	}
}
