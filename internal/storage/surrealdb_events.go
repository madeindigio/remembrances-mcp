package storage

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Event represents a temporal event with semantic search support
type Event struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Subject   string                 `json:"subject"`
	Content   string                 `json:"content"`
	Embedding []float32              `json:"embedding,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// EventSearchParams defines parameters for searching events
type EventSearchParams struct {
	UserID     string     // Required: project/user identifier
	Subject    string     // Optional: filter by subject
	Query      string     // Optional: text query for BM25 search
	Embedding  []float32  // Optional: query embedding for vector search
	FromDate   *time.Time // Optional: start date
	ToDate     *time.Time // Optional: end date
	LastHours  *int       // Optional: last N hours
	LastDays   *int       // Optional: last N days
	LastMonths *int       // Optional: last N months
	Limit      int        // Max results (default 50)
}

// EventSearchResult represents a search result with relevance score
type EventSearchResult struct {
	Event     Event   `json:"event"`
	Relevance float64 `json:"relevance"`
}

// SaveEvent stores a new event with embedding for semantic search
func (s *SurrealDBStorage) SaveEvent(ctx context.Context, userID, subject, content string, embedding []float32, metadata map[string]interface{}) (string, time.Time, error) {
	if metadata == nil {
		metadata = map[string]interface{}{}
	}

	// Normalize embedding length to the MTREE dimension
	if embedding == nil {
		embedding = make([]float32, defaultMtreeDim)
	} else if len(embedding) != defaultMtreeDim {
		norm := make([]float32, defaultMtreeDim)
		copy(norm, embedding)
		embedding = norm
	}

	// Convert embedding to []float64 for SurrealDB JSON consistency
	emb64 := make([]float64, len(embedding))
	for i, v := range embedding {
		emb64[i] = float64(v)
	}

	query := `
		INSERT INTO events {
			user_id: $user_id,
			subject: $subject,
			content: $content,
			embedding: $embedding,
			metadata: $metadata,
			created_at: time::now()
		} RETURN id, created_at
	`
	params := map[string]interface{}{
		"user_id":   userID,
		"subject":   subject,
		"content":   content,
		"embedding": emb64,
		"metadata":  metadata,
	}

	result, err := s.query(ctx, query, params)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to save event: %w", err)
	}

	// Extract ID and created_at from result
	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		if queryResult.Status == "OK" && len(queryResult.Result) > 0 {
			rec := queryResult.Result[0]
			var eventID string
			var createdAt time.Time

			if id, ok := rec["id"]; ok {
				eventID = fmt.Sprintf("%v", id)
			}
			if cat, ok := rec["created_at"]; ok {
				if catStr, ok := cat.(string); ok {
					createdAt, _ = time.Parse(time.RFC3339Nano, catStr)
				}
			}
			if createdAt.IsZero() {
				createdAt = time.Now()
			}

			log.Printf("Event saved: id=%s, subject=%s, user_id=%s", eventID, subject, userID)
			return eventID, createdAt, nil
		}
	}

	return "", time.Time{}, fmt.Errorf("failed to save event: no result returned")
}

// SearchEvents performs hybrid search on events with temporal filtering
func (s *SurrealDBStorage) SearchEvents(ctx context.Context, params EventSearchParams) ([]EventSearchResult, error) {
	if params.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	if params.Limit <= 0 {
		params.Limit = 50
	}

	// Calculate date filters from relative time offsets
	var fromDate, toDate *time.Time
	now := time.Now()

	if params.LastHours != nil && *params.LastHours > 0 {
		t := now.Add(-time.Duration(*params.LastHours) * time.Hour)
		fromDate = &t
	} else if params.LastDays != nil && *params.LastDays > 0 {
		t := now.AddDate(0, 0, -*params.LastDays)
		fromDate = &t
	} else if params.LastMonths != nil && *params.LastMonths > 0 {
		t := now.AddDate(0, -*params.LastMonths, 0)
		fromDate = &t
	}

	if params.FromDate != nil {
		fromDate = params.FromDate
	}
	if params.ToDate != nil {
		toDate = params.ToDate
	}

	// Determine search mode
	hasQuery := params.Query != "" && params.Embedding != nil
	hasTextOnly := params.Query != "" && params.Embedding == nil
	hasVectorOnly := params.Embedding != nil && params.Query == ""

	var query string
	queryParams := map[string]interface{}{
		"user_id": params.UserID,
		"limit":   params.Limit,
	}

	// Build WHERE conditions
	conditions := []string{"user_id = $user_id"}

	if params.Subject != "" {
		conditions = append(conditions, "subject = $subject")
		queryParams["subject"] = params.Subject
	}

	if fromDate != nil {
		conditions = append(conditions, "created_at >= $from_date")
		queryParams["from_date"] = fromDate.Format(time.RFC3339Nano)
	}

	if toDate != nil {
		conditions = append(conditions, "created_at <= $to_date")
		queryParams["to_date"] = toDate.Format(time.RFC3339Nano)
	}

	whereClause := ""
	for i, cond := range conditions {
		if i == 0 {
			whereClause = "WHERE " + cond
		} else {
			whereClause += " AND " + cond
		}
	}

	if hasQuery {
		// Hybrid search: combine BM25 text score and vector similarity
		emb64 := make([]float64, len(params.Embedding))
		for i, v := range params.Embedding {
			emb64[i] = float64(v)
		}
		queryParams["query_embedding"] = emb64
		queryParams["text_query"] = params.Query

		query = fmt.Sprintf(`
			SELECT 
				id, user_id, subject, content, metadata, created_at,
				(search::score(1) * 0.5 + vector::similarity::cosine(embedding, $query_embedding) * 0.5) AS relevance
			FROM events
			%s
			AND (content @1@ $text_query OR vector::similarity::cosine(embedding, $query_embedding) > 0.3)
			ORDER BY relevance DESC
			LIMIT $limit
		`, whereClause)
	} else if hasTextOnly {
		// Text-only search using BM25
		queryParams["text_query"] = params.Query

		query = fmt.Sprintf(`
			SELECT 
				id, user_id, subject, content, metadata, created_at,
				search::score(1) AS relevance
			FROM events
			%s
			AND content @1@ $text_query
			ORDER BY relevance DESC
			LIMIT $limit
		`, whereClause)
	} else if hasVectorOnly {
		// Vector-only search
		emb64 := make([]float64, len(params.Embedding))
		for i, v := range params.Embedding {
			emb64[i] = float64(v)
		}
		queryParams["query_embedding"] = emb64

		query = fmt.Sprintf(`
			SELECT 
				id, user_id, subject, content, metadata, created_at,
				vector::similarity::cosine(embedding, $query_embedding) AS relevance
			FROM events
			%s
			ORDER BY relevance DESC
			LIMIT $limit
		`, whereClause)
	} else {
		// No search query - just filter and order by recency
		query = fmt.Sprintf(`
			SELECT 
				id, user_id, subject, content, metadata, created_at,
				1.0 AS relevance
			FROM events
			%s
			ORDER BY created_at DESC
			LIMIT $limit
		`, whereClause)
	}

	result, err := s.query(ctx, query, queryParams)
	if err != nil {
		return nil, fmt.Errorf("failed to search events: %w", err)
	}

	return s.parseEventResults(result)
}

// parseEventResults converts query results to EventSearchResult slice
func (s *SurrealDBStorage) parseEventResults(result *[]QueryResult) ([]EventSearchResult, error) {
	if result == nil || len(*result) == 0 {
		return []EventSearchResult{}, nil
	}

	var results []EventSearchResult
	for _, qr := range *result {
		if qr.Status != "OK" {
			continue
		}
		for _, rec := range qr.Result {
			event := Event{}

			if id, ok := rec["id"]; ok {
				event.ID = fmt.Sprintf("%v", id)
			}
			if userID, ok := rec["user_id"].(string); ok {
				event.UserID = userID
			}
			if subject, ok := rec["subject"].(string); ok {
				event.Subject = subject
			}
			if content, ok := rec["content"].(string); ok {
				event.Content = content
			}
			if metadata, ok := rec["metadata"].(map[string]interface{}); ok {
				event.Metadata = metadata
			}
			if createdAt, ok := rec["created_at"].(string); ok {
				event.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
			}

			var relevance float64 = 1.0
			if rel, ok := rec["relevance"].(float64); ok {
				relevance = rel
			}

			results = append(results, EventSearchResult{
				Event:     event,
				Relevance: relevance,
			})
		}
	}

	return results, nil
}

// DeleteEvent deletes an event by ID
func (s *SurrealDBStorage) DeleteEvent(ctx context.Context, eventID, userID string) error {
	query := `DELETE FROM events WHERE id = $id AND user_id = $user_id`
	params := map[string]interface{}{
		"id":      eventID,
		"user_id": userID,
	}

	_, err := s.query(ctx, query, params)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	return nil
}

// GetEventsBySubject retrieves all events for a subject (useful for conversation history)
func (s *SurrealDBStorage) GetEventsBySubject(ctx context.Context, userID, subject string, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, user_id, subject, content, metadata, created_at
		FROM events
		WHERE user_id = $user_id AND subject = $subject
		ORDER BY created_at ASC
		LIMIT $limit
	`
	params := map[string]interface{}{
		"user_id": userID,
		"subject": subject,
		"limit":   limit,
	}

	result, err := s.query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get events by subject: %w", err)
	}

	searchResults, err := s.parseEventResults(result)
	if err != nil {
		return nil, err
	}

	events := make([]Event, len(searchResults))
	for i, sr := range searchResults {
		events[i] = sr.Event
	}

	return events, nil
}
