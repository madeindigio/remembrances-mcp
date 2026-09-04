package migrations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sort"
	"strings"
)

// Executor is the minimum a caller must provide to run the data half of v13.
// Both storage paths satisfy it: the remote client and the embedded FFI wrapper
// both expose "run this SurrealQL and give me the rows back", which is why v13
// can be one dialect executed identically by both.
type Executor interface {
	Query(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error)
}

// LegacyOrgID, LegacyUnassignedScopeID and the legacy principal id prefixes are
// the fixed record ids the backfill derives. They are deterministic so a dry run
// and the real run produce the same mapping, and an operator can read the
// mapping before committing to it.
const (
	LegacyOrgID             = "org:legacy"
	LegacyUnassignedScopeID = "mem_scope:legacy_unassigned"
	legacyUserPrefix        = "app_user:legacy_"
	legacyScopePrefix       = "mem_scope:legacy_personal_"
)

// defaultBatchSize bounds each backfill transaction. An unbounded
// UPDATE ... WHERE org_id = NONE over a multi-million row table is a conflict
// magnet on the 3.0 executor.
const defaultBatchSize = 5000

// V13Options configures the data half of v13.
type V13Options struct {
	// DryRun reports what would happen and writes nothing.
	DryRun bool
	// BatchSize bounds each UPDATE pass. Zero means defaultBatchSize.
	BatchSize int
}

// V13Report is what a dry run prints and what a real run returns for logging.
type V13Report struct {
	// LegacyUsers maps each distinct legacy user_id to its derived slug.
	LegacyUsers map[string]string
	// RowsPerTable counts the rows each table pass would touch.
	RowsPerTable map[string]int
	// DuplicateClusters counts the (scope, content_hash) groups that would be
	// deduplicated, per table.
	DuplicateClusters map[string]int
}

// userIDTables are the tables that carry the legacy caller-supplied user_id
// label. Rows there are treated as personal to that label, which is the
// conservative reading: the label was the only ownership signal there was.
var userIDTables = []string{"kv_memories", "vector_memories", "events"}

// contentExpr maps a table to the SurrealQL expression the backfill hashes into
// content_hash. Tables absent from the map keep content_hash at NONE.
var contentExpr = map[string]string{
	"kv_memories":     "<string> value",
	"vector_memories": "content",
	"knowledge_base":  "content",
	"events":          "content",
	"code_chunks":     "content",
}

// validFromExpr maps a table to the column the row's validity starts at. The
// edge tables record their creation time as `timestamp`, everything else as
// `created_at`.
func validFromExpr(table string) string {
	switch table {
	case "wrote", "mentioned_in", "related_to":
		return "timestamp ?? time::now()"
	default:
		return "created_at ?? time::now()"
	}
}

// ContentHash is the Go side of the content_hash column. It must agree exactly
// with the SurrealQL the backfill runs, `crypto::sha256(string::trim(...))`:
// a drift here means the unique dedup index silently stops deduplicating.
func ContentHash(content string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(content)))
	return hex.EncodeToString(sum[:])
}

// ContentHashSurrealExpr renders the SurrealQL expression that must produce the
// same value as ContentHash for the given column expression. Exported so a test
// can assert the two sides agree rather than trusting a comment.
func ContentHashSurrealExpr(columnExpr string) string {
	return fmt.Sprintf("crypto::sha256(string::trim(%s))", columnExpr)
}

// Slug derives the deterministic id fragment for a legacy user_id. Only
// lowercase ASCII letters, digits and underscore survive; every other rune
// becomes an underscore. Two distinct user_id values may therefore slug to the
// same fragment, which PreflightSlugCollisions refuses before anything is
// written.
func Slug(userID string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(userID)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	s := b.String()
	if s == "" {
		s = "unknown"
	}
	return s
}

// SlugCollision names two legacy user_id values that map to the same slug.
type SlugCollision struct {
	Slug    string
	UserIDs []string
}

// PreflightSlugCollisions groups the given legacy user ids by slug and returns
// every group with more than one distinct id. Merging two people's memory is the
// one failure this migration will not resolve automatically, so a non-empty
// result must abort the run and print the pairs.
func PreflightSlugCollisions(userIDs []string) []SlugCollision {
	bySlug := map[string]map[string]struct{}{}
	for _, id := range userIDs {
		s := Slug(id)
		if bySlug[s] == nil {
			bySlug[s] = map[string]struct{}{}
		}
		bySlug[s][id] = struct{}{}
	}

	var out []SlugCollision
	for s, ids := range bySlug {
		if len(ids) < 2 {
			continue
		}
		list := make([]string, 0, len(ids))
		for id := range ids {
			list = append(list, id)
		}
		sort.Strings(list)
		out = append(out, SlugCollision{Slug: s, UserIDs: list})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out
}

// ApplyV13Data runs phases 3 and 4 of v13: the principal backfill, the per-table
// column backfill in bounded batches, the dedup pass and the tightening of the
// columns that become required. It is idempotent: every write is an UPSERT at a
// derived record id and every table pass is guarded by WHERE org_id = NONE, so
// re-running after a failure converges rather than duplicating.
func ApplyV13Data(ctx context.Context, exec Executor, opts V13Options) (*V13Report, error) {
	batch := opts.BatchSize
	if batch <= 0 {
		batch = defaultBatchSize
	}

	report := &V13Report{
		LegacyUsers:       map[string]string{},
		RowsPerTable:      map[string]int{},
		DuplicateClusters: map[string]int{},
	}

	// Phase 3a: discover the legacy principals and refuse colliding slugs.
	userIDs, err := collectLegacyUserIDs(ctx, exec)
	if err != nil {
		return nil, fmt.Errorf("failed to collect legacy user ids: %w", err)
	}
	if collisions := PreflightSlugCollisions(userIDs); len(collisions) > 0 {
		var msg []string
		for _, c := range collisions {
			msg = append(msg, fmt.Sprintf("%q <- %s", c.Slug, strings.Join(c.UserIDs, ", ")))
		}
		return nil, fmt.Errorf(
			"migration v13 refuses to run: %d legacy user_id values collide on their derived slug and would merge two principals' memory: %s",
			len(collisions), strings.Join(msg, "; "))
	}
	for _, id := range userIDs {
		report.LegacyUsers[id] = Slug(id)
	}

	// Row counts, for the dry-run report and for the progress log.
	for _, table := range MemoryTables {
		n, err := countPending(ctx, exec, table)
		if err != nil {
			return nil, fmt.Errorf("failed to count pending rows on %s: %w", table, err)
		}
		report.RowsPerTable[table] = n
	}

	if opts.DryRun {
		slog.Info("Migration v13 dry run", "legacy_users", len(report.LegacyUsers), "tables", len(report.RowsPerTable))
		return report, nil
	}

	// Phase 3b: the principal rows.
	if err := createLegacyPrincipals(ctx, exec, userIDs); err != nil {
		return nil, err
	}

	// Phase 3c: the per-table passes. A table whose rows are all attributed
	// already is skipped outright, so a re-run over a converged datastore issues
	// no UPDATE at all rather than a no-op pass per table.
	for _, table := range MemoryTables {
		if report.RowsPerTable[table] == 0 {
			continue
		}
		if err := backfillTable(ctx, exec, table, userIDs, batch); err != nil {
			return nil, fmt.Errorf("failed to backfill %s: %w", table, err)
		}
	}

	// Phase 4: dedup, then tighten.
	for _, table := range HashUniqueTables {
		clusters, err := dedupTable(ctx, exec, table)
		if err != nil {
			return nil, fmt.Errorf("failed to deduplicate %s: %w", table, err)
		}
		report.DuplicateClusters[table] = clusters
	}
	if err := tightenColumns(ctx, exec); err != nil {
		return nil, err
	}

	return report, nil
}

// collectLegacyUserIDs returns every distinct non-empty legacy user_id across
// the tables that carry one.
func collectLegacyUserIDs(ctx context.Context, exec Executor) ([]string, error) {
	seen := map[string]struct{}{}
	for _, table := range userIDTables {
		rows, err := exec.Query(ctx, fmt.Sprintf(
			"SELECT user_id FROM %s WHERE user_id != NONE AND user_id != '' GROUP BY user_id;", table), nil)
		if err != nil {
			// A datastore that has never held this table is not an error: 3.x
			// reports "the table does not exist" and there is nothing to map.
			if isMissingTableError(err) {
				continue
			}
			return nil, err
		}
		for _, row := range rows {
			if v, ok := row["user_id"].(string); ok && v != "" {
				seen[v] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

// isMissingTableError reports whether the error is 3.x complaining about a table
// that was never defined. In 2.x this case returned an empty result set.
func isMissingTableError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "does not exist") || strings.Contains(msg, "not found")
}

// countPending counts the rows of a table that the backfill has yet to touch.
func countPending(ctx context.Context, exec Executor, table string) (int, error) {
	rows, err := exec.Query(ctx, fmt.Sprintf(
		"SELECT count() AS n FROM %s WHERE org_id = NONE GROUP ALL;", table), nil)
	if err != nil {
		if isMissingTableError(err) {
			return 0, nil
		}
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return toInt(rows[0]["n"]), nil
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	default:
		return 0
	}
}

// createLegacyPrincipals writes the legacy org, the org-wide unassigned scope,
// and one app_user plus one personal mem_scope per distinct legacy user_id.
func createLegacyPrincipals(ctx context.Context, exec Executor, userIDs []string) error {
	stmt := fmt.Sprintf(`
BEGIN TRANSACTION;
UPSERT %s SET name = 'legacy', status = 'active';
UPSERT %s SET org = %s, kind = 'org', label = 'legacy/unassigned';
COMMIT TRANSACTION;`, LegacyOrgID, LegacyUnassignedScopeID, LegacyOrgID)
	if _, err := exec.Query(ctx, stmt, nil); err != nil {
		return fmt.Errorf("failed to create the legacy org and scope: %w", err)
	}

	for _, uid := range userIDs {
		slug := Slug(uid)
		stmt := fmt.Sprintf(`
BEGIN TRANSACTION;
UPSERT %s%s SET org = %s, subject = $uid, display_name = $uid, active = true, is_org_admin = false;
UPSERT %s%s SET org = %s, kind = 'personal', owner_user = %s%s, label = 'legacy/personal/' + $uid;
COMMIT TRANSACTION;`,
			legacyUserPrefix, slug, LegacyOrgID,
			legacyScopePrefix, slug, LegacyOrgID, legacyUserPrefix, slug)
		if _, err := exec.Query(ctx, stmt, map[string]interface{}{"uid": uid}); err != nil {
			return fmt.Errorf("failed to create the legacy principal for %q: %w", uid, err)
		}
	}
	return nil
}

// backfillTable runs the bounded batch passes for one table. Tables that carry a
// legacy user_id are processed per user first, then the rows with no label; the
// rest go straight to the org-wide unassigned scope.
func backfillTable(ctx context.Context, exec Executor, table string, userIDs []string, batch int) error {
	hasUserID := false
	for _, t := range userIDTables {
		if t == table {
			hasUserID = true
			break
		}
	}

	if hasUserID {
		for _, uid := range userIDs {
			slug := Slug(uid)
			assignments := scopeAssignments(table, legacyScopePrefix+slug, legacyUserPrefix+slug, "private")
			where := "org_id = NONE AND user_id = $uid"
			if err := runBatches(ctx, exec, table, where, assignments,
				map[string]interface{}{"uid": uid}, batch); err != nil {
				return err
			}
		}
	}

	// Rows with no attributable principal. Making them org-visible restores
	// exactly the access the legacy instance already gave: these tables either
	// never had a user filter, or the row never carried a label.
	assignments := scopeAssignments(table, LegacyUnassignedScopeID, "NONE", "org")
	if err := runBatches(ctx, exec, table, "org_id = NONE", assignments, nil, batch); err != nil {
		return err
	}
	return nil
}

// scopeAssignments renders the SET clause of a backfill pass.
func scopeAssignments(table, scopeID, createdBy, visibility string) string {
	parts := []string{
		fmt.Sprintf("org_id = %s", LegacyOrgID),
		fmt.Sprintf("scope = %s", scopeID),
		fmt.Sprintf("created_by = %s", createdBy),
		fmt.Sprintf("visibility = '%s'", visibility),
		fmt.Sprintf("valid_from = %s", validFromExpr(table)),
		"updated_at = time::now()",
		"version = 1",
		"source = { origin: 'legacy', instance_id: 'unknown' }",
	}
	if expr, ok := contentExpr[table]; ok {
		parts = append(parts, "content_hash = "+ContentHashSurrealExpr(expr))
	}
	return strings.Join(parts, ", ")
}

// runBatches repeats a bounded UPDATE until no row matches the guard any more.
// Each batch is its own transaction, so a failure halfway leaves a partly
// converged datastore that a re-run finishes.
func runBatches(ctx context.Context, exec Executor, table, where, assignments string,
	params map[string]interface{}, batch int) error {
	for pass := 0; ; pass++ {
		stmt := fmt.Sprintf(`
BEGIN TRANSACTION;
LET $batch = (SELECT VALUE id FROM %s WHERE %s LIMIT %d);
UPDATE $batch SET %s;
COMMIT TRANSACTION;`, table, where, batch, assignments)

		rows, err := exec.Query(ctx, stmt, params)
		if err != nil {
			if isMissingTableError(err) {
				return nil
			}
			return fmt.Errorf("batch %d failed: %w", pass, err)
		}
		if len(rows) == 0 {
			return nil
		}
		slog.Debug("v13 backfill batch applied", "table", table, "rows", len(rows), "pass", pass)
		if len(rows) < batch {
			return nil
		}
	}
}

// dedupTable invalidates duplicate rows inside one scope rather than deleting
// them, which is the same rule the refinement worker follows: the newest row
// stays live, the older ones get a valid_to and a superseded_by. It returns the
// number of duplicate clusters it resolved.
func dedupTable(ctx context.Context, exec Executor, table string) (int, error) {
	rows, err := exec.Query(ctx, fmt.Sprintf(`
SELECT scope, content_hash, count() AS n FROM %s
WHERE org_id != NONE AND valid_to = NONE AND content_hash != NONE
GROUP BY scope, content_hash;`, table), nil)
	if err != nil {
		if isMissingTableError(err) {
			return 0, nil
		}
		return 0, err
	}

	clusters := 0
	for _, row := range rows {
		if toInt(row["n"]) < 2 {
			continue
		}
		hash, _ := row["content_hash"].(string)
		scope := fmt.Sprint(row["scope"])
		if hash == "" || scope == "" {
			continue
		}
		clusters++

		// Keep the newest, invalidate the rest. ORDER BY updated_at makes the
		// choice deterministic instead of relying on record id ordering.
		ids, err := exec.Query(ctx, fmt.Sprintf(`
SELECT VALUE id FROM %s
WHERE org_id != NONE AND valid_to = NONE AND content_hash = $hash AND scope = type::record($scope)
ORDER BY updated_at ASC;`, table), map[string]interface{}{"hash": hash, "scope": scope})
		if err != nil {
			return clusters, err
		}
		if len(ids) < 2 {
			continue
		}
		keep := fmt.Sprint(idOf(ids[len(ids)-1]))
		for _, r := range ids[:len(ids)-1] {
			victim := fmt.Sprint(idOf(r))
			if _, err := exec.Query(ctx, `
UPDATE type::record($id) SET valid_to = time::now(), superseded_by = type::record($keep);`,
				map[string]interface{}{"id": victim, "keep": keep}); err != nil {
				return clusters, err
			}
		}
	}
	return clusters, nil
}

// idOf pulls a record id out of a row that may be either the bare value or a
// map carrying an "id" key, depending on the execution path.
func idOf(row map[string]interface{}) interface{} {
	if v, ok := row["id"]; ok {
		return v
	}
	if v, ok := row["VALUE"]; ok {
		return v
	}
	for _, v := range row {
		return v
	}
	return ""
}

// tightenColumns makes org_id and scope required now that every row has them,
// and builds the UNIQUE dedup index. A collision surviving the dedup pass aborts
// here, before the version bumps, which is the correct outcome: a silently
// skipped uniqueness constraint would let the same fact be stored twice for ever.
func tightenColumns(ctx context.Context, exec Executor) error {
	for _, table := range MemoryTables {
		stmt := fmt.Sprintf(`
DEFINE FIELD OVERWRITE org_id ON %s TYPE record<org>;
DEFINE FIELD OVERWRITE scope ON %s TYPE record<mem_scope>;`, table, table)
		if _, err := exec.Query(ctx, stmt, nil); err != nil {
			if isMissingTableError(err) {
				continue
			}
			return fmt.Errorf("failed to tighten the scope columns on %s: %w", table, err)
		}
	}

	for _, table := range HashUniqueTables {
		p := tableIndexPrefix[table]
		stmt := fmt.Sprintf(`
DEFINE FIELD OVERWRITE content_hash ON %s TYPE string;
DEFINE INDEX IF NOT EXISTS idx_%s_hash ON %s FIELDS org_id, scope, content_hash UNIQUE;`, table, p, table)
		if _, err := exec.Query(ctx, stmt, nil); err != nil {
			if isMissingTableError(err) {
				continue
			}
			return fmt.Errorf("failed to build the content hash index on %s: %w", table, err)
		}
	}
	return nil
}
