package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/surrealdb/surrealdb.go"
)

// QueryResult mimics the structure returned by surrealdb.Query
type QueryResult struct {
	Status string                   `json:"status"`
	Time   string                   `json:"time,omitempty"`
	Result []map[string]interface{} `json:"result"`
}

// Query is a public method to execute custom queries (used by tools that need direct database access)
func (s *SurrealDBStorage) Query(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error) {
	results, err := s.query(ctx, query, params)
	if err != nil {
		return nil, err
	}

	// Flatten all results into a single slice
	allResults := make([]map[string]interface{}, 0)
	if results != nil {
		for _, queryResult := range *results {
			if queryResult.Status == "OK" {
				allResults = append(allResults, queryResult.Result...)
			}
		}
	}

	// Normalize SurrealDB datetime objects to ISO strings
	normalized := normalizeSurrealDBDatetimes(allResults)
	if normalizedMaps, ok := normalized.([]map[string]interface{}); ok {
		return normalizedMaps, nil
	}

	return allResults, nil
}

// query executes a query on either embedded or remote backend
func (s *SurrealDBStorage) query(ctx context.Context, query string, params map[string]interface{}) (*[]QueryResult, error) {
	if s.useEmbedded {
		return s.queryEmbedded(ctx, query, params)
	}
	return s.queryRemote(ctx, query, params)
}

// queryEmbedded executes a query on the embedded backend
func (s *SurrealDBStorage) queryEmbedded(ctx context.Context, query string, params map[string]interface{}) (*[]QueryResult, error) {
	if s.embeddedDB == nil {
		return nil, fmt.Errorf("embedded database not initialized")
	}

	results, err := s.embeddedDB.Query(query, params)
	if err != nil {
		return nil, err
	}

	// Convert embedded results to QueryResult format
	queryResults := make([]QueryResult, 0)

	// The embedded DB returns []interface{}, we need to convert to the expected format
	if len(results) > 0 {
		// First, check if we have multiple results that are all maps
		// This is the common case for SELECT queries returning multiple rows
		allMaps := true
		for _, r := range results {
			if _, ok := r.(map[string]interface{}); !ok {
				allMaps = false
				break
			}
		}

		if allMaps && len(results) > 0 {
			// All results are maps - collect them all into one QueryResult
			maps := make([]map[string]interface{}, 0, len(results))
			for _, r := range results {
				if m, ok := r.(map[string]interface{}); ok {
					maps = append(maps, m)
				}
			}
			queryResults = append(queryResults, QueryResult{
				Status: "OK",
				Result: maps,
			})
		} else {
			// Try to determine if this is a single result or array of results
			switch v := results[0].(type) {
			case []interface{}:
				// Array of results (e.g., from a single SELECT that returns an array)
				maps := make([]map[string]interface{}, 0, len(v))
				for _, item := range v {
					if m, ok := item.(map[string]interface{}); ok {
						maps = append(maps, m)
					}
				}
				queryResults = append(queryResults, QueryResult{
					Status: "OK",
					Result: maps,
				})
			default:
				// Try to convert all results
				maps := make([]map[string]interface{}, 0)
				for _, result := range results {
					if m, ok := result.(map[string]interface{}); ok {
						maps = append(maps, m)
					} else if arr, ok := result.([]interface{}); ok {
						for _, item := range arr {
							if m, ok := item.(map[string]interface{}); ok {
								maps = append(maps, m)
							}
						}
					}
				}
				queryResults = append(queryResults, QueryResult{
					Status: "OK",
					Result: maps,
				})
			}
		}
	} else {
		// Empty result
		queryResults = append(queryResults, QueryResult{
			Status: "OK",
			Result: []map[string]interface{}{},
		})
	}

	return &queryResults, nil
}

// queryRemote executes a query on the remote backend
func (s *SurrealDBStorage) queryRemote(ctx context.Context, query string, params map[string]interface{}) (*[]QueryResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("remote database not initialized")
	}

	result, err := surrealdb.Query[[]map[string]interface{}](ctx, s.db, query, params)
	if err != nil {
		return nil, err
	}

	// Convert surrealdb.QueryResult to our QueryResult format
	queryResults := make([]QueryResult, 0)
	if result != nil {
		for _, qr := range *result {
			queryResults = append(queryResults, QueryResult{
				Status: qr.Status,
				Time:   qr.Time,
				Result: qr.Result,
			})
		}
	}

	return &queryResults, nil
}

// create creates a record on either embedded or remote backend
func (s *SurrealDBStorage) create(ctx context.Context, resource string, data interface{}) (interface{}, error) {
	if s.useEmbedded {
		if s.embeddedDB == nil {
			return nil, fmt.Errorf("embedded database not initialized")
		}
		return s.embeddedDB.Create(resource, data)
	}

	if s.db == nil {
		return nil, fmt.Errorf("remote database not initialized")
	}
	return surrealdb.Create[map[string]interface{}](ctx, s.db, resource, data)
}

// update updates a record on either embedded or remote backend
func (s *SurrealDBStorage) update(ctx context.Context, resource string, data interface{}) (interface{}, error) {
	if s.useEmbedded {
		if s.embeddedDB == nil {
			return nil, fmt.Errorf("embedded database not initialized")
		}
		return s.embeddedDB.Update(resource, data)
	}

	if s.db == nil {
		return nil, fmt.Errorf("remote database not initialized")
	}
	return surrealdb.Update[map[string]interface{}](ctx, s.db, resource, data)
}

// delete deletes a record on either embedded or remote backend
func (s *SurrealDBStorage) delete(ctx context.Context, resource string) (interface{}, error) {
	if s.useEmbedded {
		if s.embeddedDB == nil {
			return nil, fmt.Errorf("embedded database not initialized")
		}
		return s.embeddedDB.Delete(resource)
	}

	if s.db == nil {
		return nil, fmt.Errorf("remote database not initialized")
	}
	return surrealdb.Delete[map[string]interface{}](ctx, s.db, resource)
}

// unmarshalResult helps unmarshal results consistently
func unmarshalResult(result interface{}, target interface{}) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return nil
}
