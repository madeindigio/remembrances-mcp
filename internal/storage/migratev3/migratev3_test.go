package migratev3

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRunner stands in for the `surreal` binary so the migration can be tested
// on a machine that does not have one.
type fakeRunner struct {
	calls [][]string
	// dump is written to the export target so the counting path has input.
	dump string
	// failOn makes the command whose first argument matches return an error.
	failOn string
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	if len(args) > 0 && args[0] == f.failOn {
		return "boom", errors.New("command failed")
	}
	switch {
	case len(args) > 0 && args[0] == "version":
		return "surreal 3.2.4 for linux on x86_64", nil
	case len(args) > 1 && args[0] == "v2" && args[1] == "export":
		// The last argument is the dump path.
		if err := os.WriteFile(args[len(args)-1], []byte(f.dump), 0o644); err != nil {
			return "", err
		}
		return "exported", nil
	}
	return "ok", nil
}

const sampleDump = `-- ------------------------------
-- TABLE DATA: kv_memories
-- ------------------------------
DEFINE TABLE kv_memories SCHEMAFULL;
INSERT INTO kv_memories [{ id: kv_memories:a, key: 'one' }];
INSERT INTO kv_memories [{ id: kv_memories:b, key: 'two' }];
INSERT INTO vector_memories [{ id: vector_memories:a }];
UPDATE events:x SET content = 'hello';
`

func TestMigrateDryRunLeavesTheTargetUntouched(t *testing.T) {
	dir := t.TempDir()
	runner := &fakeRunner{dump: sampleDump}

	result, err := Migrate(context.Background(), runner, Options{
		Source:   "rocksdb://" + filepath.Join(dir, "old"),
		Target:   "rocksdb://" + filepath.Join(dir, "new"),
		DumpPath: filepath.Join(dir, "dump.surql"),
		DryRun:   true,
	}, io.Discard)
	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}
	if !result.DryRun {
		t.Error("the result must record that it was a dry run")
	}
	if result.TableRows["kv_memories"] != 2 {
		t.Errorf("expected 2 kv_memories records, got %d", result.TableRows["kv_memories"])
	}
	if result.TableRows["events"] != 1 {
		t.Errorf("expected 1 events record, got %d", result.TableRows["events"])
	}
	for _, call := range runner.calls {
		if len(call) > 1 && call[1] == "import" {
			t.Error("a dry run must not import")
		}
	}
}

func TestMigrateExportsThenImports(t *testing.T) {
	dir := t.TempDir()
	runner := &fakeRunner{dump: sampleDump}

	if _, err := Migrate(context.Background(), runner, Options{
		Source:    "rocksdb://" + filepath.Join(dir, "old"),
		Target:    "ws://localhost:8000",
		Namespace: "prod",
		Database:  "mem",
		DumpPath:  filepath.Join(dir, "dump.surql"),
	}, io.Discard); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	var sawExport, sawImport bool
	for _, call := range runner.calls {
		joined := strings.Join(call, " ")
		if strings.Contains(joined, "v2 export --v3") {
			sawExport = true
			// --v3 is what applies the upstream rewrites: MTREE to HNSW,
			// SEARCH ANALYZER to FULLTEXT ANALYZER, the function renames.
			if !strings.Contains(joined, "--namespace prod") || !strings.Contains(joined, "--database mem") {
				t.Errorf("export does not carry the namespace and database: %s", joined)
			}
		}
		if strings.Contains(joined, " import ") {
			sawImport = true
			if !strings.Contains(joined, "ws://localhost:8000") {
				t.Errorf("import does not target the 3.2 endpoint: %s", joined)
			}
		}
	}
	if !sawExport || !sawImport {
		t.Errorf("expected both an export and an import, got export=%v import=%v", sawExport, sawImport)
	}
}

func TestMigrateNeedsASurrealBinary(t *testing.T) {
	runner := &fakeRunner{failOn: "version"}
	_, err := Migrate(context.Background(), runner, Options{
		Source: "rocksdb://old",
		DryRun: true,
	}, io.Discard)
	if !errors.Is(err, ErrBinaryMissing) {
		t.Fatalf("expected ErrBinaryMissing, got %v", err)
	}
	if !strings.Contains(err.Error(), MinBinaryVersion) {
		t.Error("the error must name the minimum binary version")
	}
}

func TestMigrateRefusesWithoutATarget(t *testing.T) {
	runner := &fakeRunner{dump: sampleDump}
	if _, err := Migrate(context.Background(), runner, Options{Source: "rocksdb://old"}, io.Discard); err == nil {
		t.Fatal("expected a missing --target to be refused")
	}
	if _, err := Migrate(context.Background(), runner, Options{}, io.Discard); err == nil {
		t.Fatal("expected a missing --source to be refused")
	}
}

func TestMigrateSurfacesAFailedExport(t *testing.T) {
	runner := &fakeRunner{dump: sampleDump, failOn: "v2"}
	if _, err := Migrate(context.Background(), runner, Options{
		Source: "rocksdb://old",
		Target: "rocksdb://new",
	}, io.Discard); err == nil {
		t.Fatal("expected a failing export to abort the migration")
	}
}

func TestCountDumpIgnoresThePreamble(t *testing.T) {
	tables, statements := CountDump(sampleDump)
	if tables["kv_memories"] != 2 {
		t.Errorf("expected 2 kv_memories records, got %d", tables["kv_memories"])
	}
	if statements == 0 {
		t.Error("expected a non-zero statement count")
	}
	if _, ok := tables["TABLE"]; ok {
		t.Error("comment lines must not be counted as tables")
	}
}
