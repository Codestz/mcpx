package stats

import (
	"path/filepath"
	"testing"
	"time"
)

func TestWriterAppendsAndReads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.jsonl")
	w, err := NewWriter(path, 16, true)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		w.Write(Record{
			Server: "serena", Tool: "find_symbol",
			ArgsBytes: 100, ResponseBytes: 1000,
			LatencyMS: int64(10 + i), TokensSaved: 200,
		})
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	recs, err := Collect(path, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 5 {
		t.Fatalf("want 5 records, got %d", len(recs))
	}
}

func TestWriterDisabledIsNoop(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.jsonl")
	w, err := NewWriter(path, 16, false)
	if err != nil {
		t.Fatal(err)
	}
	w.Write(Record{Server: "x", Tool: "y"})
	w.Close()

	recs, _ := Collect(path, Filter{})
	if len(recs) != 0 {
		t.Fatalf("disabled writer wrote %d records", len(recs))
	}
}

func TestFilterMatching(t *testing.T) {
	now := time.Now().UTC()
	r := Record{TS: now, Project: "/p", Server: "s", Tool: "t", Agent: "a"}
	cases := []struct {
		f    Filter
		want bool
	}{
		{Filter{}, true},
		{Filter{Server: "s"}, true},
		{Filter{Server: "x"}, false},
		{Filter{Project: "/p"}, true},
		{Filter{Project: "/q"}, false},
		{Filter{Tool: "t"}, true},
		{Filter{Tool: "u"}, false},
		{Filter{Since: now.Add(-1 * time.Hour)}, true},
		{Filter{Since: now.Add(1 * time.Hour)}, false},
		{Filter{Until: now.Add(1 * time.Hour)}, true},
		{Filter{Until: now.Add(-1 * time.Hour)}, false},
		{Filter{Agent: "a"}, true},
		{Filter{Agent: "b"}, false},
	}
	for i, c := range cases {
		if got := c.f.matches(r); got != c.want {
			t.Errorf("case %d: filter %+v: got %v want %v", i, c.f, got, c.want)
		}
	}
}

func TestAggregate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.jsonl")
	w, _ := NewWriter(path, 16, true)
	now := time.Now().UTC()

	w.Write(Record{TS: now, Server: "s", Tool: "find_symbol", LatencyMS: 10, TokensSaved: 100, ArgsTokensEst: 5, ResponseTokensEst: 50, SchemaCacheHit: true})
	w.Write(Record{TS: now, Server: "s", Tool: "find_symbol", LatencyMS: 20, TokensSaved: 200, ArgsTokensEst: 5, ResponseTokensEst: 60, SchemaCacheHit: true})
	w.Write(Record{TS: now, Server: "s", Tool: "search", LatencyMS: 100, TokensSaved: 50, ArgsTokensEst: 5, ResponseTokensEst: 200, ExitCode: 1})
	w.Close()

	s, err := Aggregate(path, Filter{}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if s.Calls != 3 {
		t.Errorf("calls: got %d want 3", s.Calls)
	}
	if s.TokensSaved != 350 {
		t.Errorf("tokens saved: got %d want 350", s.TokensSaved)
	}
	if len(s.TopTools) != 2 {
		t.Errorf("top tools: got %d want 2", len(s.TopTools))
	}
	if s.TopTools[0].Tool != "find_symbol" {
		t.Errorf("top tool: got %s want find_symbol", s.TopTools[0].Tool)
	}
	if s.ErrorRate < 0.33 || s.ErrorRate > 0.34 {
		t.Errorf("error rate: got %f want ~0.333", s.ErrorRate)
	}
	if s.SchemaHitRate < 0.66 || s.SchemaHitRate > 0.67 {
		t.Errorf("schema hit rate: got %f want ~0.667", s.SchemaHitRate)
	}
	if len(s.Recent) != 2 {
		t.Errorf("recent: got %d want 2", len(s.Recent))
	}
}

func TestEstimateTokens(t *testing.T) {
	if got := EstimateTokens([]byte("12345678")); got != 2 {
		t.Errorf("8 bytes: got %d want 2", got)
	}
	if got := EstimateTokens([]byte("123")); got != 1 {
		t.Errorf("3 bytes: got %d want 1", got)
	}
	if got := EstimateTokens([]byte("")); got != 0 {
		t.Errorf("empty: got %d want 0", got)
	}
}
