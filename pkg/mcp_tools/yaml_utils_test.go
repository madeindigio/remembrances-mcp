package mcp_tools

import (
	"strings"
	"testing"
)

func TestTopAlternativesFromCounts(t *testing.T) {
	counts := map[string]int{"b": 2, "a": 3, "c": 3}
	alts := TopAlternativesFromCounts(counts, 2)
	if len(alts) != 2 {
		t.Fatalf("expected 2 alternatives, got %d", len(alts))
	}
	if alts[0] != "a (3)" || alts[1] != "c (3)" {
		t.Fatalf("unexpected ordering: %v", alts)
	}
}

func TestCreateEmptyResultYAML(t *testing.T) {
	msg := "no results"
	alts := []string{"u1 (3)", "u2 (1)"}
	out := CreateEmptyResultYAML(msg, alts)
	if !containsAll(out, []string{"message: no results", "- u1 (3)", "- u2 (1)"}) {
		t.Fatalf("yaml does not contain expected content: %s", out)
	}
}

func containsAll(haystack string, needles []string) bool {
	for _, n := range needles {
		if !strings.Contains(haystack, n) {
			return false
		}
	}
	return true
}
