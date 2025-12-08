//go:build !((linux && amd64 && (embedded || embedded_cpu || embedded_cuda || embedded_cuda_portable)) || (darwin && arm64 && (embedded || embedded_metal)))

package embedded

import (
	"embed"
	"runtime"
)

// platformLibs is intentionally empty on unsupported platforms. This keeps
// compilation working even when we do not ship embedded libraries for a
// platform.
var platformLibs embed.FS

const (
	platformOS        = runtime.GOOS
	platformArch      = runtime.GOARCH
	platformVariant   = ""
	platformPortable  = false
	platformLibExt    = ""
	platformSupported = false
	platformLibDir    = ""
)
