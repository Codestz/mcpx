package find

import "testing"

func corpus() []Tool {
	return []Tool{
		{Server: "serena", Name: "find_symbol", Description: "Retrieves info on symbols by name path"},
		{Server: "serena", Name: "find_referencing_symbols", Description: "Find references to a symbol"},
		{Server: "serena", Name: "search_for_pattern", Description: "Flexible regex search across files"},
		{Server: "serena", Name: "list_dir", Description: "List directory contents"},
		{Server: "github", Name: "search_issues", Description: "Search GitHub issues by query"},
		{Server: "github", Name: "get_issue", Description: "Fetch a single GitHub issue by number"},
		{Server: "postgres", Name: "query", Description: "Execute SQL queries"},
	}
}

func TestRankByName(t *testing.T) {
	out := Rank("find symbol", corpus(), 3)
	if len(out) == 0 {
		t.Fatal("no results")
	}
	if out[0].Name != "find_symbol" {
		t.Errorf("top result: got %s want find_symbol", out[0].Name)
	}
}

func TestRankByDescription(t *testing.T) {
	out := Rank("regex pattern", corpus(), 3)
	if len(out) == 0 {
		t.Fatal("no results")
	}
	if out[0].Name != "search_for_pattern" {
		t.Errorf("top: got %s want search_for_pattern", out[0].Name)
	}
}

func TestRankCrossServer(t *testing.T) {
	out := Rank("issue", corpus(), 5)
	if len(out) < 2 {
		t.Fatal("expected >=2 results")
	}
	servers := map[string]bool{}
	for _, r := range out {
		servers[r.Server] = true
	}
	if !servers["github"] {
		t.Error("github tools missing from issue ranking")
	}
}

func TestRankNoMatchReturnsEmpty(t *testing.T) {
	out := Rank("xyzzy", corpus(), 5)
	if len(out) != 0 {
		t.Errorf("expected zero results, got %d", len(out))
	}
}

func TestRankTopKLimit(t *testing.T) {
	out := Rank("symbol", corpus(), 1)
	if len(out) != 1 {
		t.Errorf("topK=1: got %d results", len(out))
	}
}

func TestRankIgnoresStopwords(t *testing.T) {
	a := Rank("the symbol", corpus(), 3)
	b := Rank("symbol", corpus(), 3)
	if len(a) == 0 || len(b) == 0 {
		t.Fatal("no results")
	}
	if a[0].Name != b[0].Name {
		t.Errorf("stopword affected top match: a=%s b=%s", a[0].Name, b[0].Name)
	}
}

func TestCamelSplit(t *testing.T) {
	if got := camelSplit("FindSymbol"); got != "Find Symbol" {
		t.Errorf("got %q", got)
	}
	if got := camelSplit("find_symbol"); got != "find_symbol" {
		t.Errorf("got %q", got)
	}
	if got := camelSplit("HTTPRequest"); got != "HTTPRequest" {
		t.Errorf("HTTPRequest acronym (no boundary detected): got %q", got)
	}
}

func TestTokenize(t *testing.T) {
	cases := map[string][]string{
		"find_symbol":           {"find", "symbol"},
		"FindSymbol":            {"find", "symbol"},
		"search-for-pattern":    {"search", "for", "pattern"},
		"the symbol of":         {"symbol"},
		"":                      nil,
	}
	// "for" is in our stopwords, but I want it to remain because it's domain-relevant.
	// Actually "for" is in the stopwords list. So "search-for-pattern" should be ["search","pattern"].
	cases["search-for-pattern"] = []string{"search", "pattern"}

	for in, want := range cases {
		got := tokenize(in)
		if !sliceEq(got, want) {
			t.Errorf("tokenize(%q): got %v want %v", in, got, want)
		}
	}
}

func sliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
