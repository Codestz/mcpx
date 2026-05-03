// Package find ranks MCP tools across all configured servers by relevance to
// a free-text query. Built for the AI-agent discovery moment: instead of
// scanning `mcpx list -v` (5–15K tokens), `mcpx find "search code"` returns
// 3 ranked candidates in ~80 tokens.
//
// Algorithm: BM25 over (tool name + description), with a name-match bonus and
// snake_case/camelCase splitting so "find symbol" matches "find_symbol".
package find

import (
	"math"
	"regexp"
	"sort"
	"strings"
)

// Tool is the input record for ranking. Server distinguishes tools with the
// same name across multiple servers.
type Tool struct {
	Server      string
	Name        string
	Description string
}

// Result is a ranked match.
type Result struct {
	Server      string
	Name        string
	Description string
	Score       float64
}

// Rank scores tools against query and returns the top-K matches by score.
// topK <= 0 returns all matches with score > 0.
func Rank(query string, tools []Tool, topK int) []Result {
	qTokens := tokenize(query)
	if len(qTokens) == 0 || len(tools) == 0 {
		return nil
	}

	// Pre-tokenize each tool's name and description.
	type doc struct {
		idx       int
		nameToks  []string
		descToks  []string
		totalLen  int
	}
	docs := make([]doc, len(tools))
	totalLen := 0
	for i, t := range tools {
		nt := tokenize(t.Name)
		dt := tokenize(t.Description)
		docs[i] = doc{idx: i, nameToks: nt, descToks: dt, totalLen: len(nt) + len(dt)}
		totalLen += len(nt) + len(dt)
	}
	avgLen := float64(totalLen) / float64(len(docs))
	if avgLen == 0 {
		avgLen = 1
	}

	// Document frequency per query token.
	df := map[string]int{}
	for _, q := range qTokens {
		for i := range docs {
			if contains(docs[i].nameToks, q) || contains(docs[i].descToks, q) {
				df[q]++
			}
		}
	}

	qSet := map[string]bool{}
	for _, q := range qTokens {
		qSet[q] = true
	}

	const k1, b float64 = 1.5, 0.75
	results := make([]Result, 0, len(docs))
	for _, d := range docs {
		score := 0.0
		nameMatches := 0
		for _, q := range qTokens {
			n := df[q]
			if n == 0 {
				continue
			}
			idf := math.Log(1 + (float64(len(docs))-float64(n)+0.5)/(float64(n)+0.5))

			nameTF := count(d.nameToks, q)
			descTF := count(d.descToks, q)
			tf := float64(descTF) + 5.0*float64(nameTF) // name tokens weigh much more

			if tf == 0 {
				continue
			}
			if nameTF > 0 {
				nameMatches++
			}
			norm := tf * (k1 + 1) / (tf + k1*(1-b+b*float64(d.totalLen)/avgLen))
			score += idf * norm
		}

		// Tight-match bonus: tools whose name covers the entire query rank above
		// tools that merely contain the query tokens among many extras.
		if nameMatches == len(qTokens) {
			extras := 0
			for _, t := range d.nameToks {
				if !qSet[t] {
					extras++
				}
			}
			score *= 1.0 + 0.5/float64(1+extras) // 1.5x at extras=0, 1.25x at extras=1, ...
		}

		if score > 0 {
			t := tools[d.idx]
			results = append(results, Result{
				Server: t.Server, Name: t.Name, Description: t.Description, Score: score,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		// Stable tiebreak: server.tool ascending.
		return results[i].Server+"."+results[i].Name < results[j].Server+"."+results[j].Name
	})

	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}
	return results
}

// tokenize lowercases input and splits on non-alphanumeric chars and case
// boundaries. "find_symbol" → ["find","symbol"]; "FindSymbol" → ["find","symbol"].
// Trailing-s plurals collapse so "issue" matches "issues".
func tokenize(s string) []string {
	if s == "" {
		return nil
	}
	s = camelSplit(s)
	parts := splitWords(strings.ToLower(s))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" || stopwords[p] {
			continue
		}
		out = append(out, stem(p))
	}
	return out
}

// stem collapses trivial English plurals: "issues"→"issue", "files"→"file",
// but preserves "class"→"class" (double s) and "is"→"is" (too short).
func stem(s string) string {
	if len(s) <= 3 {
		return s
	}
	if !strings.HasSuffix(s, "s") {
		return s
	}
	if strings.HasSuffix(s, "ss") || strings.HasSuffix(s, "us") || strings.HasSuffix(s, "is") {
		return s
	}
	return s[:len(s)-1]
}

var camelRE = regexp.MustCompile(`([a-z0-9])([A-Z])`)

func camelSplit(s string) string {
	return camelRE.ReplaceAllString(s, "$1 $2")
}

func splitWords(s string) []string {
	var out []string
	start := -1
	for i, r := range s {
		isWord := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isWord {
			if start < 0 {
				start = i
			}
		} else {
			if start >= 0 {
				out = append(out, s[start:i])
				start = -1
			}
		}
	}
	if start >= 0 {
		out = append(out, s[start:])
	}
	return out
}

func contains(toks []string, q string) bool {
	for _, t := range toks {
		if t == q {
			return true
		}
	}
	return false
}

func count(toks []string, q string) int {
	n := 0
	for _, t := range toks {
		if t == q {
			n++
		}
	}
	return n
}

// stopwords are skipped in queries and corpora — they don't help discrimination.
var stopwords = map[string]bool{
	"the": true, "a": true, "an": true, "and": true, "or": true, "to": true,
	"of": true, "in": true, "on": true, "for": true, "with": true, "by": true,
	"is": true, "be": true, "are": true, "this": true, "that": true,
}
