//go:build surrealprobe

// Command surrealql-probe checks a SurrealQL statement against the embedded
// SurrealDB engine actually linked into this tree and prints OK or ERR per
// statement. It is a diagnostic, not a shipped command: it exists so that a
// change to the schema can be validated against the real 3.x parser and
// executor before it is written into a migration.
//
// It is behind the `surrealprobe` build tag so it stays out of release
// artefacts and out of the default build. Run and vet it with:
//
//	go run  -tags surrealprobe ./cmd/surrealql-probe
//	go vet  -tags surrealprobe ./cmd/surrealql-probe
//
// The probe runs against an in-memory datastore, so it never touches a real
// database. It needs the prebuilt embedded library at runtime
// (internal/embedded/libs/<os>/<arch>/cpu/libsurrealdb_embedded_rs.so); without
// it the process fails to open the engine and reports so.
//
// The clauses below are the ones RM-7 and RM-1 depend on and the ones the 2.x
// to 3.x migration guide changed. Keeping them here means the next person to
// touch the schema can re-establish the ground truth in one command instead of
// reading release notes.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/madeindigio/remembrances-mcp/internal/storage/migrations"
	"github.com/madeindigio/remembrances-mcp/internal/surrealembedded"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "surrealql-probe: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	db, err := surrealembedded.NewFromURL(ctx, "memory")
	if err != nil {
		return fmt.Errorf("failed to open the embedded engine: %w", err)
	}
	if err := db.Use("probe", "probe"); err != nil {
		return fmt.Errorf("failed to select namespace and database: %w", err)
	}

	fmt.Println("== dialect clauses ==")
	failed := 0
	for _, c := range dialectStatements() {
		got := probe(db, []string{c.Statement})
		if (got > 0) != c.WantErr {
			failed++
			fmt.Printf("     ^^ UNEXPECTED: wanted error=%v\n", c.WantErr)
		}
	}

	fmt.Println()
	fmt.Println("== the migration statements this tree emits ==")
	for version := 1; version <= migrations.TargetVersion; version++ {
		stmts := migrations.StatementStrings(version, migrations.EmbeddingDimension())
		fmt.Printf("-- v%d (%d statements)\n", version, len(stmts))
		failed += probe(db, stmts)
	}

	fmt.Printf("\n%d statements were rejected by the engine.\n", failed)
	if failed > 0 {
		os.Exit(1)
	}
	return nil
}

// probe executes each statement and reports the outcome, returning how many
// were rejected.
func probe(db *surrealembedded.DB, statements []string) int {
	failed := 0
	for _, stmt := range statements {
		r, err := db.Query(stmt, nil)
		if err != nil {
			failed++
			fmt.Printf("ERR  %-78s -> %s\n", truncate(stmt, 78), firstLine(err.Error()))
			continue
		}
		fmt.Printf("OK   %-78s -> %s\n", truncate(stmt, 78), truncate(fmt.Sprint(r), 60))
	}
	return failed
}

// Clause is a probe statement and the outcome 3.x is expected to produce.
// A statement whose outcome flips is what this command exists to catch.
type Clause struct {
	Statement string
	// WantErr marks the clauses 3.x is expected to reject: the 2.x spellings
	// this code base moved away from, and the strictness checks.
	WantErr bool
}

// dialectStatements are the individual clauses whose 3.x spelling this code base
// depends on. Verified against the embedded SurrealDB 3.2.4 build.
func dialectStatements() []Clause {
	return []Clause{
		// FLEXIBLE is a postfix modifier in 3.x. The prefix form is a parse
		// error: "FLEXIBLE must be specified after TYPE".
		{Statement: `DEFINE TABLE probe_v SCHEMAFULL`, WantErr: false},
		{Statement: `DEFINE FIELD md ON probe_v TYPE object FLEXIBLE DEFAULT {}`, WantErr: false},
		{Statement: `DEFINE FIELD md_prefix ON probe_v FLEXIBLE TYPE option<object>`, WantErr: true},

		// array<float, N> means exactly N elements in 3.x, not at most N.
		{Statement: `DEFINE FIELD e ON probe_v TYPE option<array<float, 4>>`, WantErr: false},

		// SCHEMAFULL rejects unknown fields rather than dropping them.
		{Statement: `CREATE probe_v CONTENT { md: { a: 1 }, unknown_field: 3 }`, WantErr: true},

		// MTREE is removed; HNSW replaces it. TYPE precedes DIST in the grammar.
		{Statement: `DEFINE TABLE probe_vec SCHEMAFULL`, WantErr: false},
		{Statement: `DEFINE FIELD emb ON probe_vec TYPE array<float, 4>`, WantErr: false},
		{Statement: `DEFINE INDEX probe_mtree ON probe_vec FIELDS emb MTREE DIMENSION 4 DIST COSINE`, WantErr: true},
		{Statement: `DEFINE INDEX probe_hnsw ON probe_vec FIELDS emb HNSW DIMENSION 4 TYPE F32 DIST COSINE EFC 150 M 12`, WantErr: false},
		{Statement: `DEFINE INDEX IF NOT EXISTS probe_hnsw_conc ON probe_vec FIELDS emb HNSW DIMENSION 4 TYPE F32 DIST COSINE EFC 150 M 12 CONCURRENTLY`, WantErr: false},
		{Statement: `DEFINE INDEX probe_defer ON probe_vec FIELDS emb HNSW DIMENSION 4 DIST COSINE DEFER`, WantErr: true},
		{Statement: `CREATE probe_vec SET emb = [0.1, 0.2, 0.3, 0.4]`, WantErr: false},
		{Statement: `SELECT id FROM probe_vec WHERE emb <|2,COSINE|> [0.1, 0.2, 0.3, 0.4]`, WantErr: false},

		// SEARCH ANALYZER is removed; FULLTEXT ANALYZER replaces it.
		{Statement: `DEFINE ANALYZER OVERWRITE probe_analyzer TOKENIZERS blank, class FILTERS lowercase, snowball(english)`, WantErr: false},
		{Statement: `DEFINE TABLE probe_txt SCHEMAFULL`, WantErr: false},
		{Statement: `DEFINE FIELD content ON probe_txt TYPE string`, WantErr: false},
		{Statement: `DEFINE INDEX probe_search ON probe_txt FIELDS content SEARCH ANALYZER probe_analyzer BM25`, WantErr: true},
		{Statement: `DEFINE INDEX probe_fulltext ON probe_txt FIELDS content FULLTEXT ANALYZER probe_analyzer BM25 HIGHLIGHTS`, WantErr: false},

		// OVERWRITE and IF NOT EXISTS, which replace the INFO FOR DB probe.
		{Statement: `DEFINE FIELD OVERWRITE content ON probe_txt TYPE option<string>`, WantErr: false},
		{Statement: `REMOVE INDEX IF EXISTS probe_absent ON probe_txt`, WantErr: false},

		// RELATION tables with ENFORCED, used by member_of.
		{Statement: `DEFINE TABLE probe_a SCHEMAFULL`, WantErr: false},
		{Statement: `DEFINE TABLE probe_b SCHEMAFULL`, WantErr: false},
		{Statement: `DEFINE TABLE probe_rel TYPE RELATION IN probe_a OUT probe_b ENFORCED SCHEMAFULL`, WantErr: false},

		// DEFAULT ... READONLY, ASSERT with a closure, option<record<t>>.
		{Statement: `DEFINE TABLE probe_field SCHEMAFULL`, WantErr: false},
		{Statement: `DEFINE FIELD created_at ON probe_field TYPE datetime DEFAULT time::now() READONLY`, WantErr: false},
		{Statement: `DEFINE FIELD perms ON probe_field TYPE array<string> ASSERT $value.all(|$p| $p IN ['read', 'write'])`, WantErr: false},
		{Statement: `DEFINE FIELD owner ON probe_field TYPE option<record<probe_a>>`, WantErr: false},

		// LET is mandatory for parameter declarations in a multi-statement
		// query, and transactions wrap a batch.
		{Statement: "BEGIN TRANSACTION;\nLET $batch = (SELECT VALUE id FROM probe_vec LIMIT 10);\nUPDATE $batch SET emb = [0.1, 0.2, 0.3, 0.4];\nCOMMIT TRANSACTION;", WantErr: false},

		// Functions the v13 backfill uses.
		{Statement: `RETURN [crypto::sha256(string::trim('  hello  '))]`, WantErr: false},
		{Statement: `RETURN [type::record('probe_a:one')]`, WantErr: false},

		// A backtick-escaped table name, used for `grant`.
		{Statement: "DEFINE TABLE IF NOT EXISTS `grant` SCHEMAFULL", WantErr: false},
		{Statement: "DEFINE FIELD OVERWRITE principal ON `grant` TYPE string", WantErr: false},
	}
}

func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > n {
		return s[:n-3] + "..."
	}
	return s
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
