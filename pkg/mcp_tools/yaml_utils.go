package mcp_tools

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// MarshalYAML converts a Go value into a YAML string. When marshaling fails,
// it returns a human-friendly error string so MCP tools still provide feedback
// instead of silently failing.
func MarshalYAML(data interface{}) string {
	b, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Sprintf("error: failed to marshal to YAML: %v", err)
	}
	return string(b)
}

// CreateEmptyResultYAML builds a standard YAML response for empty results.
// The alternatives slice can be used to suggest available user/project IDs.
func CreateEmptyResultYAML(message string, alternatives []string) string {
	payload := map[string]interface{}{
		"message": message,
	}
	if len(alternatives) > 0 {
		payload["alternatives"] = alternatives
	}
	return MarshalYAML(payload)
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
