package storage

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"
)

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

	log.Printf("getTime: key=%s type=%s value=%#v", key, reflect.TypeOf(val), val)
	switch v := val.(type) {
	case string:
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t
		}
	case time.Time:
		return v
	case float64:
		return time.Unix(int64(v), 0)
	case int64:
		return time.Unix(v, 0)
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
