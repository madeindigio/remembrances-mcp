package mcp_tools

import (
	"sort"
	"strings"

	"github.com/agnivade/levenshtein"
)

// SimilarMatch represents a candidate value and its edit distance from the
// query. Lower distances indicate closer matches.
type SimilarMatch struct {
	Value    string `json:"value" toon:"value"`
	Distance int    `json:"distance" toon:"distance"`
}

// normalizeString trims whitespace and lowercases input to make distance
// calculations more forgiving.
func normalizeString(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// LevenshteinDistance returns the normalized edit distance between two strings
// using the agnivade/levenshtein implementation.
func LevenshteinDistance(a, b string) int {
	return levenshtein.ComputeDistance(normalizeString(a), normalizeString(b))
}

// FindSimilarStrings returns candidates whose distance to the query is less
// than or equal to maxDistance (if maxDistance is >= 0). Results are ordered by
// ascending distance, then lexicographically by value for deterministic output.
func FindSimilarStrings(query string, candidates []string, maxDistance int) []SimilarMatch {
	normalizedQuery := normalizeString(query)

	matches := make([]SimilarMatch, 0, len(candidates))
	for _, candidate := range candidates {
		distance := levenshtein.ComputeDistance(normalizedQuery, normalizeString(candidate))
		if maxDistance >= 0 && distance > maxDistance {
			continue
		}
		matches = append(matches, SimilarMatch{Value: candidate, Distance: distance})
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Distance == matches[j].Distance {
			return matches[i].Value < matches[j].Value
		}
		return matches[i].Distance < matches[j].Distance
	})

	return matches
}
