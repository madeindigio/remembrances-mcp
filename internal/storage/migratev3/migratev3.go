// Package migratev3 implements `remembrances-mcp migrate-v3`: the one supported
// path from a SurrealDB 2.3 datastore to a 3.2 one.
//
// SurrealDB 3.x refuses to open a 2.x datastore (Datastore::check_version bails
// with OutdatedStorageVersion) and there is no in-place upgrade in the 3.2 line;
// the automatic migration ledger documented for 3.3.0 only covers 3.x to 3.x.
// The supported path is export/import driven by a 3.x binary:
//
//	surreal v2 export --v3 ... dump.surql     # 3.0.3 or newer
//	surreal import  ...       dump.surql
//
// --v3 applies the upstream rewrites: MTREE to HNSW, SEARCH ANALYZER to FULLTEXT
// ANALYZER, the function renames, LET insertion, <future> to COMPUTED.
//
// Decision 001 chose shelling out to the upstream binary over streaming through
// the embedded wrapper, because the alternative means linking two incompatible
// major versions of the surrealdb crate into one binary.
package migratev3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MinBinaryVersion is the oldest `surreal` release that carries `v2 export --v3`.
const MinBinaryVersion = "3.0.3"

// Options configures a migration run.
type Options struct {
	// Source is the 2.3 datastore, as a path or a SurrealDB connection string.
	Source string
	// Target is the 3.2 datastore the dump is imported into.
	Target string
	// Namespace and Database select what is exported and where it lands.
	Namespace string
	Database  string
	// DumpPath is where the intermediate .surql dump is written. Empty means a
	// temporary file that is removed on success.
	DumpPath string
	// Binary is the `surreal` 3.x executable. Empty means "surreal" on PATH.
	Binary string
	// Token authenticates against a remote endpoint, when one is used.
	Token string
	// DryRun validates the preconditions and prints the plan without writing.
	DryRun bool
}

// Runner executes external commands. It exists so the migration can be unit
// tested without a `surreal` binary on the machine.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (stdout string, err error)
}

// ExecRunner is the production Runner: os/exec.
type ExecRunner struct{}

// Run executes a command and returns its combined output.
func (ExecRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Result reports what a run did, per table, so the operator can compare both
// sides rather than trusting that the import succeeded.
type Result struct {
	DumpPath   string
	Statements int
	// TableRows counts the records the dump carries per table. It is derived
	// from the dump itself, which is the only artefact both sides share.
	TableRows map[string]int
	DryRun    bool
}

// ErrBinaryMissing is returned when no usable `surreal` 3.x binary was found.
var ErrBinaryMissing = errors.New(
	"no `surreal` binary found: migrate-v3 needs SurrealDB " + MinBinaryVersion +
		" or newer on PATH, because only a 3.x binary can read a 2.x datastore")

// Migrate runs the export/import pair. With DryRun it stops after the export and
// the verification counts, leaving the target untouched.
func Migrate(ctx context.Context, runner Runner, opts Options, out io.Writer) (*Result, error) {
	if opts.Source == "" {
		return nil, errors.New("migrate-v3 needs a --source datastore")
	}
	if !opts.DryRun && opts.Target == "" {
		return nil, errors.New("migrate-v3 needs a --target datastore, or --dry-run")
	}
	if opts.Namespace == "" {
		opts.Namespace = "test"
	}
	if opts.Database == "" {
		opts.Database = "test"
	}

	binary := opts.Binary
	if binary == "" {
		binary = "surreal"
	}
	version, err := runner.Run(ctx, binary, "version")
	if err != nil {
		return nil, fmt.Errorf("%w (tried %q: %v)", ErrBinaryMissing, binary, err)
	}
	fmt.Fprintf(out, "using %s: %s\n", binary, strings.TrimSpace(version))

	dumpPath := opts.DumpPath
	cleanup := false
	if dumpPath == "" {
		f, err := os.CreateTemp("", "remembrances-v2-export-*.surql")
		if err != nil {
			return nil, fmt.Errorf("failed to create the dump file: %w", err)
		}
		dumpPath = f.Name()
		_ = f.Close()
		cleanup = true
	} else if err := os.MkdirAll(filepath.Dir(dumpPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create the dump directory: %w", err)
	}

	exportArgs := []string{"v2", "export", "--v3",
		"--namespace", opts.Namespace, "--database", opts.Database}
	if opts.Token != "" {
		exportArgs = append(exportArgs, "--token", opts.Token)
	}
	exportArgs = append(exportArgs, "--endpoint", opts.Source, dumpPath)

	fmt.Fprintf(out, "exporting %s -> %s\n", opts.Source, dumpPath)
	if stdout, err := runner.Run(ctx, binary, exportArgs...); err != nil {
		return nil, fmt.Errorf("export failed: %w\n%s", err, stdout)
	}

	result := &Result{DumpPath: dumpPath, DryRun: opts.DryRun}
	dump, err := os.ReadFile(dumpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read the dump back: %w", err)
	}
	result.TableRows, result.Statements = CountDump(string(dump))

	fmt.Fprintf(out, "dump carries %d statements\n", result.Statements)
	for _, table := range sortedKeys(result.TableRows) {
		fmt.Fprintf(out, "  %-24s %d records\n", table, result.TableRows[table])
	}

	if opts.DryRun {
		fmt.Fprintf(out, "dry run: the target was not written. Dump kept at %s\n", dumpPath)
		return result, nil
	}

	importArgs := []string{"import",
		"--namespace", opts.Namespace, "--database", opts.Database}
	if opts.Token != "" {
		importArgs = append(importArgs, "--token", opts.Token)
	}
	importArgs = append(importArgs, "--endpoint", opts.Target, dumpPath)

	fmt.Fprintf(out, "importing %s -> %s\n", dumpPath, opts.Target)
	if stdout, err := runner.Run(ctx, binary, importArgs...); err != nil {
		return nil, fmt.Errorf("import failed: %w\n%s", err, stdout)
	}

	if cleanup {
		_ = os.Remove(dumpPath)
		result.DumpPath = ""
	}

	fmt.Fprintf(out, "migrate-v3 complete. Start the server against %s; it will run migration v13.\n", opts.Target)
	fmt.Fprintf(out, "The 2.3 datastore at %s is untouched and is your rollback.\n", opts.Source)
	return result, nil
}

// CountDump derives the per-table record count and the statement count from a
// .surql dump. It counts INSERT and CREATE/UPDATE record statements, which is
// what `surreal export` emits for data, and ignores the DEFINE preamble.
func CountDump(dump string) (map[string]int, int) {
	tables := map[string]int{}
	statements := 0

	for _, line := range strings.Split(dump, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		statements++

		upper := strings.ToUpper(trimmed)
		switch {
		case strings.HasPrefix(upper, "INSERT INTO "):
			if name := firstWord(trimmed[len("INSERT INTO "):]); name != "" {
				tables[name]++
			}
		case strings.HasPrefix(upper, "UPDATE "), strings.HasPrefix(upper, "CREATE "):
			// `UPDATE table:id SET ...` and `CREATE table:id ...`.
			rest := trimmed[strings.Index(trimmed, " ")+1:]
			if name, _, found := strings.Cut(firstWord(rest), ":"); found {
				tables[name]++
			}
		}
	}
	return tables, statements
}

func firstWord(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		s = s[:i]
	}
	return strings.Trim(s, "`;")
}

func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
