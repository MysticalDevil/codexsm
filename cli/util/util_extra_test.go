package util

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMarshalPrettyJSONProducesIndentedOutput(t *testing.T) {
	b, err := MarshalPrettyJSON(map[string]any{"k": "v"})
	if err != nil {
		t.Fatalf("marshal pretty json: %v", err)
	}

	text := string(b)
	if !strings.Contains(text, "\n") || !strings.Contains(text, "  ") {
		t.Fatalf("expected pretty json, got: %q", text)
	}
}

func TestWriteFileAtomicWritesExpectedContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "x.json")

	if err := WriteFileAtomic(path, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write file atomic: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	if string(data) != "hello" {
		t.Fatalf("unexpected file content: %q", string(data))
	}
}

func TestStripANSIAndTerminalHelpers(t *testing.T) {
	in := "\x1b[31mred\x1b[0m"
	if got := StripANSI(in); got != "red" {
		t.Fatalf("strip ansi mismatch: %q", got)
	}

	if IsTerminalWriter(&bytes.Buffer{}) {
		t.Fatal("bytes buffer should not be terminal writer")
	}

	if ShouldUseColor("never", &bytes.Buffer{}) {
		t.Fatal("never mode should disable color")
	}
}

func TestExitErrorAndWithExitCodeHelpers(t *testing.T) {
	if WithExitCode(nil, 2) != nil {
		t.Fatal("expected nil when wrapping nil error")
	}

	base := errors.New("boom")
	err := WithExitCode(base, 2)
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError wrapper, got: %T", err)
	}

	if exitErr.ExitCode() != 2 {
		t.Fatalf("unexpected exit code: %d", exitErr.ExitCode())
	}

	if exitErr.Error() == "" {
		t.Fatal("expected non-empty error text")
	}

	if (&ExitError{}).ExitCode() != 1 {
		t.Fatal("expected default exit code fallback")
	}
}

func TestShouldUseColorAutoAndParseHealthInvalid(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if ShouldUseColor("auto", &bytes.Buffer{}) {
		t.Fatal("auto mode should disable color when NO_COLOR is set")
	}

	if _, err := ParseHealth("bad"); err == nil {
		t.Fatal("expected invalid health error")
	}
}
