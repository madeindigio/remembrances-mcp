//go:build darwin && arm64 && embedded && embedded_metal

package embedded

import "embed"

// platformLibs contains the macOS arm64 Metal shared libraries that are embedded into the binary.
//
//go:embed libs/darwin/arm64/metal/*.dylib
var platformLibs embed.FS

const (
	platformOS        = "darwin"
	platformArch      = "arm64"
	platformVariant   = "metal"
	platformPortable  = false
	platformLibExt    = ".dylib"
	platformSupported = true
	platformLibDir    = "libs/darwin/arm64/metal"
)
