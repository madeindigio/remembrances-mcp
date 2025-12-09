package mcp_tools

import (
	"fmt"
	"sort"

	"github.com/toon-format/toon-go"
)

// MarshalTOON converts a Go value into a TOON string. On failure, it returns a
// human-friendly error string so MCP tools still provide feedback instead of
// silently failing.
func MarshalTOON(data interface{}) string {
	out, err := toon.MarshalString(data, toon.WithLengthMarkers(true))
	if err != nil {
		return fmt.Sprintf("error: failed to marshal to TOON: %v", err)
	}
	return out
}

// MarshalYAML is kept for backward compatibility with existing handlers while
// responses transition to TOON. It delegates to MarshalTOON so all MCP outputs
// use TOON format going forward.
func MarshalYAML(data interface{}) string {
	return MarshalTOON(data)
}

// CreateEmptyResultTOON builds a standard TOON response for empty results. The
// alternatives slice can be used to suggest available user/project IDs.
func CreateEmptyResultTOON(message string, suggestions AlternativeSuggestions) string {
	payload := map[string]interface{}{
		"message": message,
	}

	if len(suggestions.SimilarNames) > 0 {
		payload["did_you_mean"] = suggestions.SimilarNames
	}
	if len(suggestions.OtherIDs) > 0 {
		payload["available_ids"] = suggestions.OtherIDs
	}

	return MarshalTOON(payload)
}

// CreateEmptyResultYAML is maintained to avoid breaking existing handlers. It
// now produces TOON output by delegating to CreateEmptyResultTOON.
func CreateEmptyResultYAML(message string, alternatives []string) string {
	return CreateEmptyResultTOON(message, AlternativeSuggestions{OtherIDs: alternatives})
}

// TopAlternativesFromCounts converts a map of counts into a sorted list of
// "id (count)" strings limited to the provided size. It sorts by count
// descending and then by key for deterministic output.
func TopAlternativesFromCounts(counts map[string]int, limit int) []string {
	type kv struct {
		Key   string
		Count int
	}

	items := make([]kv, 0, len(counts))
	for k, v := range counts {
		items = append(items, kv{Key: k, Count: v})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Key < items[j].Key
		}
		return items[i].Count > items[j].Count
	})

	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}

	alternatives := make([]string, 0, limit)
	for idx := 0; idx < limit; idx++ {
		alternatives = append(alternatives, fmt.Sprintf("%s (%d)", items[idx].Key, items[idx].Count))
	}
	return alternatives
}
