// Package storage provides code indexing storage operations for SurrealDB.
// This file contains symbol-related operations for code indexing.
package storage

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/madeindigio/remembrances-mcp/v2/pkg/treesitter"
)

// ===== SYMBOL OPERATIONS =====

// SaveCodeSymbol saves or updates a code symbol
func (s *SurrealDBStorage) SaveCodeSymbol(ctx context.Context, symbol *treesitter.CodeSymbol) error {
	return s.withTxnRetry(ctx, func(ctx context.Context) error {
		return s.saveCodeSymbolAttempt(ctx, symbol)
	})
}

func (s *SurrealDBStorage) saveCodeSymbolAttempt(ctx context.Context, symbol *treesitter.CodeSymbol) error {
	t, err := s.scope(ctx, "")
	if err != nil {
		return err
	}

	// Check if the symbol is already indexed in one of the caller's scopes.
	existsQuery := fmt.Sprintf("SELECT id FROM code_symbols %s",
		t.where("project_id = $project_id", "name_path = $name_path"))
	existsResult, err := s.query(ctx, existsQuery, t.params(map[string]interface{}{
		"project_id": symbol.ProjectID,
		"name_path":  symbol.NamePath,
	}))
	if err != nil {
		return fmt.Errorf("failed to check existing symbol: %w", err)
	}

	isNewSymbol := true
	if existsResult != nil && len(*existsResult) > 0 {
		queryResult := (*existsResult)[0]
		if queryResult.Status == "OK" && len(queryResult.Result) > 0 {
			isNewSymbol = false
		}
	}

	// Build params with only non-empty optional fields
	params := map[string]interface{}{
		"project_id":  symbol.ProjectID,
		"file_path":   symbol.FilePath,
		"language":    string(symbol.Language),
		"symbol_type": string(symbol.SymbolType),
		"name":        symbol.Name,
		"name_path":   symbol.NamePath,
		"start_line":  symbol.StartLine,
		"end_line":    symbol.EndLine,
		"start_byte":  symbol.StartByte,
		"end_byte":    symbol.EndByte,
	}

	// Only add optional fields if they have values
	if symbol.SourceCode != "" {
		params["source_code"] = symbol.SourceCode
	}
	if symbol.Signature != "" {
		params["signature"] = symbol.Signature
	}
	if symbol.DocString != "" {
		params["doc_string"] = symbol.DocString
	}
	if len(symbol.Embedding) > 0 {
		params["embedding"] = normalizeEmbedding(symbol.Embedding)
	}
	if symbol.ParentID != nil && *symbol.ParentID != "" {
		params["parent_id"] = *symbol.ParentID
	}
	if symbol.Metadata != nil {
		params["metadata"] = symbol.Metadata
	}

	// The field list is built from the params map, whose key set varies with
	// which optional columns the symbol carries. Sorting it keeps the emitted
	// SurrealQL deterministic, which matters for reading a query log.
	fields := make([]string, 0, len(params))
	for k := range params {
		fields = append(fields, k)
	}
	sort.Strings(fields)

	scoped := t.params(params)

	if isNewSymbol {
		assignments := make([]string, 0, len(fields))
		for _, k := range fields {
			assignments = append(assignments, fmt.Sprintf("%s: $%s", k, k))
		}
		query := fmt.Sprintf("CREATE code_symbols CONTENT {\n\t\t\t%s%s\n\t\t}",
			t.contentFields(), strings.Join(assignments, ",\n\t\t\t"))

		if _, err := s.query(ctx, query, scoped); err != nil {
			return fmt.Errorf("failed to create symbol: %w", err)
		}
	} else {
		// Exclude the identity columns: an update must not move the row.
		assignments := make([]string, 0, len(fields))
		for _, k := range fields {
			if k == "project_id" || k == "name_path" {
				continue
			}
			assignments = append(assignments, fmt.Sprintf("%s = $%s", k, k))
		}

		query := fmt.Sprintf(`
			UPDATE code_symbols SET
				%s%s
			%s
		`, t.updateSet(), strings.Join(assignments, ", "),
			t.where("project_id = $project_id", "name_path = $name_path"))

		if _, err := s.query(ctx, query, scoped); err != nil {
			return fmt.Errorf("failed to update symbol: %w", err)
		}
	}

	return nil
}

// SaveCodeSymbols saves multiple symbols in batch
func (s *SurrealDBStorage) SaveCodeSymbols(ctx context.Context, symbols []*treesitter.CodeSymbol) error {
	// Resolved up front so an empty batch with no principal still fails closed.
	if _, err := s.scope(ctx, ""); err != nil {
		return err
	}
	for _, symbol := range symbols {
		if err := s.SaveCodeSymbol(ctx, symbol); err != nil {
			return fmt.Errorf("failed to save symbol %s: %w", symbol.NamePath, err)
		}
	}
	return nil
}

// GetCodeSymbol retrieves a symbol by project and name path
func (s *SurrealDBStorage) GetCodeSymbol(ctx context.Context, projectID, namePath string) (*CodeSymbol, error) {
	t, err := s.scope(ctx, "")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM code_symbols %s LIMIT 1;",
		t.where("project_id = $project_id", "name_path = $name_path"))
	params := t.params(map[string]interface{}{
		"project_id": projectID,
		"name_path":  namePath,
	})

	result, err := s.query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get symbol: %w", err)
	}

	symbols, err := decodeResult[CodeSymbol](result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode symbol: %w", err)
	}

	if len(symbols) == 0 {
		return nil, nil
	}

	return &symbols[0], nil
}

// FindSymbolsByName finds symbols by name (partial or exact match)
func (s *SurrealDBStorage) FindSymbolsByName(ctx context.Context, projectID, name string, symbolTypes []treesitter.SymbolType, limit int) ([]CodeSymbol, error) {
	t, err := s.scope(ctx, "")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM code_symbols %s",
		t.where("project_id = $project_id", "name CONTAINS $name"))

	params := t.params(map[string]interface{}{
		"project_id": projectID,
		"name":       name,
	})

	if len(symbolTypes) > 0 {
		types := make([]string, len(symbolTypes))
		for i, t := range symbolTypes {
			types[i] = string(t)
		}
		query += ` AND symbol_type IN $types`
		params["types"] = types
	}

	query += fmt.Sprintf(` LIMIT %d;`, limit)

	result, err := s.query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find symbols: %w", err)
	}

	return decodeResult[CodeSymbol](result)
}

// FindSymbolsByFile retrieves all symbols in a file
func (s *SurrealDBStorage) FindSymbolsByFile(ctx context.Context, projectID, filePath string) ([]CodeSymbol, error) {
	t, err := s.scope(ctx, "")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM code_symbols %s ORDER BY start_line ASC;",
		t.where("project_id = $project_id", "file_path = $file_path"))
	params := t.params(map[string]interface{}{
		"project_id": projectID,
		"file_path":  filePath,
	})

	result, err := s.query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find symbols: %w", err)
	}

	return decodeResult[CodeSymbol](result)
}

// FindChildSymbols retrieves child symbols of a parent
func (s *SurrealDBStorage) FindChildSymbols(ctx context.Context, projectID, parentID string) ([]CodeSymbol, error) {
	t, err := s.scope(ctx, "")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM code_symbols %s ORDER BY start_line ASC;",
		t.where("project_id = $project_id", "parent_id = $parent_id"))
	params := t.params(map[string]interface{}{
		"project_id": projectID,
		"parent_id":  parentID,
	})

	result, err := s.query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find child symbols: %w", err)
	}

	return decodeResult[CodeSymbol](result)
}

// SearchSymbolsBySimilarity performs semantic search on code symbols
func (s *SurrealDBStorage) SearchSymbolsBySimilarity(ctx context.Context, projectID string, queryEmbedding []float32, symbolTypes []treesitter.SymbolType, limit int) ([]CodeSymbolSearchResult, error) {
	t, err := s.scope(ctx, "")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		SELECT *, vector::similarity::cosine(embedding, $embedding) AS similarity
		FROM code_symbols
		%s
	`, t.where("project_id = $project_id", "embedding != NONE"))

	params := t.params(map[string]interface{}{
		"project_id": projectID,
		"embedding":  normalizeEmbedding(queryEmbedding),
	})

	if len(symbolTypes) > 0 {
		types := make([]string, len(symbolTypes))
		for i, t := range symbolTypes {
			types[i] = string(t)
		}
		query += ` AND symbol_type IN $types`
		params["types"] = types
	}

	query += fmt.Sprintf(` ORDER BY similarity DESC LIMIT %d;`, limit)

	result, err := s.query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search symbols: %w", err)
	}

	// Decode results with similarity
	type symbolWithSimilarity struct {
		CodeSymbol
		Similarity float64 `json:"similarity"`
	}

	symbols, err := decodeResult[symbolWithSimilarity](result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	results := make([]CodeSymbolSearchResult, len(symbols))
	for i, s := range symbols {
		sym := s.CodeSymbol
		results[i] = CodeSymbolSearchResult{
			Symbol:     &sym,
			Similarity: s.Similarity,
		}
	}

	return results, nil
}

// DeleteSymbolsByFile deletes all symbols in a file
func (s *SurrealDBStorage) DeleteSymbolsByFile(ctx context.Context, projectID, filePath string) error {
	t, err := s.scope(ctx, "")
	if err != nil {
		return err
	}

	query := fmt.Sprintf("DELETE FROM code_symbols %s;",
		t.whereAny("project_id = $project_id", "file_path = $file_path"))
	params := t.params(map[string]interface{}{
		"project_id": projectID,
		"file_path":  filePath,
	})

	_, err = s.query(ctx, query, params)
	return err
}
