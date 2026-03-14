package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MysticalDevil/codexsm/internal/testsupport"
)

func TestCopyAndMoveFile(t *testing.T) {
	workspace := testsupport.PrepareFixtureSandbox(t, "rich")
	root := filepath.Join(workspace, "tmp-files")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir test root: %v", err)
	}

	src := filepath.Join(root, "src.txt")
	dst := filepath.Join(root, "dst.txt")
	if err := os.WriteFile(src, []byte("abc"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil || string(data) != "abc" {
		t.Fatalf("copied content mismatch err=%v data=%q", err, string(data))
	}

	src2 := filepath.Join(root, "src2.txt")
	dst2 := filepath.Join(root, "dst2.txt")
	if err := os.WriteFile(src2, []byte("xyz"), 0o644); err != nil {
		t.Fatalf("write src2: %v", err)
	}
	if err := MoveFile(src2, dst2); err != nil {
		t.Fatalf("MoveFile: %v", err)
	}
	if _, err := os.Stat(src2); !os.IsNotExist(err) {
		t.Fatalf("source should be removed after move, err=%v", err)
	}
	data2, err := os.ReadFile(dst2)
	if err != nil || string(data2) != "xyz" {
		t.Fatalf("moved content mismatch err=%v data=%q", err, string(data2))
	}
}
