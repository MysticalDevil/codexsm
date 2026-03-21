package list

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/session"
)

func TestColumnValueAndHelpers(t *testing.T) {
	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	s := session.Session{
		SessionID: "12345678-1234-1234-1234-1234567890ab",
		CreatedAt: now,
		UpdatedAt: now,
		SizeBytes: 1024,
		Path:      "/tmp/a.jsonl",
		HostDir:   "/home/u/work",
		Health:    session.HealthCorrupted,
		Head:      "abcdefghijklmnopqrstuvwxyz",
	}

	if got := ColumnValue("id", s, "/home/u", 8, true); got == "" {
		t.Fatal("expected id column")
	}

	if got := ColumnValue("session_id", s, "/home/u", 8, true); got == "" {
		t.Fatal("expected session_id column")
	}

	if got := ColumnValue("created_at", s, "/home/u", 8, true); !strings.Contains(got, "2026-03-21") {
		t.Fatalf("unexpected created_at: %q", got)
	}

	if got := ColumnValue("updated_at", s, "/home/u", 8, true); !strings.Contains(got, "2026-03-21") {
		t.Fatalf("unexpected updated_at: %q", got)
	}

	if got := ColumnValue("size", s, "/home/u", 8, true); got == "" {
		t.Fatal("expected size column")
	}

	if got := ColumnValue("size_bytes", s, "/home/u", 8, true); got != "1024" {
		t.Fatalf("unexpected size_bytes: %q", got)
	}

	if got := ColumnValue("health", s, "/home/u", 8, true); got == "" {
		t.Fatal("expected health column")
	}

	if got := ColumnValue("host", s, "/home/u", 8, true); !strings.Contains(got, "~/work") {
		t.Fatalf("unexpected host compact path: %q", got)
	}

	if got := ColumnValue("path", s, "/home/u", 8, true); !strings.Contains(got, "/tmp/a.jsonl") {
		t.Fatalf("unexpected path: %q", got)
	}

	if got := ColumnValue("head", s, "/home/u", 8, true); !strings.Contains(got, "...") {
		t.Fatalf("expected truncated head, got: %q", got)
	}

	if got := TruncateDisplayText("abcdef", 4); got != "abcd..." {
		t.Fatalf("unexpected truncate result: %q", got)
	}

	if got := TruncateDisplayText("abc", 0); got != "abc" {
		t.Fatalf("expected passthrough for zero width, got: %q", got)
	}

	if !HasHealthColumn([]Column{{Key: "id"}, {Key: "health"}}) {
		t.Fatal("expected health column detection")
	}

	if HasHealthColumn([]Column{{Key: "id"}, {Key: "path"}}) {
		t.Fatal("did not expect health column detection")
	}
}

func TestWriteDelimitedCSVAndTSV(t *testing.T) {
	items := []session.Session{
		{
			SessionID: "x1",
			UpdatedAt: time.Date(2026, 3, 21, 13, 0, 0, 0, time.UTC),
			SizeBytes: 5,
			Health:    session.HealthOK,
			Path:      "/tmp/a.jsonl",
		},
	}
	cols := []Column{{Key: "id", Header: "ID"}, {Key: "updated_at", Header: "UPDATED_AT"}}

	var csvOut bytes.Buffer
	if err := WriteDelimited(&csvOut, items, ',', false, cols); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	if !strings.Contains(csvOut.String(), "id,updated_at") {
		t.Fatalf("expected csv header, got: %q", csvOut.String())
	}

	var tsvOut bytes.Buffer
	if err := WriteDelimited(&tsvOut, items, '\t', false, cols); err != nil {
		t.Fatalf("write tsv: %v", err)
	}

	if !strings.Contains(tsvOut.String(), "id\tupdated_at") {
		t.Fatalf("expected tsv header, got: %q", tsvOut.String())
	}
}

func TestParseColumnsModes(t *testing.T) {
	cols, err := ParseColumns("", false, "table")
	if err != nil {
		t.Fatalf("parse columns default table: %v", err)
	}

	if len(cols) == 0 {
		t.Fatal("expected default table columns")
	}

	cols, err = ParseColumns("id,health,path", false, "csv")
	if err != nil {
		t.Fatalf("parse columns csv: %v", err)
	}

	if len(cols) != 3 {
		t.Fatalf("unexpected custom columns len: %d", len(cols))
	}

	if _, err := ParseColumns("bad-column", false, "table"); err == nil {
		t.Fatal("expected invalid column error")
	}
}
