package storage

import "testing"

// TestATombstoneNamesTheClientsOwnRecord pins the RM-9 feed's half of defect 11
// of the Alchemai v1 defect list.
//
// The sync contract promises a puller that a deletion propagates, and a puller
// holding a corporate id it never chose can only act on that promise if the
// tombstone names the row it pushed. A tombstone deliberately ships no payload,
// so recovering the stamped `source.local_id` from the payload always recovered
// nothing; it is recovered from the row instead and carried on the entry.
//
// The assertion is both halves at once: the local id is present, and the
// content of the deleted record still is not.
func TestATombstoneNamesTheClientsOwnRecord(t *testing.T) {
	// The three places a stamped provenance object is held, one per row shape.
	shapes := map[string]map[string]interface{}{
		"source column": {
			"source": map[string]interface{}{"local_id": "L-1"},
		},
		"metadata column": {
			"metadata": map[string]interface{}{"source": map[string]interface{}{"local_id": "L-1"}},
		},
		"nested under value": {
			"value": map[string]interface{}{
				"metadata": map[string]interface{}{"source": map[string]interface{}{"local_id": "L-1"}},
			},
		},
	}
	for name, extra := range shapes {
		t.Run(name, func(t *testing.T) {
			row := map[string]interface{}{
				"id":         "vector_memories:one",
				"scope":      "scope:personal_alice",
				"version":    3,
				"valid_to":   "2026-09-06T00:00:00Z",
				"content":    "the deleted content",
				"updated_at": "2026-09-06T00:00:00Z",
			}
			for k, v := range extra {
				row[k] = v
			}
			result := &[]QueryResult{{Status: "OK", Result: []map[string]interface{}{row}}}

			changes := changesFromResult(result, "vector")
			if len(changes) != 1 {
				t.Fatalf("changes = %+v", changes)
			}
			change := changes[0]
			if !change.Deleted {
				t.Fatalf("a row with valid_to set is a tombstone; got %+v", change)
			}
			if change.LocalID != "L-1" {
				t.Errorf("LocalID = %q, want the client's own id", change.LocalID)
			}
			if change.Payload != nil {
				t.Errorf("a tombstone shipped a payload: %+v", change.Payload)
			}
		})
	}

	t.Run("an unstamped row simply has none", func(t *testing.T) {
		result := &[]QueryResult{{Status: "OK", Result: []map[string]interface{}{
			{"id": "vector_memories:two", "valid_to": "2026-09-06T00:00:00Z"},
		}}}
		changes := changesFromResult(result, "vector")
		if len(changes) != 1 || changes[0].LocalID != "" {
			t.Errorf("changes = %+v, want one entry with no local id", changes)
		}
	})

	t.Run("a live row carries it too", func(t *testing.T) {
		result := &[]QueryResult{{Status: "OK", Result: []map[string]interface{}{
			{"id": "vector_memories:three", "source": map[string]interface{}{"local_id": "L-2"}},
		}}}
		changes := changesFromResult(result, "vector")
		if len(changes) != 1 {
			t.Fatalf("changes = %+v", changes)
		}
		if changes[0].Deleted {
			t.Fatal("a row without valid_to is not a tombstone")
		}
		if changes[0].LocalID != "L-2" {
			t.Errorf("LocalID = %q", changes[0].LocalID)
		}
	})
}
