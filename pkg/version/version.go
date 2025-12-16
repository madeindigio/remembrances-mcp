package version

import (
	"fmt"
	"strings"
)

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

// Suffix returns a human-readable suffix describing the build variant and
// library loading mode (if available).
//
// Examples:
//   "cuda purego"
//   "cpu shared-libs"
func Suffix() string {
	var parts []string
	if Variant != "" && Variant != "unknown" {
		parts = append(parts, Variant)
	}
	if LibMode != "" {
		parts = append(parts, LibMode)
	}
	return strings.Join(parts, " ")
}

// Describe returns a single-line version string suitable for --version output.
//
// Format:
//   <Version> (<CommitHash>) <Suffix>
// Where commit hash and suffix are omitted when not available.
func Describe() string {
	base := Version
	if CommitHash != "" && CommitHash != "unknown" {
		base = fmt.Sprintf("%s (%s)", base, CommitHash)
	}
	if suffix := Suffix(); suffix != "" {
		base = base + " " + suffix
	}
	return base
}
