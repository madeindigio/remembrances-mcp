//go:build surrealprobe

// This file carries the RM-2, RM-3 and RM-9 probes: the parameter binding and
// predicate shapes the tenancy layer emits at runtime, as opposed to the DDL
// the rest of the probe covers.
//
// The question these answer is not "does the statement parse" but "does it
// select the right rows": a record link compared against a plain string
// parameter parses fine and silently matches nothing, which would turn every
// scoped query into an empty result rather than an error. Each case therefore
// asserts an expected row count against real rows.
package main

import (
	"fmt"

	"github.com/madeindigio/remembrances-mcp/internal/surrealembedded"
)

// paramCase is one runtime query shape with its parameters and the number of
// rows it must return.
type paramCase struct {
	Name     string
	Query    string
	Params   map[string]interface{}
	WantRows int
	WantErr  bool
}

// rm239Fixture builds the smallest datastore that can answer the questions:
// two orgs, three scopes, and rows spread across them including one tombstone.
func rm239Fixture() []string {
	return []string{
		`DEFINE TABLE IF NOT EXISTS org SCHEMAFULL`,
		`DEFINE FIELD OVERWRITE name ON org TYPE string`,
		`DEFINE TABLE IF NOT EXISTS mem_scope SCHEMAFULL`,
		`DEFINE FIELD OVERWRITE org ON mem_scope TYPE record<org>`,
		`DEFINE FIELD OVERWRITE label ON mem_scope TYPE string`,
		`DEFINE TABLE IF NOT EXISTS probe_mem SCHEMAFULL`,
		`DEFINE FIELD OVERWRITE org_id ON probe_mem TYPE record<org>`,
		`DEFINE FIELD OVERWRITE scope ON probe_mem TYPE record<mem_scope>`,
		`DEFINE FIELD OVERWRITE valid_to ON probe_mem TYPE option<datetime>`,
		`DEFINE FIELD OVERWRITE version ON probe_mem TYPE int DEFAULT 0`,
		`DEFINE FIELD OVERWRITE updated_at ON probe_mem TYPE datetime DEFAULT time::now()`,
		`DEFINE FIELD OVERWRITE content ON probe_mem TYPE string`,
		`DEFINE INDEX IF NOT EXISTS idx_probe_org_version ON probe_mem FIELDS org_id, version`,
		`DEFINE INDEX IF NOT EXISTS idx_probe_valid ON probe_mem FIELDS org_id, scope, valid_to`,
		`UPSERT org:a SET name = 'a'`,
		`UPSERT org:b SET name = 'b'`,
		`UPSERT mem_scope:a1 SET org = org:a, kind = 'org', label = 'a/1'`,
		`UPSERT mem_scope:a2 SET org = org:a, kind = 'org', label = 'a/2'`,
		`UPSERT mem_scope:b1 SET org = org:b, kind = 'org', label = 'b/1'`,
		`CREATE probe_mem SET org_id = org:a, scope = mem_scope:a1, content = 'a1-live', version = 1`,
		`CREATE probe_mem SET org_id = org:a, scope = mem_scope:a2, content = 'a2-live', version = 2`,
		`CREATE probe_mem SET org_id = org:a, scope = mem_scope:a1, content = 'a1-dead', version = 3, valid_to = time::now()`,
		`CREATE probe_mem SET org_id = org:b, scope = mem_scope:b1, content = 'b1-live', version = 4`,
	}
}

// rm239Cases are the runtime shapes the storage layer emits.
func rm239Cases() []paramCase {
	return []paramCase{
		{
			Name:     "record link compared to a plain string parameter matches nothing",
			Query:    `SELECT content FROM probe_mem WHERE org_id = $org`,
			Params:   map[string]interface{}{"org": "org:a"},
			WantRows: 0,
		},
		{
			Name:     "type::record casts the string parameter into the link",
			Query:    `SELECT content FROM probe_mem WHERE org_id = type::record($org)`,
			Params:   map[string]interface{}{"org": "org:a"},
			WantRows: 3,
		},
		{
			Name:  "scope set membership over cast strings",
			Query: `SELECT content FROM probe_mem WHERE org_id = type::record($org) AND scope IN $scopes.map(|$s| type::record($s))`,
			Params: map[string]interface{}{
				"org":    "org:a",
				"scopes": []string{"mem_scope:a1"},
			},
			WantRows: 2,
		},
		{
			Name:  "live rows only, the idx_valid shape",
			Query: `SELECT content FROM probe_mem WHERE org_id = type::record($org) AND scope IN $scopes.map(|$s| type::record($s)) AND valid_to = NONE`,
			Params: map[string]interface{}{
				"org":    "org:a",
				"scopes": []string{"mem_scope:a1", "mem_scope:a2"},
			},
			WantRows: 2,
		},
		{
			Name:  "an empty visible scope set selects nothing rather than everything",
			Query: `SELECT content FROM probe_mem WHERE org_id = type::record($org) AND scope IN $scopes.map(|$s| type::record($s))`,
			Params: map[string]interface{}{
				"org":    "org:a",
				"scopes": []string{},
			},
			WantRows: 0,
		},
		{
			Name:  "org b never sees org a",
			Query: `SELECT content FROM probe_mem WHERE org_id = type::record($org) AND scope IN $scopes.map(|$s| type::record($s))`,
			Params: map[string]interface{}{
				"org":    "org:b",
				"scopes": []string{"mem_scope:a1", "mem_scope:a2", "mem_scope:b1"},
			},
			WantRows: 1,
		},
		{
			Name:  "the RM-9 change feed range scan, tombstones included",
			Query: `SELECT content, version, updated_at, valid_to FROM probe_mem WHERE org_id = type::record($org) AND version > $since AND scope IN $scopes.map(|$s| type::record($s)) ORDER BY version ASC LIMIT $limit`,
			Params: map[string]interface{}{
				"org":    "org:a",
				"since":  1,
				"scopes": []string{"mem_scope:a1", "mem_scope:a2"},
				"limit":  10,
			},
			WantRows: 2,
		},
		{
			Name:     "a monotonic version bump inside one statement",
			Query:    `UPDATE probe_mem SET version = version + 1, updated_at = time::now() WHERE content = 'a1-live' RETURN version`,
			Params:   nil,
			WantRows: 1,
		},
		{
			Name:     "a soft delete leaves the row and stamps valid_to",
			Query:    `UPDATE probe_mem SET valid_to = time::now(), version = version + 1 WHERE content = 'a2-live' RETURN version`,
			Params:   nil,
			WantRows: 1,
		},
		{
			Name:     "the org-wide max version, the cursor high water mark",
			Query:    `SELECT VALUE math::max(version) FROM probe_mem WHERE org_id = type::record($org) GROUP ALL`,
			Params:   map[string]interface{}{"org": "org:a"},
			WantRows: 1,
		},
		{
			Name:     "DEFINE SEQUENCE exists in 3.x",
			Query:    `DEFINE SEQUENCE IF NOT EXISTS probe_seq BATCH 100 START 1`,
			WantRows: 0,
		},
		{
			Name:     "sequence::nextval",
			Query:    `RETURN [sequence::nextval('probe_seq')]`,
			WantRows: 1,
		},
		{
			Name:     "time::unix nanos as an int",
			Query:    `RETURN [<int> time::nano(time::now())]`,
			WantRows: 1,
		},
		{
			Name:     "record::id of a row for cursor tie breaking",
			Query:    `SELECT VALUE record::id(id) FROM probe_mem LIMIT 1`,
			WantRows: 1,
		},
		{
			Name:     "a write that omits org_id is refused by the tightened schema",
			Query:    `CREATE probe_mem SET content = 'orphan'`,
			Params:   nil,
			WantErr:  true,
			WantRows: 0,
		},
	}
}

// probeParams runs the parameterised cases and returns the number that did not
// behave as documented.
func probeParams(db *surrealembedded.DB, cases []paramCase) int {
	failed := 0
	for _, c := range cases {
		rows, err := db.Query(c.Query, c.Params)
		if err != nil {
			if c.WantErr {
				fmt.Printf("OK   %-60s -> rejected: %s\n", truncate(c.Name, 60), firstLine(err.Error()))
				continue
			}
			failed++
			fmt.Printf("ERR  %-60s -> %s\n", truncate(c.Name, 60), firstLine(err.Error()))
			continue
		}
		if c.WantErr {
			failed++
			fmt.Printf("ERR  %-60s -> expected a rejection, got %d rows\n", truncate(c.Name, 60), len(rows))
			continue
		}
		if len(rows) != c.WantRows {
			failed++
			fmt.Printf("ERR  %-60s -> %d rows, wanted %d\n", truncate(c.Name, 60), len(rows), c.WantRows)
			continue
		}
		fmt.Printf("OK   %-60s -> %d rows\n", truncate(c.Name, 60), len(rows))
	}
	return failed
}
