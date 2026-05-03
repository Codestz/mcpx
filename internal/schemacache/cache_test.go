package schemacache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/codestz/mcpx/internal/mcp"
)

func withTempHome(t *testing.T) func() {
	t.Helper()
	tmp := t.TempDir()
	old := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	return func() { os.Setenv("HOME", old) }
}

func TestKeyStable(t *testing.T) {
	k1 := Key("serena", []string{"start"}, map[string]string{"FOO": "bar"}, "")
	k2 := Key("serena", []string{"start"}, map[string]string{"FOO": "bar"}, "")
	if k1 != k2 {
		t.Errorf("expected stable keys, got %s vs %s", k1, k2)
	}
	k3 := Key("serena", []string{"start"}, map[string]string{"FOO": "baz"}, "")
	if k1 == k3 {
		t.Errorf("expected different keys for different env, got same: %s", k1)
	}
}

func TestSaveAndLoad(t *testing.T) {
	defer withTempHome(t)()

	key := "abc123"
	e := &Entry{
		TTL: 5 * time.Minute,
		Tools: []mcp.Tool{
			{Name: "find_symbol", Description: "find a symbol"},
		},
	}
	if err := Save(key, e); err != nil {
		t.Fatal(err)
	}

	loaded, hit, err := Load(key)
	if err != nil {
		t.Fatal(err)
	}
	if !hit {
		t.Fatal("expected cache hit")
	}
	if loaded.Tools[0].Name != "find_symbol" {
		t.Errorf("tool name: got %s", loaded.Tools[0].Name)
	}
	if loaded.NativeBaselineToks <= 0 {
		t.Errorf("baseline tokens should be > 0, got %d", loaded.NativeBaselineToks)
	}
	if loaded.Version != 1 {
		t.Errorf("version: got %d", loaded.Version)
	}
}

func TestExpiredEntryMisses(t *testing.T) {
	defer withTempHome(t)()
	key := "expired"
	e := &Entry{
		CapturedAt: time.Now().Add(-1 * time.Hour),
		TTL:        1 * time.Minute,
	}
	if err := Save(key, e); err != nil {
		t.Fatal(err)
	}
	_, hit, _ := Load(key)
	if hit {
		t.Error("expected miss on expired entry")
	}
}

func TestMissingEntryMisses(t *testing.T) {
	defer withTempHome(t)()
	_, hit, err := Load("never-saved")
	if err != nil {
		t.Fatal(err)
	}
	if hit {
		t.Error("expected miss")
	}
}

func TestClear(t *testing.T) {
	defer withTempHome(t)()
	key := "to-clear"
	Save(key, &Entry{TTL: time.Hour})
	if _, hit, _ := Load(key); !hit {
		t.Fatal("expected initial hit")
	}
	if err := Clear(key); err != nil {
		t.Fatal(err)
	}
	if _, hit, _ := Load(key); hit {
		t.Error("expected miss after clear")
	}
}

func TestClearAll(t *testing.T) {
	defer withTempHome(t)()
	Save("a", &Entry{TTL: time.Hour})
	Save("b", &Entry{TTL: time.Hour})
	if err := ClearAll(); err != nil {
		t.Fatal(err)
	}
	if _, hit, _ := Load("a"); hit {
		t.Error("expected miss after ClearAll")
	}
}

func TestBypass(t *testing.T) {
	old := os.Getenv("MCPX_CACHE")
	defer os.Setenv("MCPX_CACHE", old)

	for _, v := range []string{"off", "0", "false", "OFF"} {
		os.Setenv("MCPX_CACHE", v)
		if !Bypass() {
			t.Errorf("MCPX_CACHE=%s should bypass", v)
		}
	}
	for _, v := range []string{"", "on", "1"} {
		os.Setenv("MCPX_CACHE", v)
		if Bypass() {
			t.Errorf("MCPX_CACHE=%s should not bypass", v)
		}
	}
}

func TestComputeBaselineGrowsWithSize(t *testing.T) {
	small := &Entry{Tools: []mcp.Tool{{Name: "x"}}}
	big := &Entry{}
	for i := 0; i < 50; i++ {
		big.Tools = append(big.Tools, mcp.Tool{Name: "tool", Description: "lengthy description x"})
	}
	if ComputeBaseline(big) <= ComputeBaseline(small) {
		t.Errorf("big baseline should exceed small")
	}
}

func TestCachePathWithHome(t *testing.T) {
	defer withTempHome(t)()
	want := filepath.Join(os.Getenv("HOME"), ".mcpx", "cache", "schemas", "abc.json")
	if got := Path("abc"); got != want {
		t.Errorf("path: got %s want %s", got, want)
	}
}
