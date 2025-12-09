package mcp_tools

import "testing"

func TestLevenshteinDistanceNormalization(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"Test", "test", 0},
		{" Test ", "test", 0},
		{"kitten", "sitting", 3},
		{"", "", 0},
		{"a", "", 1},
	}

	for _, tc := range tests {
		if got := LevenshteinDistance(tc.a, tc.b); got != tc.expected {
			t.Fatalf("distance(%q,%q)=%d, want %d", tc.a, tc.b, got, tc.expected)
		}
	}
}

func TestFindSimilarStrings(t *testing.T) {
	candidates := []string{"preferences", "Preference", "other"}
	matches := FindSimilarStrings("Preferencias", candidates, 3)

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}

	if matches[0].Value != "preferences" || matches[0].Distance != 2 {
		t.Fatalf("unexpected first match: %+v", matches[0])
	}

	if matches[1].Value != "Preference" || matches[1].Distance != 3 {
		t.Fatalf("unexpected second match: %+v", matches[1])
	}
}

func TestFindSimilarStringsMaxDistance(t *testing.T) {
	candidates := []string{"alpha", "beta", "gamma"}
	matches := FindSimilarStrings("omega", candidates, 2)
	if len(matches) != 0 {
		t.Fatalf("expected no matches, got %v", matches)
	}
}
