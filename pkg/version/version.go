package version

// Version and CommitHash are set at build time with -ldflags. Defaults are
// useful for local development.
var (
	CommitHash string = "unknown"
	// Variant indicates the build variant (e.g. cpu, cuda, cuda-portable, metal).
	// It is optional, but when set it is printed by --version for easier debugging.
	Variant string = "unknown"
	Version    string = "dev"
)
