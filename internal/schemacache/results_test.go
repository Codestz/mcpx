package schemacache

import (
	"os"
	"testing"
	"time"

	"github.com/codestz/mcpx/internal/mcp"
)

func TestResultKeyStable(t *testing.T) {
	a := ResultKey("s", "t", map[string]any{"x": 1, "y": "two"})
	b := ResultKey("s", "t", map[string]any{"y": "two", "x": 1})
	if a != b {
		t.Errorf("key not stable across map ordering: %s vs %s", a, b)
	}
	c := ResultKey("s", "t", map[string]any{"x": 2, "y": "two"})
	if a == c {
		t.Errorf("different args should yield different keys")
	}
}

func TestResultKeyExtrasParticipateInHash(t *testing.T) {
	args := map[string]any{"x": 1}
	a := ResultKey("s", "t", args)
	b := ResultKey("s", "t", args, "extra1")
	c := ResultKey("s", "t", args, "extra2")
	if a == b || b == c {
		t.Errorf("extras should change the key: %s / %s / %s", a, b, c)
	}
	d := ResultKey("s", "t", args, "extra1")
	if b != d {
		t.Errorf("same extras should yield same key: %s vs %s", b, d)
	}
}

func TestSaveLoadResult(t *testing.T) {
	defer withTempHome(t)()

	key := ResultKey("serena", "find_symbol", map[string]any{"name": "Auth"})
	entry := &ResultEntry{
		TTL:    1 * time.Minute,
		Server: "serena",
		Tool:   "find_symbol",
		Result: mcp.CallResult{Content: []mcp.Content{{Type: "text", Text: "hit"}}},
	}
	if err := SaveResult(key, entry); err != nil {
		t.Fatal(err)
	}
	got, hit, err := LoadResult(key)
	if err != nil {
		t.Fatal(err)
	}
	if !hit {
		t.Fatal("expected cache hit")
	}
	if got.Result.Content[0].Text != "hit" {
		t.Errorf("text: got %q", got.Result.Content[0].Text)
	}
}

func TestResultExpiry(t *testing.T) {
	defer withTempHome(t)()
	key := "expired"
	SaveResult(key, &ResultEntry{
		CapturedAt: time.Now().Add(-1 * time.Hour),
		TTL:        1 * time.Minute,
	})
	_, hit, _ := LoadResult(key)
	if hit {
		t.Error("expired entry should miss")
	}
}

func TestIsIdempotent_AllowList(t *testing.T) {
	tool := &mcp.Tool{Name: "do_something"}
	if !IsIdempotent(tool, "srv", []string{"srv.do_something"}, false) {
		t.Error("allow-list should match")
	}
	if IsIdempotent(tool, "srv", []string{"srv.other"}, false) {
		t.Error("non-matching allow-list should not match")
	}
}

func TestIsIdempotent_Heuristic(t *testing.T) {
	cases := map[string]bool{
		"get_user":    true,
		"list_repos":  true,
		"find_symbol": true,
		"search_x":    true,
		"read_file":   true,
		"write_file":  false,
		"do_thing":    false,
		"create_x":    false,
	}
	for name, want := range cases {
		got := IsIdempotent(&mcp.Tool{Name: name}, "s", nil, true)
		if got != want {
			t.Errorf("heuristic %s: got %v want %v", name, got, want)
		}
	}
}

func TestIsIdempotent_HeuristicOff(t *testing.T) {
	if IsIdempotent(&mcp.Tool{Name: "get_x"}, "s", nil, false) {
		t.Error("heuristic off should not match")
	}
}

func TestCanonicalizeNested(t *testing.T) {
	a := canonicalize(map[string]any{
		"b": []any{1, 2},
		"a": map[string]any{"y": "z", "x": 1},
	})
	b := canonicalize(map[string]any{
		"a": map[string]any{"x": 1, "y": "z"},
		"b": []any{1, 2},
	})
	if a != b {
		t.Errorf("nested canonicalize unstable:\n a=%s\n b=%s", a, b)
	}
}

func TestClearResults(t *testing.T) {
	defer withTempHome(t)()
	key := "x"
	SaveResult(key, &ResultEntry{TTL: time.Hour})
	if _, hit, _ := LoadResult(key); !hit {
		t.Fatal("setup hit failed")
	}
	if err := ClearResults(); err != nil {
		t.Fatal(err)
	}
	if _, hit, _ := LoadResult(key); hit {
		t.Error("expected miss after ClearResults")
	}
}

func TestResultPathInsideHomeDir(t *testing.T) {
	defer withTempHome(t)()
	p := resultPath("abc")
	if p == "" {
		t.Fatal("path empty")
	}
	if got := p; got == "" || got[len(got)-9:] != "abc.json " && got[len(got)-8:] != "abc.json" {
		// trim test: just verify suffix
		if !endsWith(p, "/results/abc.json") {
			t.Errorf("unexpected path: %s", p)
		}
	}
	_ = os.Remove(p)
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
