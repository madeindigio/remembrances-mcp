package version

// Version and CommitHash are set at build time with -ldflags. Defaults are
// useful for local development.
var (
	CommitHash string = "unknown"
	// LibMode describes how native dependencies are shipped/loaded.
	// Values:
	//   - "shared-libs": expects shared libraries next to the binary or on the system
	//   - "purego": uses go:embed + purego loader to extract and dlopen bundled libraries
	LibMode string = "shared-libs"
	// Variant indicates the build variant (e.g. cpu, cuda, cuda-portable, metal).
	// It is optional, but when set it is printed by --version for easier debugging.
	Variant string = "unknown"
	Version    string = "dev"
)
