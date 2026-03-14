package util

import (
	"bufio"
	"bytes"
	"io"
	"testing"
)

func TestReadBoundedLineLargeWithinLimit(t *testing.T) {
	lineBytes := bytes.Repeat([]byte("a"), 12<<10)
	lineBytes = append(lineBytes, '\n')
	r := bufio.NewReaderSize(bytes.NewReader(lineBytes), 4<<10)

	line, truncated, err := ReadBoundedLine(r, 1<<20)
	if err != nil {
		t.Fatalf("ReadBoundedLine: %v", err)
	}
	if truncated {
		t.Fatal("expected non-truncated line")
	}
	if got, want := len(line), 12<<10; got != want {
		t.Fatalf("line length=%d want=%d", got, want)
	}
}

func TestReadBoundedLineTruncatedAndEof(t *testing.T) {
	lineBytes := bytes.Repeat([]byte("x"), 32)
	r := bufio.NewReader(bytes.NewReader(lineBytes))

	line, truncated, err := ReadBoundedLine(r, 8)
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
	if !truncated {
		t.Fatal("expected truncated line")
	}
	if got, want := len(line), 8; got != want {
		t.Fatalf("line length=%d want=%d", got, want)
	}
}
