package storage

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The RM-9 change feed.
//
// Agent's background sync pulls corporate records with a cursor, so the
// storage layer has to be able to answer "what changed since X" in an order a
// client can resume from. Three properties make that possible, and all three
// are enforced below rather than assumed:
//
//   - Order. Every write draws its `version` from the org-wide sequence added
//     in v14, so (org_id, version) is a strict total order and
//     idx_<prefix>_org_version serves the range scan directly. The id is still
//     used as a tie break because the rows the v13 backfill wrote all carry
//     version 1.
//   - Completeness. Nothing is deleted, only invalidated, so a removal is a
//     row with valid_to set and a fresh version: a tombstone the consumer can
//     mirror. The feed therefore reads with the scope predicate but without
//     the live restriction.
//   - Resumability. The cursor is opaque and carries the version, the
//     timestamp and the id of the last row served, so it is stateless and
//     survives a restart of either side.

// ChangeLister is the pull half of the sync contract.
//
// It is deliberately a standalone interface rather than a member of Storage:
// the wrappers in modules/commercial implement FullStorage by listing every
// method, so folding a new method into Storage would break them in a tree this
// change is not allowed to touch. Consumers assert for it:
//
//	if feed, ok := st.(storage.ChangeLister); ok { ... }
//
// Folding it into FullStorage is a follow-up, once MergedStorage and
// SyncedStorage forward it.
type ChangeLister interface {
	// ListChanges returns the changes visible to the context principal after
	// the given cursor, in a total order, together with the cursor to resume
	// from. An empty scope means every scope the principal can read; a
	// non-empty one must be among them.
	ListChanges(ctx context.Context, scope, since string, limit int) ([]Change, string, error)
}

// Change is one entry of the pull feed. A tombstone carries Deleted true and
// no payload; the row itself still exists, invalidated.
type Change struct {
	ID        string                 `json:"id"`
	Kind      string                 `json:"kind"`
	Scope     string                 `json:"scope"`
	Version   int64                  `json:"version"`
	UpdatedAt time.Time              `json:"updatedAt"`
	Deleted   bool                   `json:"deleted"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}

// changeFeedTables maps each table the feed reports to the kind name a
// consumer sees. The code_* family is deliberately absent: a code index is
// derived from a working tree and is rebuilt rather than synchronised, so
// streaming it would be a large amount of traffic no consumer can use.
var changeFeedTables = []struct {
	Table string
	Kind  string
}{
	{Table: "kv_memories", Kind: "fact"},
	{Table: "vector_memories", Kind: "vector"},
	{Table: "knowledge_base", Kind: "document"},
	{Table: "events", Kind: "event"},
	{Table: "entities", Kind: "entity"},
	{Table: "wrote", Kind: "relationship"},
	{Table: "mentioned_in", Kind: "relationship"},
	{Table: "related_to", Kind: "relationship"},
}

// defaultChangeFeedLimit is the page size used when the caller asks for none.
const defaultChangeFeedLimit = 100

// maxChangeFeedLimit bounds a page. The feed queries one table at a time and
// merges, so the work per call is limit multiplied by the table count.
const maxChangeFeedLimit = 1000

// payloadOmitted are the columns stripped from a change payload. Embeddings
// are hundreds of floats per row and the consumer re-embeds anyway.
var payloadOmitted = map[string]struct{}{
	"embedding": {},
}

// cursor is the decoded form of the opaque string a client passes back.
type cursor struct {
	Version   int64
	UpdatedAt time.Time
	ID        string
}

// encode renders the cursor as the opaque token the client stores. The format
// is an implementation detail and the client must not parse it, which is why
// it is base64 rather than something inviting to read.
func (c cursor) encode() string {
	raw := fmt.Sprintf("1|%d|%d|%s", c.Version, c.UpdatedAt.UTC().UnixNano(), c.ID)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// decodeCursor parses a cursor token. An empty token is the beginning of the
// feed. A malformed token is an error rather than a silent restart: replaying
// the whole feed because a byte was mangled is worse than failing.
func decodeCursor(token string) (cursor, error) {
	if token == "" {
		return cursor{Version: 0, ID: ""}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return cursor{}, fmt.Errorf("invalid cursor: %w", err)
	}
	parts := strings.SplitN(string(raw), "|", 4)
	if len(parts) != 4 || parts[0] != "1" {
		return cursor{}, fmt.Errorf("invalid cursor: unrecognised format")
	}
	version, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return cursor{}, fmt.Errorf("invalid cursor: bad version: %w", err)
	}
	nanos, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return cursor{}, fmt.Errorf("invalid cursor: bad timestamp: %w", err)
	}
	return cursor{Version: version, UpdatedAt: time.Unix(0, nanos).UTC(), ID: parts[3]}, nil
}

// ListChanges implements ChangeLister.
func (s *SurrealDBStorage) ListChanges(ctx context.Context, scope, since string, limit int) ([]Change, string, error) {
	t, err := s.scope(ctx, "")
	if err != nil {
		return nil, "", err
	}

	if limit <= 0 {
		limit = defaultChangeFeedLimit
	}
	if limit > maxChangeFeedLimit {
		limit = maxChangeFeedLimit
	}

	from, err := decodeCursor(since)
	if err != nil {
		return nil, "", err
	}

	// A named scope narrows the feed, it never widens it: a scope the
	// principal cannot read is refused rather than quietly ignored.
	scopes := t.Principal().ReadScopes()
	if scope != "" {
		if !containsString(scopes, scope) {
			return nil, "", fmt.Errorf("%w: scope %s is not readable by this principal", ErrNoPrincipal, scope)
		}
		scopes = []string{scope}
	}

	params := t.params(map[string]interface{}{
		"__since_version": from.Version,
		"__since_id":      from.ID,
		"__feed_limit":    limit,
	})
	params[paramScopes] = scopes

	// The cursor predicate is a lexicographic (version, id) comparison. The
	// id tie break matters because every row the v13 backfill wrote carries
	// version 1, so version alone is not injective over an upgraded datastore.
	cursorPredicate := "(version > $__since_version OR (version = $__since_version AND record::id(id) > $__since_id))"

	var all []Change
	for _, tbl := range changeFeedTables {
		query := fmt.Sprintf(`
			SELECT * FROM %s
			%s
			ORDER BY version ASC, id ASC
			LIMIT $__feed_limit
		`, tbl.Table, t.whereAny(cursorPredicate))

		result, err := s.query(ctx, query, params)
		if err != nil {
			if isMissingTable(err) {
				continue
			}
			return nil, "", fmt.Errorf("failed to read the change feed of %s: %w", tbl.Table, err)
		}
		all = append(all, changesFromResult(result, tbl.Kind)...)
	}

	// Each table was ordered and bounded on its own; the merge restores the
	// single total order across them and the page bound.
	sort.Slice(all, func(i, j int) bool {
		if all[i].Version != all[j].Version {
			return all[i].Version < all[j].Version
		}
		return all[i].ID < all[j].ID
	})
	if len(all) > limit {
		all = all[:limit]
	}

	next := since
	if len(all) > 0 {
		last := all[len(all)-1]
		next = cursor{
			Version:   last.Version,
			UpdatedAt: last.UpdatedAt,
			ID:        recordKey(last.ID),
		}.encode()
	}
	return all, next, nil
}

// changesFromResult converts one table's rows into feed entries.
func changesFromResult(result *[]QueryResult, kind string) []Change {
	if result == nil || len(*result) == 0 {
		return nil
	}
	qr := (*result)[0]
	if qr.Status != "OK" {
		return nil
	}

	out := make([]Change, 0, len(qr.Result))
	for _, row := range qr.Result {
		validTo := getTime(row, "valid_to")
		deleted := !validTo.IsZero()

		change := Change{
			ID:        extractRecordID(row["id"]),
			Kind:      kind,
			Scope:     extractRecordID(row["scope"]),
			Version:   int64(convertToInt(row["version"])),
			UpdatedAt: getTime(row, "updated_at"),
			Deleted:   deleted,
		}
		if !deleted {
			payload := make(map[string]interface{}, len(row))
			for k, v := range row {
				if _, skip := payloadOmitted[k]; skip {
					continue
				}
				payload[k] = v
			}
			change.Payload = payload
		}
		out = append(out, change)
	}
	return out
}

// recordKey returns the id half of a "table:key" record id.
func recordKey(recordID string) string {
	if _, key, ok := strings.Cut(recordID, ":"); ok {
		return key
	}
	return recordID
}

// containsString reports whether the slice holds the value.
func containsString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

// ChangeFeedKinds returns the kind names the feed can emit, in table order. It
// exists so a consumer can validate a kind filter without duplicating the map.
func ChangeFeedKinds() []string {
	seen := make(map[string]struct{}, len(changeFeedTables))
	out := make([]string, 0, len(changeFeedTables))
	for _, tbl := range changeFeedTables {
		if _, dup := seen[tbl.Kind]; dup {
			continue
		}
		seen[tbl.Kind] = struct{}{}
		out = append(out, tbl.Kind)
	}
	return out
}

// ChangeFeedTables returns the tables the feed reads, so a test can assert
// they are all real memory tables rather than trusting the list.
func ChangeFeedTables() []string {
	out := make([]string, 0, len(changeFeedTables))
	for _, tbl := range changeFeedTables {
		out = append(out, tbl.Table)
	}
	return out
}
