package version

// Version and CommitHash are set at build time with -ldflags. Defaults are
// useful for local development.
var (
	CommitHash string = "unknown"
	Version    string = "dev"
)
