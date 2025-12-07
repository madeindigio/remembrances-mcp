//go:build linux && amd64

package embedded

import "embed"

// platformLibs contains the Linux/amd64 shared libraries that are embedded into the binary.
//
//go:embed libs/linux/amd64/*.so
var platformLibs embed.FS

const (
	platformOS        = "linux"
	platformArch      = "amd64"
	platformSupported = true
	platformLibDir    = "libs/linux/amd64"
)
