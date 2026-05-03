package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreCallEditGuard_NoOpOnIrrelevantTools(t *testing.T) {
	// preCallEditGuard writes to stderr; assert it does not panic or fire on
	// tools that aren't body-mutating, on missing args, or on insert variants.
	preCallEditGuard("serena", "find_symbol", map[string]any{"body": "type X struct{}", "relative_path": "x.go"})
	preCallEditGuard("serena", "replace_symbol_body", map[string]any{})
	preCallEditGuard("serena", "replace_symbol_body", map[string]any{"body": "type X struct{}", "relative_path": "x.unknown"})
	preCallEditGuard("serena", "insert_after_symbol", map[string]any{"body": "type X struct{}", "relative_path": "x.go"})
}

func TestPreCallEditGuard_DetectsAcrossLanguages(t *testing.T) {
	cases := []struct {
		ext  string
		body string
	}{
		{".go", "type Foo struct{}"},
		{".go", "func Foo() {}"},
		{".py", "def foo():"},
		{".py", "class Foo:"},
		{".py", "async def foo():"},
		{".ts", "function foo() {}"},
		{".ts", "class Foo {}"},
		{".ts", "interface Foo {}"},
		{".rs", "fn foo() {}"},
		{".rs", "struct Foo {}"},
		{".rb", "def foo end"},
		{".java", "class Foo {}"},
	}
	for _, c := range cases {
		args := map[string]any{"body": c.body, "relative_path": "x" + c.ext}
		// We can't capture stderr without plumbing — just confirm no panic.
		preCallEditGuard("serena", "replace_symbol_body", args)
	}
}

func TestPostCallEditGuard_ReportsBrokenGo(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "broken.go")
	if err := os.WriteFile(bad, []byte("package x\nfunc func Foo() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := postCallEditGuard("serena", "replace_symbol_body",
		map[string]any{"relative_path": "broken.go"}, dir)
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestPostCallEditGuard_AllowsValidGo(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.go")
	if err := os.WriteFile(good, []byte("package x\nfunc Foo() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := postCallEditGuard("serena", "replace_symbol_body",
		map[string]any{"relative_path": "good.go"}, dir)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestPostCallEditGuard_SkipsNonGo(t *testing.T) {
	err := postCallEditGuard("serena", "replace_symbol_body",
		map[string]any{"relative_path": "anything.py"}, t.TempDir())
	if err != nil {
		t.Errorf("non-Go should pass through: %v", err)
	}
}

func TestPostCallEditGuard_SkipsNonEditTools(t *testing.T) {
	err := postCallEditGuard("serena", "find_symbol",
		map[string]any{"relative_path": "broken.go"}, t.TempDir())
	if err != nil {
		t.Errorf("non-edit tool should pass through: %v", err)
	}
}

func TestPostCallEditGuard_HandlesMissingFile(t *testing.T) {
	err := postCallEditGuard("serena", "replace_symbol_body",
		map[string]any{"relative_path": "does/not/exist.go"}, t.TempDir())
	if err != nil {
		t.Errorf("missing file should not error: %v", err)
	}
}
