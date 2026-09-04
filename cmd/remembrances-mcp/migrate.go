package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/internal/storage/migratev3"
)

// runSubcommand dispatches the migration subcommands before the main
// configuration is loaded, because they take their own flag sets and must work
// against a datastore the server itself refuses to open.
//
// It returns false when os.Args carries no subcommand, so the caller falls
// through to the normal server start-up.
func runSubcommand(ctx context.Context) (handled bool, exitCode int) {
	if len(os.Args) < 2 {
		return false, 0
	}
	switch os.Args[1] {
	case "migrate-v3":
		return true, runMigrateV3(ctx, os.Args[2:])
	case "migrate":
		return true, runMigrate(ctx, os.Args[2:])
	default:
		return false, 0
	}
}

// runMigrateV3 exports a SurrealDB 2.3 datastore and imports it into 3.2.
func runMigrateV3(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("migrate-v3", flag.ContinueOnError)
	source := fs.String("source", "", "the 2.3 datastore to export (path or endpoint)")
	target := fs.String("target", "", "the 3.2 datastore to import into (path or endpoint)")
	namespace := fs.String("namespace", "test", "SurrealDB namespace")
	database := fs.String("database", "test", "SurrealDB database")
	dump := fs.String("dump", "", "keep the intermediate .surql dump at this path")
	binary := fs.String("surreal-binary", "", "path to a SurrealDB "+migratev3.MinBinaryVersion+"+ binary (default: surreal on PATH)")
	token := fs.String("token", "", "authentication token for a remote endpoint")
	dryRun := fs.Bool("dry-run", false, "export and report the per-table counts without writing the target")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	result, err := migratev3.Migrate(ctx, migratev3.ExecRunner{}, migratev3.Options{
		Source:    *source,
		Target:    *target,
		Namespace: *namespace,
		Database:  *database,
		DumpPath:  *dump,
		Binary:    *binary,
		Token:     *token,
		DryRun:    *dryRun,
	}, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate-v3 failed: %v\n", err)
		return 1
	}
	if result.DryRun {
		fmt.Println("Nothing was written. Re-run without --dry-run to apply.")
	}
	return 0
}

// runMigrate runs the schema migrations against an already 3.x datastore.
//
// v13 is one-way: there is no v13 to v12 path and none will be written, so the
// command refuses to start unless --backup names a writable location or
// --no-backup is passed explicitly. A backup is the rollback.
func runMigrate(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	to := fs.Int("to", storage.TargetSchemaVersion, "target schema version")
	dbPath := fs.String("db-path", "", "embedded datastore path")
	url := fs.String("surrealdb-url", "", "remote SurrealDB URL")
	user := fs.String("surrealdb-user", "root", "remote SurrealDB user")
	pass := fs.String("surrealdb-pass", "root", "remote SurrealDB password")
	namespace := fs.String("surrealdb-namespace", "test", "SurrealDB namespace")
	database := fs.String("surrealdb-database", "test", "SurrealDB database")
	dimension := fs.Int("embedding-dimension", storage.DefaultEmbeddingDimension, "embedding vector width for the schema")
	backup := fs.String("backup", "", "write a backup here before migrating")
	noBackup := fs.Bool("no-backup", false, "migrate without a backup (v13 is one-way)")
	dryRun := fs.Bool("dry-run", false, "report what v13 would do and write nothing")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *to != storage.TargetSchemaVersion {
		fmt.Fprintf(os.Stderr,
			"only --to %d is supported by this binary; downgrades are not offered (v13 is one-way)\n",
			storage.TargetSchemaVersion)
		return 2
	}
	if !*dryRun && *backup == "" && !*noBackup {
		fmt.Fprintln(os.Stderr,
			"migration v13 is one-way: pass --backup <path> to record a rollback point, or --no-backup to proceed without one")
		return 2
	}
	if *dbPath == "" && *url == "" {
		fmt.Fprintln(os.Stderr, "either --db-path or --surrealdb-url must be provided")
		return 2
	}

	storage.SetEmbeddingDimension(*dimension)

	st := storage.NewSurrealDBStorage(&storage.ConnectionConfig{
		URL:             *url,
		Username:        *user,
		Password:        *pass,
		DBPath:          *dbPath,
		Namespace:       *namespace,
		Database:        *database,
		UseEmbeddedLibs: true,
		Timeout:         30 * time.Second,
	})
	if err := st.Connect(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to open the datastore: %v\n", err)
		return 1
	}
	defer func() { _ = st.Close() }()

	if *dryRun {
		report, err := st.V13DryRun(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dry run failed: %v\n", err)
			return 1
		}
		fmt.Printf("legacy principals to create: %d\n", len(report.LegacyUsers))
		for uid, slug := range report.LegacyUsers {
			fmt.Printf("  %-32s -> app_user:legacy_%s\n", uid, slug)
		}
		fmt.Println("rows to backfill per table:")
		for table, n := range report.RowsPerTable {
			fmt.Printf("  %-24s %d\n", table, n)
		}
		fmt.Println("Nothing was written. Re-run without --dry-run to apply.")
		return 0
	}

	if *backup != "" {
		fmt.Printf("Backup requested at %s. Take it with `surreal export` before continuing;\n", *backup)
		fmt.Println("this command does not shell out to the SurrealDB binary for backups.")
	}

	if err := st.InitializeSchema(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "migration failed: %v\n", err)
		return 1
	}
	fmt.Printf("Schema is at version %d.\n", storage.TargetSchemaVersion)
	if *backup != "" {
		fmt.Printf("If you need to roll back, restore the backup at %s: there is no v13 to v12 path.\n", *backup)
	}
	return 0
}
