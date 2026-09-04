package storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// HybridSearch performs a combined search across vector, graph, and key-value
// stores, all of them already restricted to the caller's visible scopes.
//
// The principal is resolved here as well as in each leg, so a call with no
// identity fails before any query runs rather than three times over.
func (s *SurrealDBStorage) HybridSearch(ctx context.Context, userID string, queryEmbedding []float32, entities []string, limit int) (*HybridSearchResult, error) {
	if _, err := s.scope(ctx, userID); err != nil {
		return nil, err
	}

	start := time.Now()

	vectorResults, err := s.SearchSimilar(ctx, userID, queryEmbedding, limit)
	if err != nil {
		return nil, fmt.Errorf("failed vector search: %w", err)
	}

	var graphResults []GraphResult
	for _, entity := range entities {
		results, err := s.TraverseGraph(ctx, entity, "", 2)
		if err != nil {
			slog.Warn("failed to traverse graph for entity", "entity", entity, "error", err)
			continue
		}
		graphResults = append(graphResults, results...)
	}

	facts, err := s.ListFacts(ctx, userID)
	if err != nil {
		slog.Warn("failed to get facts", "error", err)
		facts = make(map[string]interface{})
	}

	result := &HybridSearchResult{
		VectorResults: vectorResults,
		GraphResults:  graphResults,
		Facts:         facts,
		TotalResults:  len(vectorResults) + len(graphResults) + len(facts),
		QueryTime:     time.Since(start),
	}

	return result, nil
}
