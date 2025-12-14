//go:build linux && amd64 && embedded && embedded_cpu

package embedded

import "embed"

// platformLibs contains the Linux/amd64 CPU-only shared libraries that are embedded into the binary.
//
//go:embed libs/linux/amd64/cpu/*.so
var platformLibs embed.FS

const (
	platformOS        = "linux"
	platformArch      = "amd64"
	platformVariant   = "cpu"
	platformPortable  = false
	platformLibExt    = ".so"
	platformSupported = true
	platformLibDir    = "libs/linux/amd64/cpu"
)
