package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestFileMtimeExtras_InvalidatesOnEdit pins the correctness fix for
// idempotent caching: when a file referenced by `relative_path` changes, the
// extras list shifts so the resulting cache key shifts, forcing a fresh fetch.
func TestFileMtimeExtras_InvalidatesOnEdit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.txt")
	if err := os.WriteFile(path, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}

	args := map[string]any{"relative_path": "x.txt"}
	before := fileMtimeExtras(args, dir)
	if len(before) != 1 {
		t.Fatalf("expected 1 extra, got %v", before)
	}

	// Touch the file with a clearly later mtime. The brief sleep guards against
	// nsec mtime collisions on very-fast filesystems.
	time.Sleep(20 * time.Millisecond)
	if err := os.WriteFile(path, []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	after := fileMtimeExtras(args, dir)
	if len(after) != 1 {
		t.Fatalf("expected 1 extra, got %v", after)
	}
	if before[0] == after[0] {
		t.Errorf("mtime extras did not change after edit: %s", before[0])
	}
}

func TestFileMtimeExtras_MissingFileNoExtra(t *testing.T) {
	args := map[string]any{"relative_path": "/nonexistent/path/xyz.txt"}
	if extras := fileMtimeExtras(args, ""); len(extras) != 0 {
		t.Errorf("missing file should produce no extras, got %v", extras)
	}
}

func TestFileMtimeExtras_NoFilePathKey(t *testing.T) {
	args := map[string]any{"name": "Foo", "depth": 3}
	if extras := fileMtimeExtras(args, ""); len(extras) != 0 {
		t.Errorf("args without filepath keys should produce no extras, got %v", extras)
	}
}

func TestFileMtimeExtras_HandlesAlternativeKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "y.txt")
	if err := os.WriteFile(path, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"path", "file", "file_path"} {
		args := map[string]any{key: "y.txt"}
		if extras := fileMtimeExtras(args, dir); len(extras) != 1 {
			t.Errorf("key %s: expected 1 extra, got %v", key, extras)
		}
	}
}
