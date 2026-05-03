package cli

import (
	"fmt"

	"github.com/codestz/mcpx/internal/mcp"
)

// nearestTool returns "Did you mean 'X'?" if any tool name is within Levenshtein
// distance 2 of the typo. Empty string otherwise.
func nearestTool(typo string, tools []mcp.Tool) string {
	best, dist := "", 99
	for _, t := range tools {
		d := levenshtein(typo, t.Name)
		if d < dist {
			best, dist = t.Name, d
		}
	}
	if dist <= 2 && best != "" {
		return fmt.Sprintf("Did you mean %q?", best)
	}
	return ""
}

// nearestProperty suggests a flag name when an unknown one is given.
func nearestProperty(typo string, props map[string]mcp.PropertySchema) string {
	best, dist := "", 99
	for name := range props {
		d := levenshtein(typo, name)
		if d < dist {
			best, dist = name, d
		}
	}
	if dist <= 2 && best != "" {
		return fmt.Sprintf("did you mean --%s?", best)
	}
	return ""
}

// levenshtein returns the edit distance between a and b.
// Iterative two-row implementation; O(len(a)*len(b)) time, O(len(b)) space.
func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			curr[j] = min3(del, ins, sub)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func min3(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}
	return c
}
