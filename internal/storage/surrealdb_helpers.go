package storage

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// decodeResult is a generic function to decode query results into typed slices
func decodeResult[T any](result *[]QueryResult) ([]T, error) {
	if result == nil || len(*result) == 0 {
		return nil, nil
	}

	queryResult := (*result)[0]
	if queryResult.Status != "OK" {
		return nil, fmt.Errorf("query failed: %s", queryResult.Status)
	}

	if queryResult.Result == nil || len(queryResult.Result) == 0 {
		return nil, nil
	}

	// Debug: log the raw result before normalization
	if len(queryResult.Result) > 0 {
		firstItem := queryResult.Result[0]
		if idVal, hasID := firstItem["id"]; hasID {
			slog.Debug("decodeResult: raw id value", "type", fmt.Sprintf("%T", idVal), "value", idVal)
		}
	}

	// Pre-process results to convert SurrealDB datetime objects to ISO strings
	processedResult := normalizeSurrealDBDatetimes(queryResult.Result)

	// Marshal to JSON and unmarshal to typed slice
	jsonData, err := json.Marshal(processedResult)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	// Debug logging
	slog.Debug("decodeResult: normalized JSON", "json", string(jsonData))

	var items []T
	if err := json.Unmarshal(jsonData, &items); err != nil {
		slog.Error("decodeResult: unmarshal failed", "error", err, "json", string(jsonData))
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return items, nil
}

// normalizeSurrealDBDatetimes recursively converts SurrealDB datetime objects
// from {"Datetime": "2025-..."} format to plain ISO8601 strings for proper JSON unmarshaling.
// Also normalizes SurrealDB record IDs from {"id": "xxx", "tb": "table"} to "table:xxx".
func normalizeSurrealDBDatetimes(data interface{}) interface{} {
	// Log the type we're processing for debugging
	dataType := fmt.Sprintf("%T", data)
	if strings.Contains(dataType, "RecordID") {
		slog.Debug("NORMALIZE: processing RecordID type", "type", dataType, "value", data)
		
		// Direct check for RecordID struct - it has Table and ID fields
		val := reflect.ValueOf(data)
		if val.Kind() == reflect.Struct {
			typ := val.Type()
			
			// Try to get Table and ID fields
			var tableField, idField reflect.Value
			for i := 0; i < val.NumField(); i++ {
				fieldName := typ.Field(i).Name
				if fieldName == "Table" {
					tableField = val.Field(i)
				}
				if fieldName == "ID" {
					idField = val.Field(i)
				}
			}
			
			// If we have both fields, construct the string representation
			if tableField.IsValid() && idField.IsValid() {
				tableStr := fmt.Sprintf("%v", tableField.Interface())
				idStr := fmt.Sprintf("%v", idField.Interface())
				result := tableStr + ":" + idStr
				slog.Debug("NORMALIZE: Converting RecordID", "from", data, "to", result)
				return result
			}
		}
	}
	
	switch v := data.(type) {
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = normalizeSurrealDBDatetimes(item)
		}
		return result
	case []map[string]interface{}:
		// Handle slices of maps (common return type from Query)
		result := make([]map[string]interface{}, len(v))
		for i, item := range v {
			normalized := normalizeSurrealDBDatetimes(item)
			if m, ok := normalized.(map[string]interface{}); ok {
				result[i] = m
			} else {
				result[i] = item
			}
		}
		return result
	case map[string]interface{}:
		// Log what we're processing
		if _, hasTable := v["Table"]; hasTable {
			if _, hasID := v["ID"]; hasID {
				slog.Debug("NORMALIZE: Processing map with Table and ID", "map", v, "len", len(v))
			}
		}
		
		// Check if this is a SurrealDB Datetime object
		// Format: {"Datetime": "2025-..."} or {"Time": "2025-..."}
		if datetime, ok := v["Datetime"]; ok && len(v) == 1 {
			if dtStr, ok := datetime.(string); ok {
				return dtStr
			}
		}
		if timeVal, ok := v["Time"]; ok && len(v) == 1 {
			if dtStr, ok := timeVal.(string); ok {
				return dtStr
			}
		}
		
		// Check if this is a SurrealDB Record ID object
		// Format 1: {"id": "xxx", "tb": "table"} (embedded SurrealDB / Go driver lowercase)
		// Format 2: {"ID": "xxx", "Table": "table"} (external SurrealDB / Go driver uppercase)
		// Handle lowercase format
		if id, hasID := v["id"]; hasID {
			if tb, hasTB := v["tb"]; hasTB && len(v) == 2 {
				if idStr, ok := id.(string); ok {
					if tbStr, ok := tb.(string); ok {
						return tbStr + ":" + idStr
					}
				}
			}
		}
		// Handle uppercase format (external SurrealDB)
		if id, hasID := v["ID"]; hasID {
			if tb, hasTB := v["Table"]; hasTB {
				slog.Debug("NORMALIZE: Found potential Record ID with uppercase", "keys", len(v), "id", id, "table", tb, "map", v)
				if len(v) == 2 {
					if idStr, ok := id.(string); ok {
						if tbStr, ok := tb.(string); ok {
							result := tbStr + ":" + idStr
							slog.Debug("NORMALIZE: Converting Record ID", "from", v, "to", result)
							return result
						} else {
							slog.Warn("NORMALIZE: table is not string", "type", fmt.Sprintf("%T", tb))
						}
					} else {
						slog.Warn("NORMALIZE: id is not string", "type", fmt.Sprintf("%T", id))
					}
				} else {
					slog.Warn("NORMALIZE: Record ID object has unexpected number of keys", "expected", 2, "got", len(v), "keys", v)
				}
			}
		}
		
		// Recursively process all map values
		result := make(map[string]interface{}, len(v))
		for key, val := range v {
			result[key] = normalizeSurrealDBDatetimes(val)
		}
		return result
	default:
		return data
	}
}

// convertToInt safely converts various numeric types to int
func convertToInt(value interface{}) int {
	if value == nil {
		return 0
	}

	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	case uint64:
		return int(v)
	case uint32:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case string:
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return 0
}

// extractRecordID extracts a SurrealDB record ID from various formats
func extractRecordID(id interface{}) string {
	if id == nil {
		return ""
	}

	if str, ok := id.(string); ok {
		return str
	}

	if idMap, ok := id.(map[string]interface{}); ok {
		if table, hasTable := idMap["Table"]; hasTable {
			if tableStr, ok := table.(string); ok {
				if recordID, hasID := idMap["ID"]; hasID {
					if idStr, ok := recordID.(string); ok {
						return tableStr + ":" + idStr
					}
				}
			}
		}
	}

	idStr := fmt.Sprintf("%v", id)

	if strings.HasPrefix(idStr, "{") && strings.Contains(idStr, " ") && strings.HasSuffix(idStr, "}") {
		inner := idStr[1 : len(idStr)-1]
		parts := strings.SplitN(inner, " ", 2)
		if len(parts) == 2 {
			table := parts[0]
			recordID := parts[1]
			return table + ":" + recordID
		}
	}

	return idStr
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	if val, ok := m[key].(float32); ok {
		return float64(val)
	}
	return 0
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if val, ok := m[key].(map[string]interface{}); ok {
		return val
	}
	return make(map[string]interface{})
}

func getTime(m map[string]interface{}, key string) time.Time {
	val, ok := m[key]
	if !ok {
		return time.Time{}
	}

	slog.Debug("getTime", "key", key, "type", reflect.TypeOf(val), "value", val)
	switch v := val.(type) {
	case string:
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return t
		}
	case time.Time:
		return v
	case float64:
		return time.Unix(int64(v), 0)
	case int64:
		return time.Unix(v, 0)
	case map[string]interface{}:
		// Handle SurrealDB Datetime object format: {"Datetime": "2025-..."}
		if datetime, ok := v["Datetime"]; ok {
			if dtStr, ok := datetime.(string); ok {
				if t, err := time.Parse(time.RFC3339, dtStr); err == nil {
					return t
				}
				if t, err := time.Parse(time.RFC3339Nano, dtStr); err == nil {
					return t
				}
			}
		}
	default:
		rv := reflect.ValueOf(val)
		if rv.Kind() == reflect.Struct {
			f := rv.FieldByName("Time")
			if f.IsValid() && f.Type() == reflect.TypeOf(time.Time{}) {
				return f.Interface().(time.Time)
			}
		}
	}
	return time.Time{}
}

func convertEmbeddingToFloat64(embedding []float32) []float64 {
	if embedding == nil {
		embedding = make([]float32, defaultMtreeDim)
	} else if len(embedding) != defaultMtreeDim {
		norm := make([]float32, defaultMtreeDim)
		copy(norm, embedding)
		embedding = norm
	}

	emb64 := make([]float64, len(embedding))
	for i, v := range embedding {
		emb64[i] = float64(v)
	}
	return emb64
}
